package network

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/choria-io/go-config"
	"github.com/choria-io/go-srvcache"

	gnatsd "github.com/nats-io/nats-server/v2/server"
	logrus "github.com/sirupsen/logrus"
)

// BuildInfoProvider provider build time flag information, example go-choria/build
type BuildInfoProvider interface {
	MaxBrokerClients() int
}

// ChoriaFramework provider access to choria
type ChoriaFramework interface {
	Logger(string) *logrus.Entry
	NetworkBrokerPeers() (srvcache.Servers, error)
	TLSConfig() (*tls.Config, error)
	Configuration() *config.Config
}

type accountStore interface {
	Start(context.Context, *sync.WaitGroup)
	Stop()

	gnatsd.AccountResolver
}

// Server represents the Choria network broker server
type Server struct {
	gnatsd   *gnatsd.Server
	opts     *gnatsd.Options
	choria   ChoriaFramework
	config   *config.Config
	log      *logrus.Entry
	as       accountStore
	operator string

	started bool

	mu *sync.Mutex
}

// NewServer creates a new instance of the Server struct with a fully configured NATS embedded
func NewServer(c ChoriaFramework, bi BuildInfoProvider, debug bool) (s *Server, err error) {
	s = &Server{
		choria:  c,
		config:  c.Configuration(),
		opts:    &gnatsd.Options{},
		log:     c.Logger("network"),
		started: false,
		mu:      &sync.Mutex{},
	}

	s.opts.Host = s.config.Choria.NetworkListenAddress
	s.opts.Port = s.config.Choria.NetworkClientPort
	s.opts.WriteDeadline = s.config.Choria.NetworkWriteDeadline
	s.opts.MaxConn = bi.MaxBrokerClients()
	s.opts.NoSigs = true
	s.opts.Logtime = false

	if debug || s.config.LogLevel == "debug" {
		s.opts.Debug = true
	}

	err = s.setupTLS()
	if err != nil {
		return s, fmt.Errorf("could not setup TLS: %s", err)
	}

	if s.config.Choria.StatsPort > 0 {
		s.opts.HTTPHost = s.config.Choria.StatsListenAddress
		s.opts.HTTPPort = s.config.Choria.StatsPort
	}

	if len(s.config.Choria.NetworkAllowedClientHosts) > 0 {
		s.opts.CustomClientAuthentication = &IPAuth{
			allowList: s.config.Choria.NetworkAllowedClientHosts,
			log:       s.choria.Logger("ipauth"),
		}
	}

	err = s.setupAccounts()
	if err != nil {
		return s, fmt.Errorf("could not set up accounts: %s", err)
	}

	err = s.setupCluster()
	if err != nil {
		s.log.Errorf("Could not setup clustering: %s", err)
	}

	err = s.setupLeafNodes()
	if err != nil {
		s.log.Errorf("Could not setup leafnodes: %s", err)
	}

	err = s.setupGateways()
	if err != nil {
		s.log.Errorf("Could not setup gateways: %s", err)
	}

	s.gnatsd, err = gnatsd.NewServer(s.opts)
	if err != nil {
		return s, fmt.Errorf("could not setup server: %s", err)
	}

	s.gnatsd.SetLogger(newLogger(), s.opts.Debug, false)

	err = s.setSystemAccount()
	if err != nil {
		return s, fmt.Errorf("could not set system account: %s", err)
	}

	return
}

// HTTPHandler Exposes the gnatsd HTTP Handler
func (s *Server) HTTPHandler() http.Handler {
	return s.gnatsd.HTTPHandler()
}

// Start the embedded NATS instance, this is a blocking call until it exits
func (s *Server) Start(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	s.log.Infof("Starting new Network Broker with NATS version %s on %s:%d using config file %s", gnatsd.VERSION, s.opts.Host, s.opts.Port, s.config.ConfigFile)

	if s.as != nil {
		wg.Add(1)
		s.as.Start(ctx, wg)
	}

	go s.gnatsd.Start()

	s.mu.Lock()
	s.started = true
	s.mu.Unlock()

	s.publishStats(ctx, 10*time.Second)

	select {
	case <-ctx.Done():
		s.log.Warn("Choria Network Broker shutting down")
		s.gnatsd.Shutdown()

		if s.as != nil {
			s.as.Stop()
		}
	}

	s.log.Warn("Choria Network Broker shut down")
}

func (s *Server) setupTLS() (err error) {
	if !s.IsTLS() {
		return nil
	}

	s.opts.TLS = true
	s.opts.TLSVerify = !s.config.DisableTLSVerify
	s.opts.TLSTimeout = 2

	tlsc, err := s.choria.TLSConfig()
	if err != nil {
		return err
	}

	s.opts.TLSConfig = tlsc

	return
}

// Started determines if the server have been started
func (s *Server) Started() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.started
}

// IsTLS determines if tls should be enabled
func (s *Server) IsTLS() bool {
	return !s.config.DisableTLS
}

// IsVerifiedTLS determines if tls should be enabled
func (s *Server) IsVerifiedTLS() bool {
	return !s.config.DisableTLSVerify
}
