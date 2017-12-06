package network

import (
	"context"
	"fmt"
	"sync"

	"github.com/choria-io/go-choria/choria"
	log "github.com/sirupsen/logrus"

	gnatsd "github.com/nats-io/gnatsd/server"
)

// Server represents the Choria network broker server
type Server struct {
	gnatsd *gnatsd.Server
	opts   *gnatsd.Options
	choria *choria.Framework
	config *choria.Config
}

// NewServer creates a new instance of the Server struct with a fully configured NATS embedded
func NewServer(c *choria.Framework, debug bool) (s *Server, err error) {
	s = &Server{
		choria: c,
		config: c.Config,
		opts:   &gnatsd.Options{},
	}

	s.opts.Host = c.Config.Choria.NetworkListenAddress
	s.opts.Port = c.Config.Choria.NetworkClientPort
	s.opts.Logtime = false

	if debug || c.Config.LogLevel == "debug" {
		s.opts.Debug = true
	}

	if !c.Config.DisableTLS {
		err = s.setupTLS()
		if err != nil {
			return s, fmt.Errorf("Could not setup TLS: %s", err.Error())
		}
	}

	if c.Config.Choria.NetworkMonitorPort > 0 {
		s.opts.HTTPHost = c.Config.Choria.NetworkListenAddress
		s.opts.HTTPPort = c.Config.Choria.NetworkMonitorPort
	}

	err = s.setupCluster()
	if err != nil {
		return s, fmt.Errorf("Could not setup Clustering: %s", err.Error())
	}

	s.gnatsd = gnatsd.New(s.opts)
	s.gnatsd.SetLogger(newLogger(), s.opts.Debug, false)

	return
}

// Start the embedded NATS instance, this is a blocking call until it exits
func (s *Server) Start(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Infof("Starting new Network Broker with NATS version %s on %s:%d using config file %s", gnatsd.VERSION, s.opts.Host, s.opts.Port, s.choria.Config.ConfigFile)

	s.gnatsd.Start()

	log.Warn("Choria Network Broker has been shut down")
}

func (s *Server) setupCluster() (err error) {
	s.opts.Cluster.Host = s.config.Choria.NetworkListenAddress
	s.opts.Cluster.NoAdvertise = true
	s.opts.Cluster.Port = s.choria.Config.Choria.NetworkPeerPort
	s.opts.Cluster.Username = s.choria.Config.Choria.NetworkPeerUser
	s.opts.Cluster.Password = s.choria.Config.Choria.NetworkPeerPassword

	peers, err := s.choria.NetworkBrokerPeers()
	if err != nil {
		return fmt.Errorf("Could not determine network broker peers: %s", err.Error())
	}

	for _, p := range peers {
		u, err := p.URL()
		if err != nil {
			return fmt.Errorf("Could not parse Peer configuration: %s", err.Error())
		}

		log.Infof("Adding %s as network peer", u.String())
		s.opts.Routes = append(s.opts.Routes, u)
	}

	// Remove any host/ip that points to itself in Route
	newroutes, err := gnatsd.RemoveSelfReference(s.opts.Cluster.Port, s.opts.Routes)
	if err != nil {
		return fmt.Errorf("Could not remove own Self from cluster configuration: %s", err.Error())
	}

	s.opts.Routes = newroutes

	return
}

func (s *Server) setupTLS() (err error) {
	s.opts.TLS = true
	s.opts.TLSVerify = true

	// seems weird to set all this when the thing that it cares for is TlsConfig
	// but that's what gnatsd main also does, so sticking with that pattern
	if p, err := s.choria.CAPath(); err == nil {
		s.opts.TLSCaCert = p
	} else {
		return fmt.Errorf("Could not set the CA: %s", err.Error())
	}

	if p, err := s.choria.ClientPublicCert(); err == nil {
		s.opts.TLSCert = p
	} else {
		return fmt.Errorf("Could not set the Public Cert: %s", err.Error())
	}

	if p, err := s.choria.ClientPrivateKey(); err == nil {
		s.opts.TLSKey = p
	}

	tc := gnatsd.TLSConfigOpts{}
	tc.CaFile = s.opts.TLSCaCert
	tc.CertFile = s.opts.TLSCert
	tc.KeyFile = s.opts.TLSKey
	tc.Verify = true
	tc.Timeout = 2

	if s.opts.TLSConfig, err = gnatsd.GenTLSConfig(&tc); err != nil {
		return
	}

	s.opts.Cluster.TLSConfig = s.opts.TLSConfig

	return
}
