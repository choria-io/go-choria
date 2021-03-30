package network

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	gnatsd "github.com/nats-io/nats-server/v2/server"
	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/srvcache"
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
	ClientTLSConfig() (*tls.Config, error)
	Configuration() *config.Config
	ValidateSecurity() (errors []string, ok bool)
}

// Server represents the Choria network broker server
type Server struct {
	gnatsd *gnatsd.Server
	opts   *gnatsd.Options
	choria ChoriaFramework
	config *config.Config
	log    *logrus.Entry

	choriaAccount *gnatsd.Account
	systemAccount *gnatsd.Account

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

	if s.config.Identity != "" {
		s.opts.ServerName = s.config.Identity
	}

	s.opts.Host = s.config.Choria.NetworkListenAddress
	s.opts.Port = s.config.Choria.NetworkClientPort
	s.opts.WriteDeadline = s.config.Choria.NetworkWriteDeadline
	s.opts.MaxConn = bi.MaxBrokerClients()
	s.opts.NoSigs = true
	s.opts.Logtime = false
	s.opts.Cluster.Name = s.config.Choria.NetworkGatewayName
	s.opts.ProfPort = s.config.Choria.NetworkProfilePort

	if s.config.Choria.NetworkClientAdvertiseName != "" {
		s.opts.ClientAdvertise = s.config.Choria.NetworkClientAdvertiseName
	} else if s.config.Identity != "" {
		s.opts.ClientAdvertise = fmt.Sprintf("%s:%d", s.config.Identity, s.config.Choria.NetworkClientPort)
	}

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

	err = s.setupCluster()
	if err != nil {
		s.log.Errorf("Could not setup clustering: %s", err)
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

	err = s.setupAccounts()
	if err != nil {
		return s, fmt.Errorf("could not set up accounts: %s", err)
	}

	// This has to happen after accounts to ensure local accounts exist to map things
	err = s.setupLeafNodes()
	if err != nil {
		s.log.Errorf("Could not setup leafnodes: %s", err)
	}

	ipauth := &IPAuth{
		allowList:     s.config.Choria.NetworkAllowedClientHosts,
		log:           s.choria.Logger("ipauth"),
		denyServers:   s.config.Choria.NetworkDenyServers,
		anonTLS:       s.config.Choria.NetworkClientTLSAnon,
		jwtSigner:     s.config.Choria.RemoteSignerSigningCert,
		systemAccount: s.systemAccount,
		choriaAccount: s.choriaAccount,
		systemUser:    s.config.Choria.NetworkSystemUsername,
		systemPass:    s.config.Choria.NetworkSystemPassword,
	}
	s.opts.CustomClientAuthentication = ipauth

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

	go s.gnatsd.Start()

	if !s.gnatsd.ReadyForConnections(time.Minute) {
		s.log.Errorf("broker did not become ready after a minute, terminating")
		return
	}

	s.mu.Lock()
	s.started = true
	s.mu.Unlock()

	go s.publishStats(ctx, 10*time.Second)

	err := s.setupStreaming()
	if err != nil {
		s.log.Errorf("Could not set up Choria Streams: %s", err)
	}

	err = s.configureSystemStreams(ctx)
	if err != nil {
		s.log.Errorf("could not setup system streams: %s", err)
	}

	<-ctx.Done()

	s.log.Warn("Choria Network Broker shutting down")
	s.gnatsd.Shutdown()
	s.log.Warn("Choria Network Broker shut down")
}

func (s *Server) setupTLS() (err error) {
	if !s.config.Choria.NetworkClientTLSForce && !s.IsTLS() {
		s.log.WithField("client_tls_force_required", s.config.Choria.NetworkClientTLSForce).WithField("disable_tls", s.config.DisableTLS).Warn("Skipping broker TLS set up")
		return nil
	}

	// this can be forcing TLS while the framework isn't and so would not have
	// validated the security setup, so we do it again now if force is set
	if s.config.Choria.NetworkClientTLSForce {
		errs, _ := s.choria.ValidateSecurity()
		if len(errs) != 0 {
			return fmt.Errorf("invalid security setup: %s", strings.Join(errs, ", "))
		}
	}

	s.opts.TLS = true
	s.opts.AllowNonTLS = false

	s.opts.TLSTimeout = float64(s.config.Choria.NetworkTLSTimeout)

	tlsc, err := s.choria.TLSConfig()
	if err != nil {
		return err
	}

	s.opts.TLSVerify = true
	tlsc.ClientAuth = tls.RequireAndVerifyClientCert

	if s.config.DisableTLSVerify {
		s.opts.TLSVerify = false
		tlsc.ClientAuth = tls.NoClientCert
	}

	if s.config.Choria.NetworkClientTLSAnon {
		if len(s.config.Choria.NetworkLeafRemotes) == 0 {
			return fmt.Errorf("can only configure anonymous TLS for client connections when leafnodes are defined using plugin.choria.network.leafnode_remotes")
		}

		if !s.config.Choria.NetworkDenyServers {
			s.log.Warnf("Disabling connections from Servers while in Anon TLS mode")
			s.config.Choria.NetworkDenyServers = true
		}

		if len(s.config.Choria.NetworkAllowedClientHosts) == 0 {
			s.log.Warnf("Adding 0.0.0.0/0 to client hosts list, override using plugin.choria.network.client_hosts")
			s.config.Choria.NetworkAllowedClientHosts = []string{"0.0.0.0/0"}
		}

		s.log.Warnf("Configuring anonymous TLS for client connections")
		s.opts.TLSVerify = false
		s.opts.TLS = true
		tlsc.InsecureSkipVerify = true
		tlsc.ClientAuth = tls.NoClientCert
	}

	s.opts.TLSConfig = tlsc

	return
}
