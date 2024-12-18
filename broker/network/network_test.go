// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package network

import (
	"crypto/tls"
	"testing"
	"time"

	imock "github.com/choria-io/go-choria/inter/imocks"
	"github.com/choria-io/go-choria/srvcache"
	"go.uber.org/mock/gomock"

	"github.com/choria-io/go-choria/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Broker/Network")
}

var _ = Describe("Network Broker", func() {
	var (
		mockctl *gomock.Controller
		cfg     *config.Config
		fw      *imock.MockFramework
		bi      *MockBuildInfoProvider
		srv     *Server
		err     error
	)

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())
		bi = NewMockBuildInfoProvider(mockctl)
		fw, cfg = imock.NewFrameworkForTests(mockctl, GinkgoWriter)
		bi.EXPECT().MaxBrokerClients().Return(50000).AnyTimes()
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	Describe("NewServer", func() {
		BeforeEach(func() {
			cfg.Choria.NetworkListenAddress = "0.0.0.0"
			cfg.Choria.NetworkClientPort = 8080
			cfg.Choria.NetworkWriteDeadline = time.Duration(10 * time.Second)
			cfg.LogLevel = "error"
			cfg.Choria.StatsPort = 8081
			cfg.Choria.StatsListenAddress = "192.168.1.1"
			cfg.Choria.NetworkPeerPort = 8082
			cfg.Choria.NetworkPeerUser = "bob"
			cfg.Choria.NetworkPeerPassword = "secret"
			cfg.Choria.NetworkPeers = []string{"nats://localhost:9000", "nats://localhost:9001", "nats://localhost:8082"}
		})

		It("Should deny trusted signers and issuers being used together", func() {
			fw.EXPECT().NetworkBrokerPeers().Return(srvcache.NewServers(
				srvcache.NewServer("localhost", 9000, "nats"),
				srvcache.NewServer("localhost", 9001, "nats"),
				srvcache.NewServer("localhost", 8082, "nats"),
			), nil).AnyTimes()
			fw.EXPECT().TLSConfig().Return(&tls.Config{}, nil).AnyTimes()

			cfg.Choria.IssuerNames = []string{"choria"}
			cfg.Choria.NetworkClientTokenSigners = []string{"x"}
			_, err = NewServer(fw, bi, false)
			Expect(err).To(MatchError("cannot set trusted client signers using plugin.choria.network.client_signer_cert when an issuer is configured"))

			cfg.Choria.NetworkClientTokenSigners = []string{}
			cfg.Choria.NetworkServerTokenSigners = []string{"x"}
			_, err = NewServer(fw, bi, false)
			Expect(err).To(MatchError("cannot set trusted server signers using plugin.choria.network.server_signer_cert when an issuer is configured"))

			cfg.Choria.NetworkClientTokenSigners = []string{}
			cfg.Choria.NetworkServerTokenSigners = []string{}
			cfg.Choria.ChoriaSecurityTrustedSigners = []string{"x"}
			_, err = NewServer(fw, bi, false)
			Expect(err).To(MatchError("cannot set trusted client signers using plugin.security.choria.trusted_signers when an issuer is configured"))
		})

		It("Should initialize the server correctly", func() {
			cfg.Choria.IssuerNames = []string{"choria"}
			cfg.SetOption("plugin.security.issuer.choria.public", "pk")

			fw.EXPECT().NetworkBrokerPeers().Return(srvcache.NewServers(
				srvcache.NewServer("localhost", 9000, "nats"),
				srvcache.NewServer("localhost", 9001, "nats"),
				srvcache.NewServer("localhost", 8082, "nats"),
			), nil).AnyTimes()
			fw.EXPECT().TLSConfig().Return(&tls.Config{}, nil)

			srv, err = NewServer(fw, bi, false)
			Expect(err).ToNot(HaveOccurred())

			Expect(srv.opts.CustomClientAuthentication.(*ChoriaAuth).issuerTokens["choria"]).To(Equal("pk"))
			Expect(srv.opts.Host).To(Equal("0.0.0.0"))
			Expect(srv.opts.Port).To(Equal(8080))
			Expect(srv.opts.Logtime).To(BeFalse())
			Expect(srv.opts.MaxConn).To(Equal(50000))
			Expect(srv.opts.WriteDeadline).To(Equal(time.Duration(10 * time.Second)))
			Expect(srv.opts.NoSigs).To(BeTrue())
			Expect(srv.opts.Debug).To(BeFalse())
			Expect(srv.opts.HTTPHost).To(Equal("192.168.1.1"))
			Expect(srv.opts.HTTPPort).To(Equal(8081))
			Expect(srv.opts.Cluster.Host).To(Equal("0.0.0.0"))
			Expect(srv.opts.Cluster.NoAdvertise).To(BeTrue())
			Expect(srv.opts.Cluster.Port).To(Equal(8082))
			Expect(srv.opts.Cluster.Username).To(Equal("bob"))
			Expect(srv.opts.Cluster.Password).To(Equal("secret"))
			Expect(srv.opts.Routes).To(HaveLen(2))
			Expect(srv.opts.Routes[0].Host).To(Equal("localhost:9000"))
			Expect(srv.opts.Routes[1].Host).To(Equal("localhost:9001"))
			Expect(srv.opts.TLS).To(BeTrue())
			Expect(srv.opts.TLSVerify).To(BeTrue())
			Expect(srv.opts.TLSTimeout).To(Equal(float64(2)))
			Expect(srv.opts.LeafNode.Host).To(Equal(""))
			Expect(srv.opts.LeafNode.Port).To(Equal(0))
		})

		It("Should support disabling TLS", func() {
			cfg.DisableTLS = true
			fw.EXPECT().NetworkBrokerPeers().Return(srvcache.NewServers(), nil)

			srv, err = NewServer(fw, bi, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(srv.opts.TLS).To(BeFalse())
		})

		It("Should support forcing client TLS on while framework TLS is off", func() {
			cfg.DisableTLS = true
			cfg.Choria.NetworkClientTLSForce = true

			fw.EXPECT().NetworkBrokerPeers().Return(srvcache.NewServers(), nil)
			fw.EXPECT().ValidateSecurity().Return([]string{}, true)
			fw.EXPECT().TLSConfig().Return(&tls.Config{}, nil)

			srv, err = NewServer(fw, bi, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(srv.opts.TLS).To(BeTrue())
		})

		Describe("WebSocket", func() {
			BeforeEach(func() {
				fw.EXPECT().TLSConfig().Return(&tls.Config{}, nil).AnyTimes()
				fw.EXPECT().NetworkBrokerPeers().Return(srvcache.NewServers(), nil)
			})

			It("Should correctly configure websockets", func() {
				cfg.Choria.NetworkWebSocketPort = 4223
				cfg.Choria.NetworkWebSocketAdvertise = "wss://example.net:433"
				srv, err = NewServer(fw, bi, false)
				Expect(err).ToNot(HaveOccurred())
				Expect(srv.opts.Websocket.Port).To(Equal(cfg.Choria.NetworkWebSocketPort))

				tlsc, _ := fw.TLSConfig()
				Expect(srv.opts.Websocket.TLSConfig).To(Equal(tlsc))
			})
		})

		Describe("Gateways", func() {
			It("Should require a name and remotes", func() {
				fw, cfg = imock.NewFrameworkForTests(mockctl, GinkgoWriter, imock.WithConfigFile("testdata/gateways/noremotes.cfg"))
				fw.EXPECT().TLSConfig().Return(&tls.Config{}, nil).AnyTimes()
				fw.EXPECT().NetworkBrokerPeers().Return(srvcache.NewServers(), nil).AnyTimes()

				srv, err = NewServer(fw, bi, false)
				Expect(err).ToNot(HaveOccurred())

				err = srv.setupGateways()
				Expect(err).ToNot(HaveOccurred())
				Expect(srv.opts.Gateway.Port).To(Equal(0))
			})

			It("Should support remote gateways", func() {
				fw, cfg = imock.NewFrameworkForTests(mockctl, GinkgoWriter, imock.WithConfigFile("testdata/gateways/remotes.cfg"))
				fw.EXPECT().TLSConfig().Return(&tls.Config{}, nil).AnyTimes()
				fw.EXPECT().NetworkBrokerPeers().Return(srvcache.NewServers(), nil).AnyTimes()

				srv, err = NewServer(fw, bi, false)
				Expect(err).ToNot(HaveOccurred())

				Expect(srv.config.Choria.NetworkGatewayRemotes).To(Equal([]string{"C1", "C2"}))

				Expect(srv.opts.Gateway.Name).To(Equal("CHORIA"))
				Expect(srv.opts.Gateway.RejectUnknown).To(BeTrue())

				remotes := srv.opts.Gateway.Gateways
				Expect(remotes).To(HaveLen(2))
				Expect(remotes[0].Name).To(Equal("C1"))
				Expect(remotes[0].URLs).To(HaveLen(2))
				Expect(remotes[0].URLs[0].String()).To(Equal("nats://c1-1.example.net:7222"))
				Expect(remotes[0].URLs[1].String()).To(Equal("nats://c1-2.example.net:7222"))
				Expect(remotes[1].Name).To(Equal("C2"))
				Expect(remotes[1].TLSConfig).To(BeNil())
				Expect(remotes[1].URLs).To(HaveLen(2))
				Expect(remotes[1].URLs[0].String()).To(Equal("nats://c2-1.example.net:7222"))
				Expect(remotes[1].URLs[1].String()).To(Equal("nats://c2-2.example.net:7222"))
			})

			It("Should handle missing custom TLS", func() {
				fw, cfg = imock.NewFrameworkForTests(mockctl, GinkgoWriter, imock.WithConfigFile("testdata/gateways/missingtls.cfg"))
				fw.EXPECT().TLSConfig().Return(&tls.Config{}, nil).AnyTimes()
				fw.EXPECT().NetworkBrokerPeers().Return(srvcache.NewServers(), nil).AnyTimes()

				srv, err = NewServer(fw, bi, false)
				Expect(err).ToNot(HaveOccurred())

				Expect(srv.opts.Gateway.Gateways).To(HaveLen(1))
				Expect(srv.opts.Gateway.Gateways[0].Name).To(Equal("C2"))
			})

			It("Should support custom TLS", func() {
				fw, cfg = imock.NewFrameworkForTests(mockctl, GinkgoWriter, imock.WithConfigFile("testdata/gateways/customtls.cfg"))
				fw.EXPECT().TLSConfig().Return(&tls.Config{}, nil).AnyTimes()
				fw.EXPECT().NetworkBrokerPeers().Return(srvcache.NewServers(), nil).AnyTimes()

				srv, err = NewServer(fw, bi, false)
				Expect(err).ToNot(HaveOccurred())

				Expect(srv.opts.Gateway.Gateways).To(HaveLen(2))
				Expect(srv.opts.Gateway.Gateways[1].TLSConfig).ToNot(BeNil())
			})
		})

		Describe("Leafnodes", func() {
			It("Should support basic listening only leafnodes mode", func() {
				fw, cfg = imock.NewFrameworkForTests(mockctl, GinkgoWriter, imock.WithConfigFile("testdata/leafnodes/listening.cfg"))
				fw.EXPECT().TLSConfig().Return(&tls.Config{}, nil).AnyTimes()
				fw.EXPECT().NetworkBrokerPeers().Return(srvcache.NewServers(), nil).AnyTimes()

				srv, err = NewServer(fw, bi, false)
				Expect(err).ToNot(HaveOccurred())
				Expect(srv.opts.LeafNode.Port).To(Equal(6222))
				Expect(srv.opts.LeafNode.Remotes).To(BeEmpty())
			})

			It("Should support connecting to leafnodes", func() {
				fw, cfg = imock.NewFrameworkForTests(mockctl, GinkgoWriter, imock.WithConfigFile("testdata/leafnodes/remotes.cfg"))
				fw.EXPECT().TLSConfig().Return(&tls.Config{}, nil).AnyTimes()
				fw.EXPECT().NetworkBrokerPeers().Return(srvcache.NewServers(), nil).AnyTimes()
				Expect(cfg.Choria.NetworkLeafRemotes).To(Equal([]string{"ln1", "ln2"}))

				srv, err = NewServer(fw, bi, false)
				Expect(err).ToNot(HaveOccurred())
				Expect(srv.IsTLS()).To(BeTrue())
				Expect(srv.opts.LeafNode.Port).To(Equal(6222))
				Expect(srv.opts.LeafNode.Remotes).To(HaveLen(2))
				Expect(srv.opts.LeafNode.Remotes[0].URLs[0].String()).To(Equal("leafnode://ln1-1.example.net:6222"))
				Expect(srv.opts.LeafNode.Remotes[0].TLS).To(BeTrue())
				Expect(srv.opts.LeafNode.Remotes[1].URLs[0].String()).To(Equal("leafnode://ln2.example.net:6222"))
			})

			It("Should handle missing custom TLS", func() {
				fw, cfg = imock.NewFrameworkForTests(mockctl, GinkgoWriter, imock.WithConfigFile("testdata/leafnodes/missingtls.cfg"))
				fw.EXPECT().TLSConfig().Return(&tls.Config{}, nil).AnyTimes()
				fw.EXPECT().NetworkBrokerPeers().Return(srvcache.NewServers(), nil).AnyTimes()
				Expect(cfg.Choria.NetworkLeafRemotes).To(Equal([]string{"ln1", "ln2"}))

				srv, err = NewServer(fw, bi, false)
				Expect(err).ToNot(HaveOccurred())
				Expect(srv.opts.LeafNode.Port).To(Equal(6222))
				Expect(srv.opts.LeafNode.Remotes).To(HaveLen(1))
				Expect(srv.opts.LeafNode.Remotes[0].URLs[0].String()).To(Equal("leafnode://ln2.example.net:6222"))
			})

			It("Should handle custom TLS", func() {
				fw, cfg = imock.NewFrameworkForTests(mockctl, GinkgoWriter, imock.WithConfigFile("testdata/leafnodes/customtls.cfg"))
				fw.EXPECT().TLSConfig().Return(&tls.Config{}, nil).AnyTimes()
				fw.EXPECT().NetworkBrokerPeers().Return(srvcache.NewServers(), nil).AnyTimes()

				srv, err = NewServer(fw, bi, false)
				Expect(err).ToNot(HaveOccurred())
				Expect(srv.opts.LeafNode.Port).To(Equal(6222))
				Expect(srv.opts.LeafNode.Remotes).To(HaveLen(1))
				Expect(srv.opts.LeafNode.Remotes[0].URLs[0].String()).To(Equal("leafnode://ln1.example.net:6222"))
				Expect(srv.opts.LeafNode.Remotes[0].TLS).To(BeTrue())
				Expect(srv.opts.LeafNode.Remotes[0].TLSConfig).ToNot(BeNil())
			})
		})
	})
})
