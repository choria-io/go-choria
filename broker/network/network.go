// Copyright (c) 2017-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package network

import (
	"context"
	"crypto/ed25519"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/choria-io/go-choria/inter"
	natsd "github.com/nats-io/nats-server/v2/server"
	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/config"
)

// BuildInfoProvider provider build time flag information, example go-choria/build
type BuildInfoProvider interface {
	MaxBrokerClients() int
}

// Server represents the Choria network broker server
type Server struct {
	gnatsd *natsd.Server
	opts   *natsd.Options
	choria inter.Framework
	config *config.Config
	log    *logrus.Entry

	choriaAccount       *natsd.Account
	systemAccount       *natsd.Account
	provisioningAccount *natsd.Account

	started bool

	mu *sync.Mutex
}

// NewServer creates a new instance of the Server struct with a fully configured NATS embedded
func NewServer(c inter.Framework, bi BuildInfoProvider, debug bool) (s *Server, err error) {
	s = &Server{
		choria:  c,
		config:  c.Configuration(),
		opts:    &natsd.Options{},
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
		// s.opts.Trace = true
		// s.opts.TraceVerbose = true
		// s.opts.Logtime = true
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

	// Initial configuration to enable JetStream but accounts will come later
	if s.config.Choria.NetworkStreamStore != "" {
		s.log.Infof("Configuring Choria Streams in %v", s.config.Choria.NetworkStreamStore)
		s.opts.JetStream = true
		s.opts.StoreDir = s.config.Choria.NetworkStreamStore
	}

	s.gnatsd, err = natsd.NewServer(s.opts)
	if err != nil {
		return s, fmt.Errorf("could not setup server: %s", err)
	}
	s.gnatsd.SetLogger(NewLogger(s.choria.Logger("network_broker")), s.opts.Debug, false)

	err = s.setupAccounts()
	if err != nil {
		return s, fmt.Errorf("could not set up accounts: %s", err)
	}

	err = s.setupMappings()
	if err != nil {
		s.log.Errorf("Network Mapping setup failed: %v", err)
	}

	err = s.setupWebSockets()
	if err != nil {
		return s, fmt.Errorf("could not set up WebSocket: %s", err)
	}

	// This has to happen after accounts to ensure local accounts exist to map things
	err = s.setupLeafNodes()
	if err != nil {
		s.log.Errorf("Could not setup leafnodes: %s", err)
	}

	choriaAuth := &ChoriaAuth{
		clientAllowList: s.config.Choria.NetworkAllowedClientHosts,
		choriaAccount:   s.choriaAccount,
		denyServers:     s.config.Choria.NetworkDenyServers,
		isTLS:           s.isClientTlSBroker(),
		log:             s.choria.Logger("authentication"),
		systemAccount:   s.systemAccount,
		systemPass:      s.config.Choria.NetworkSystemPassword,
		systemUser:      s.config.Choria.NetworkSystemUsername,
		tokenCache:      make(map[string]ed25519.PublicKey),
		issuerTokens:    make(map[string]string),
	}

	if len(s.config.Choria.IssuerNames) > 0 {
		if len(s.config.Choria.NetworkClientTokenSigners) > 0 {
			return nil, fmt.Errorf("cannot set trusted client signers using plugin.choria.network.client_signer_cert when an issuer is configured")
		}
		if len(s.config.Choria.NetworkServerTokenSigners) > 0 {
			return nil, fmt.Errorf("cannot set trusted server signers using plugin.choria.network.server_signer_cert when an issuer is configured")
		}
		if len(s.config.Choria.ChoriaSecurityTrustedSigners) > 0 {
			return nil, fmt.Errorf("cannot set trusted client signers using plugin.security.choria.trusted_signers when an issuer is configured")
		}

		for _, issuer := range s.config.Choria.IssuerNames {
			name := fmt.Sprintf("plugin.security.issuer.%s.public", issuer)
			pk := s.config.Option(name, "")
			if pk == "" {
				s.log.Errorf("Could not find option %s while adding issuer %s", name, issuer)
				continue
			}

			s.log.Warnf("Loaded Organization Issuer %s with public key %s", issuer, pk)
			choriaAuth.issuerTokens[issuer] = pk
		}

		if s.config.Choria.NetworkProvisioningClientPassword != "" {
			s.log.Warnf("Allowing Provisioner connections subject to JWT claims")
		}
	}

	if choriaAuth.isTLS {
		// provisioning happens over clear, so we can't have clear clients and clear provisioning
		choriaAuth.provPass = s.config.Choria.NetworkProvisioningClientPassword
		choriaAuth.provisioningAccount = s.provisioningAccount
		choriaAuth.provisioningTokenSigner = s.config.Choria.NetworkProvisioningTokenSignerFile
		choriaAuth.clientJwtSigners = s.config.Choria.NetworkClientTokenSigners
		choriaAuth.serverJwtSigners = s.config.Choria.NetworkServerTokenSigners
	}

	if len(choriaAuth.clientJwtSigners) > 0 || len(choriaAuth.serverJwtSigners) > 0 || len(choriaAuth.issuerTokens) > 0 {
		s.opts.AlwaysEnableNonce = true
	}

	s.opts.CustomClientAuthentication = choriaAuth

	return
}

// HTTPHandler Exposes the natsd HTTP Handler
func (s *Server) HTTPHandler() http.Handler {
	return s.gnatsd.HTTPHandler()
}

// InProcessConn implements nats.InProcessConnProvider
func (s *Server) InProcessConn() (net.Conn, error) {
	return s.gnatsd.InProcessConn()
}

// Start the embedded NATS instance, this is a blocking call until it exits
func (s *Server) Start(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	s.log.Infof("Starting new Network Broker with NATS version %s on %s:%d using config file %s", natsd.VERSION, s.opts.Host, s.opts.Port, s.config.ConfigFile)

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
	if s.gnatsd.JetStreamEnabled() {
		s.log.Warnf("Disabling Choria Streams")
		s.gnatsd.DisableJetStream()
	}
	s.gnatsd.Shutdown()
	s.gnatsd.WaitForShutdown()
	s.log.Warn("Choria Network Broker shut down")
}

func (s *Server) isClientTlSBroker() bool {
	return s.config.Choria.NetworkClientTLSForce || s.IsTLS()
}

func (s *Server) setupTLS() (err error) {
	if !s.isClientTlSBroker() {
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
	s.opts.TLSVerify = true
	s.opts.TLSTimeout = float64(s.config.Choria.NetworkTLSTimeout)

	tlsc, err := s.choria.TLSConfig()
	if err != nil {
		return err
	}

	if tlsc.ClientCAs == nil {
		// if we have no CA configured we cant verify clients
		tlsc.ClientAuth = tls.NoClientCert
	} else {
		// the auth layer does various checks for tls as we have a varied tls/non tls setup to handle pipe connections
		// and provisioning connections etc, so this is safe.
		tlsc.ClientAuth = tls.VerifyClientCertIfGiven
	}

	switch {
	case s.config.DisableTLSVerify:
		s.log.Warnf("Disabling client certificate verification due to configuration or CLI override")
		s.opts.TLSVerify = false
		tlsc.ClientAuth = tls.NoClientCert

	case s.config.Choria.NetworkProvisioningTokenSignerFile != "", len(s.config.Choria.NetworkClientTokenSigners) > 0, len(s.config.Choria.IssuerNames) > 0:
		if s.config.Choria.NetworkProvisioningTokenSignerFile != "" {
			s.log.Warnf("Allowing unverified TLS connections for provisioning purposes")
		}

		if len(s.config.Choria.IssuerNames) > 0 {
			s.log.Warnf("Allowing unverified TLS connections for Organization Issuer issued connections")
		}

		if len(s.config.Choria.NetworkClientTokenSigners) > 0 {
			s.log.Warnf("Allowing unverified TLS connections for AAA signed clients")
		}

		if len(s.config.Choria.NetworkServerTokenSigners) > 0 {
			s.log.Warnf("Allowing unverified TLS connections for Provisioner signed servers")
		}
	}

	s.opts.TLSConfig = tlsc

	return
}
