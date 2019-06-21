package network

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/config"
	log "github.com/sirupsen/logrus"

	gnatsd "github.com/nats-io/nats-server/v2/server"
)

// Server represents the Choria network broker server
type Server struct {
	gnatsd *gnatsd.Server
	opts   *gnatsd.Options
	choria *choria.Framework
	config *config.Config

	started bool

	mu *sync.Mutex
}

// NewServer creates a new instance of the Server struct with a fully configured NATS embedded
func NewServer(c *choria.Framework, debug bool) (s *Server, err error) {
	s = &Server{
		choria:  c,
		config:  c.Config,
		opts:    &gnatsd.Options{},
		started: false,
		mu:      &sync.Mutex{},
	}

	s.opts.Host = c.Config.Choria.NetworkListenAddress
	s.opts.Port = c.Config.Choria.NetworkClientPort
	s.opts.WriteDeadline = c.Config.Choria.NetworkWriteDeadline
	s.opts.MaxConn = build.MaxBrokerClients()
	s.opts.NoSigs = true
	s.opts.Logtime = false

	if debug || c.Config.LogLevel == "debug" {
		s.opts.Debug = true
	}

	if !c.Config.DisableTLS {
		err = s.setupTLS()
		if err != nil {
			return s, fmt.Errorf("Could not setup TLS: %s", err)
		}
	}

	if c.Config.Choria.StatsPort > 0 {
		s.opts.HTTPHost = c.Config.Choria.StatsListenAddress
		s.opts.HTTPPort = c.Config.Choria.StatsPort
	}

	if len(c.Config.Choria.NetworkAllowedClientHosts) > 0 {
		s.opts.CustomClientAuthentication = &IPAuth{
			allowList: c.Config.Choria.NetworkAllowedClientHosts,
			log:       s.choria.Logger("ipauth"),
		}
	}

	err = s.setupCluster()
	if err != nil {
		return s, fmt.Errorf("could not setup clustering: %s", err)
	}

	s.gnatsd, err = gnatsd.NewServer(s.opts)
	if err != nil {
		return s, fmt.Errorf("could not setup server: %s", err)
	}

	s.gnatsd.SetLogger(newLogger(), s.opts.Debug, false)

	return
}

// HTTPHandler Exposes the gnatsd HTTP Handler
func (s *Server) HTTPHandler() http.Handler {
	return s.gnatsd.HTTPHandler()
}

// Start the embedded NATS instance, this is a blocking call until it exits
func (s *Server) Start(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Infof("Starting new Network Broker with NATS version %s on %s:%d using config file %s", gnatsd.VERSION, s.opts.Host, s.opts.Port, s.choria.Config.ConfigFile)

	go s.gnatsd.Start()

	s.mu.Lock()
	s.started = true
	s.mu.Unlock()

	s.publishStats(ctx, 10*time.Second)

	select {
	case <-ctx.Done():
		s.gnatsd.Shutdown()
	}

	log.Warn("Choria Network Broker shutting down")
}

func (s *Server) setupCluster() (err error) {
	s.opts.Cluster.Host = s.config.Choria.NetworkListenAddress
	s.opts.Cluster.NoAdvertise = true
	s.opts.Cluster.Port = s.choria.Config.Choria.NetworkPeerPort
	s.opts.Cluster.Username = s.choria.Config.Choria.NetworkPeerUser
	s.opts.Cluster.Password = s.choria.Config.Choria.NetworkPeerPassword

	peers, err := s.choria.NetworkBrokerPeers()
	if err != nil {
		return fmt.Errorf("Could not determine network broker peers: %s", err)
	}

	for _, p := range peers {
		u, err := p.URL()
		if err != nil {
			return fmt.Errorf("Could not parse Peer configuration: %s", err)
		}

		log.Infof("Adding %s as network peer", u.String())
		s.opts.Routes = append(s.opts.Routes, u)
	}

	// Remove any host/ip that points to itself in Route
	newroutes, err := gnatsd.RemoveSelfReference(s.opts.Cluster.Port, s.opts.Routes)
	if err != nil {
		return fmt.Errorf("Could not remove own Self from cluster configuration: %s", err)
	}

	s.opts.Routes = newroutes

	return
}

func (s *Server) setupTLS() (err error) {
	s.opts.TLS = true
	s.opts.TLSVerify = !s.config.DisableTLSVerify
	s.opts.TLSTimeout = 2

	tlsc, err := s.choria.TLSConfig()
	if err != nil {
		return err
	}

	s.opts.TLSConfig = tlsc

	s.opts.Cluster.TLSConfig = s.opts.TLSConfig
	s.opts.Cluster.TLSConfig.ClientAuth = tls.RequireAndVerifyClientCert
	s.opts.Cluster.TLSConfig.RootCAs = tlsc.ClientCAs
	s.opts.Cluster.TLSTimeout = s.opts.TLSTimeout

	return
}

// Started determines if the server have been started
func (s *Server) Started() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.started
}
