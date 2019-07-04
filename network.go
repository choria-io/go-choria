package network

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
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
		return s, fmt.Errorf("could not setup clustering: %s", err)
	}

	err = s.setupLeafNodes()
	if err != nil {
		return s, fmt.Errorf("could not setup leafnodes: %s", err)
	}

	err = s.setupGateways()
	if err != nil {
		return s, fmt.Errorf("could not setup gateways: %s", err)
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

func (s *Server) setupGateways() (err error) {
	if s.config.Choria.NetworkGatewayPort == 0 {
		return nil
	}

	if s.config.Choria.NetworkGatewayName == "" {
		return fmt.Errorf("Network Gateways require a name")
	}

	s.log.Infof("Starting Broker Gateway support listening on %s:%d", s.config.Choria.NetworkListenAddress, s.config.Choria.NetworkGatewayPort)

	s.opts.Gateway.Host = s.config.Choria.NetworkListenAddress
	s.opts.Gateway.Port = s.config.Choria.NetworkLeafPort
	s.opts.Gateway.Name = s.config.Choria.NetworkGatewayName
	s.opts.Gateway.RejectUnknown = true

	if s.IsTLS() {
		s.opts.Gateway.TLSConfig = s.opts.TLSConfig
		s.opts.Gateway.TLSTimeout = s.opts.TLSTimeout
	}

	for _, r := range s.config.Choria.NetworkGatewayRemotes {
		root := fmt.Sprintf("plugin.choria.network.gateway_remote.%s", r)
		s.log.Infof("Adding gateway remote %s via %s", r, root)

		remote := &gnatsd.RemoteGatewayOpts{Name: r}

		urlStr := s.config.Option(root+".urls", "")
		if urlStr == "" {
			s.log.Errorf("Gateway %s has no remote url, ignoring", r)
			continue
		}

		urlSrvs, err := srvcache.StringHostsToServers([]string{urlStr}, "nats")
		if err != nil {
			s.log.Errorf("Could not parse URL for gateway remote %s urls '%s': %s", r, urlStr, err)
			continue
		}

		if urlSrvs.Count() == 0 {
			s.log.Errorf("Could not parse URL for gateway remote %s url '%s': needs at least 1 url", r, urlStr)
			continue
		}

		urlU, err := urlSrvs.URLs()
		if err != nil {
			s.log.Errorf("Could not parse URL for gateway remote %s url '%s': %s", r, urlStr, err)
			continue
		}

		remote.URLs = urlU

		if s.IsTLS() {
			remote.TLSConfig = s.opts.Gateway.TLSConfig
			remote.TLSTimeout = s.opts.Gateway.TLSTimeout
		}

		s.opts.Gateway.Gateways = append(s.opts.Gateway.Gateways, remote)
		s.log.Infof("Added remote Gateway %s with servers %s", r, strings.Join(urlSrvs.Strings(), ", "))
	}

	return nil
}

func (s *Server) setupLeafNodes() (err error) {
	if s.config.Choria.NetworkLeafPort == 0 {
		return nil
	}

	s.log.Infof("Starting Broker Leafnode support listening on %s:%d", s.config.Choria.NetworkListenAddress, s.config.Choria.NetworkLeafPort)

	s.opts.LeafNode.Host = s.config.Choria.NetworkListenAddress
	s.opts.LeafNode.Port = s.config.Choria.NetworkLeafPort
	s.opts.LeafNode.NoAdvertise = true

	if s.IsTLS() {
		s.opts.LeafNode.TLSConfig = s.opts.TLSConfig
		s.opts.LeafNode.TLSTimeout = s.opts.TLSTimeout
	}

	for _, r := range s.config.Choria.NetworkLeafRemotes {
		root := fmt.Sprintf("plugin.choria.network.leafnode_remote.%s", r)
		s.log.Infof("Adding leafnode remote %s via %s", r, root)

		remote := &gnatsd.RemoteLeafOpts{
			LocalAccount: s.config.Option(root+".account", ""),
			Credentials:  s.config.Option(root+".credentials", ""),
		}

		urlStr := s.config.Option(root+".url", "")
		if urlStr == "" {
			s.log.Errorf("Leafnode %s has no remote url, ignoring", r)
			continue
		}

		urlSrvs, err := srvcache.StringHostsToServers([]string{urlStr}, "leafnode")
		if err != nil {
			s.log.Errorf("Could not parse URL for leafnode remote %s url '%s': %s", r, urlStr, err)
			continue
		}

		if urlSrvs.Count() != 1 {
			s.log.Errorf("Could not parse URL for leafnode remote %s url '%s': need exactly 1 url", r, urlStr)
			continue
		}

		urlU, err := urlSrvs.URLs()
		if err != nil {
			s.log.Errorf("Could not parse URL for leafnode remote %s url '%s': %s", r, urlStr, err)
			continue
		}

		remote.URL = urlU[0]

		if s.IsTLS() {
			remote.TLS = true
			remote.TLSConfig = s.opts.LeafNode.TLSConfig
			remote.TLSTimeout = s.opts.LeafNode.TLSTimeout
		}

		s.opts.LeafNode.Remotes = append(s.opts.LeafNode.Remotes, remote)
		s.log.Infof("Added remote Leafnode %s with remote %s", r, remote.URL.String())
	}

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

	if s.IsTLS() {
		s.opts.Cluster.TLSConfig = s.opts.TLSConfig
		s.opts.Cluster.TLSConfig.ClientAuth = tls.RequireAndVerifyClientCert
		s.opts.Cluster.TLSConfig.RootCAs = s.opts.TLSConfig.ClientCAs
		s.opts.Cluster.TLSTimeout = s.opts.TLSTimeout
	}

	return
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
