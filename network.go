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

	gnatsd "github.com/nats-io/gnatsd/server"
)

// Server represents the Choria network broker server
type Server struct {
	gnatsd      *gnatsd.Server
	opts        *gnatsd.Options
	choria      *choria.Framework
	config      *config.Config
	vzTransport *http.Transport

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
		vzTransport: &http.Transport{
			MaxIdleConns:    1,
			IdleConnTimeout: 5 * time.Second,
		},
	}

	s.opts.Host = c.Config.Choria.NetworkListenAddress
	s.opts.Port = c.Config.Choria.NetworkClientPort
	s.opts.Logtime = false
	s.opts.MaxConn = build.MaxBrokerClients()
	s.opts.WriteDeadline = c.Config.Choria.NetworkWriteDeadline
	s.opts.NoSigs = true

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

	err = s.setupCluster()
	if err != nil {
		return s, fmt.Errorf("Could not setup Clustering: %s", err)
	}

	s.gnatsd = gnatsd.New(s.opts)

	// We always supply true for debug here because in our logger
	// we intercept a few debug logs that really should have been
	// info or warning ones.  This will hopefully be able to go
	// back to normal once nats-io/gnatsd#622 is fixed
	//
	// this does though disable a performance optimisation in the
	// nats logging classes where they don't call debug at all when
	// not needed but I imagine logrus does not have a huge bottleneck
	// in that area so its probably safe
	s.gnatsd.SetLogger(newLogger(), true, false)

	return
}

// Exposes the gnatsd HTTP Handler
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
	s.opts.TLSVerify = true
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
