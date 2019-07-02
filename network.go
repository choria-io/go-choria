package network

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"path/filepath"
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

	if !s.config.DisableTLS {
		err = s.setupTLS()
		if err != nil {
			return s, fmt.Errorf("could not setup TLS: %s", err)
		}
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

	err = s.setupCluster()
	if err != nil {
		return s, fmt.Errorf("could not setup clustering: %s", err)
	}

	err = s.setupAccounts()
	if err != nil {
		return s, fmt.Errorf("could not set up accounts: %s", err)
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

	wg.Add(1)
	s.as.Start(ctx, wg)

	go s.gnatsd.Start()

	s.mu.Lock()
	s.started = true
	s.mu.Unlock()

	s.publishStats(ctx, 10*time.Second)

	select {
	case <-ctx.Done():
		s.log.Warn("Choria Network Broker shutting down")
		s.gnatsd.Shutdown()
		s.as.Stop()
	}

	s.log.Warn("Choria Network Broker shut down")
}

func (s *Server) setSystemAccount() (err error) {
	if s.config.Choria.NetworkAccountOperator == "" || s.config.Choria.NetworkSystemAccount == "" {
		return nil
	}

	s.log.Infof("Setting the Broker Systems Account to %s and enabling broker events", s.config.Choria.NetworkAccountOperator)
	err = s.gnatsd.SetSystemAccount(s.config.Choria.NetworkSystemAccount)
	if err != nil {
		return err
	}

	return nil
}

func (s *Server) setupAccounts() (err error) {
	if s.config.Choria.NetworkAccountOperator == "" {
		return nil
	}

	s.log.Infof("Starting Broker Account services under operator %s", s.config.Choria.NetworkAccountOperator)

	operatorRoot := filepath.Join(filepath.Dir(s.config.ConfigFile), "accounts", "nats", s.config.Choria.NetworkAccountOperator)
	operatorPath := filepath.Join(operatorRoot, fmt.Sprintf("%s.jwt", s.config.Choria.NetworkAccountOperator))

	opc, err := gnatsd.ReadOperatorJWT(operatorPath)
	if err != nil {
		return fmt.Errorf("could not load operator JWT from %s: %s", operatorPath, err)
	}
	s.opts.TrustedOperators = append(s.opts.TrustedOperators, opc)

	s.as, err = newDirAccountStore(s.gnatsd, operatorRoot)
	if err != nil {
		return fmt.Errorf("could not start account store: %s", err)
	}

	s.opts.AccountResolver = s.as

	return nil
}

func (s *Server) setupCluster() (err error) {
	s.opts.Cluster.Host = s.config.Choria.NetworkListenAddress
	s.opts.Cluster.NoAdvertise = true
	s.opts.Cluster.Port = s.config.Choria.NetworkPeerPort
	s.opts.Cluster.Username = s.config.Choria.NetworkPeerUser
	s.opts.Cluster.Password = s.config.Choria.NetworkPeerPassword

	peers, err := s.choria.NetworkBrokerPeers()
	if err != nil {
		return fmt.Errorf("could not determine network broker peers: %s", err)
	}

	for _, p := range peers.Servers() {
		u, err := p.URL()
		if err != nil {
			return fmt.Errorf("Could not parse Peer configuration: %s", err)
		}

		s.log.Infof("Adding %s as network peer", u.String())
		s.opts.Routes = append(s.opts.Routes, u)
	}

	// Remove any host/ip that points to itself in Route
	newroutes, err := gnatsd.RemoveSelfReference(s.opts.Cluster.Port, s.opts.Routes)
	if err != nil {
		return fmt.Errorf("could not remove own Self from cluster configuration: %s", err)
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
