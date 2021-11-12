// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package network

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/choria-io/go-choria/tokens"
	"github.com/golang-jwt/jwt/v4"
	"github.com/golang/mock/gomock"
	"github.com/nats-io/nats-server/v2/server"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

var _ = Describe("Network Broker/ChoriaAuth", func() {
	var (
		log        *logrus.Entry
		auth       *ChoriaAuth
		user       *server.User
		mockClient *MockClientAuthentication
		mockctl    *gomock.Controller
	)

	BeforeEach(func() {
		logger := logrus.New()
		logger.Out = io.Discard
		log = logrus.NewEntry(logger)
		auth = &ChoriaAuth{
			clientAllowList: []string{},
			log:             log,
			choriaAccount:   &server.Account{Name: "choria"},
		}
		user = &server.User{
			Username:    "bob",
			Password:    "secret",
			Permissions: &server.Permissions{},
		}

		mockctl = gomock.NewController(GinkgoT())
		mockClient = NewMockClientAuthentication(mockctl)
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	createKeyPair := func() (td string, pri *rsa.PrivateKey) {
		td, err := os.MkdirTemp("", "")
		Expect(err).ToNot(HaveOccurred())

		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		Expect(err).ToNot(HaveOccurred())

		template := x509.Certificate{
			SerialNumber: big.NewInt(1),
			Subject: pkix.Name{
				Organization: []string{"Choria.IO"},
			},
			NotBefore: time.Now(),
			NotAfter:  time.Now().Add(time.Hour * 24 * 180),

			KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
			ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			BasicConstraintsValid: true,
		}

		derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
		Expect(err).ToNot(HaveOccurred())

		out := &bytes.Buffer{}

		pem.Encode(out, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
		err = os.WriteFile(filepath.Join(td, "public.pem"), out.Bytes(), 0600)
		Expect(err).ToNot(HaveOccurred())

		out.Reset()

		blk := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}
		pem.Encode(out, blk)

		err = os.WriteFile(filepath.Join(td, "private.pem"), out.Bytes(), 0600)
		Expect(err).ToNot(HaveOccurred())

		return td, privateKey
	}

	createSignedJWT := func(pk *rsa.PrivateKey, claims map[string]interface{}) string {
		c := map[string]interface{}{
			"exp":      time.Now().UTC().Add(time.Hour).Unix(),
			"nbf":      time.Now().UTC().Add(-1 * time.Minute).Unix(),
			"iat":      time.Now().UTC().Unix(),
			"iss":      "Ginkgo",
			"callerid": "up=ginkgo",
			"sub":      "up=ginkgo",
		}

		for k, v := range claims {
			c[k] = v
		}

		token := jwt.NewWithClaims(jwt.GetSigningMethod("RS512"), jwt.MapClaims(c))
		signed, err := token.SignedString(pk)
		Expect(err).ToNot(HaveOccurred())

		return signed
	}

	Describe("Check", func() {
		Describe("Provisioning user", func() {
			BeforeEach(func() {
				auth.isTLS = true
				auth.provPass = "s3cret"
				auth.provisioningAccount = &server.Account{Name: provisioningUser}
				copts := &server.ClientOpts{Username: "provisioner", Password: "s3cret"}
				mockClient.EXPECT().GetOpts().Return(copts).AnyTimes()
			})

			It("Should only prov auth when tls is enabled", func() {
				auth.isTLS = false
				mockClient.EXPECT().GetTLSConnectionState().Return(&tls.ConnectionState{})
				mockClient.EXPECT().RemoteAddress().Return(&net.IPAddr{IP: net.ParseIP("192.168.0.1"), Zone: ""}).AnyTimes()

				Expect(auth.Check(mockClient)).To(BeFalse())

				auth.isTLS = true
				mockClient.EXPECT().GetTLSConnectionState().Return(&tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{nil}}).AnyTimes()
				mockClient.EXPECT().RegisterUser(gomock.Any()).Do(func(user *server.User) {
					Expect(user.Account).To(Equal(auth.provisioningAccount))
				})

				Expect(auth.Check(mockClient)).To(BeTrue())
			})

			It("Should reject provision user on a plain connection", func() {
				auth.provisioningTokenSigner = "testdata/ssl/certs/rip.mcollective.pem"

				mockClient.EXPECT().GetTLSConnectionState().Return(nil)
				mockClient.EXPECT().RemoteAddress().Return(&net.IPAddr{IP: net.ParseIP("192.168.0.1"), Zone: ""})

				Expect(auth.Check(mockClient)).To(BeFalse())
			})

			It("Should not do provision auth for unverified connections", func() {
				auth.provisioningTokenSigner = "testdata/ssl/certs/rip.mcollective.pem"

				mockClient.EXPECT().RemoteAddress().Return(&net.IPAddr{IP: net.ParseIP("192.168.0.1"), Zone: ""}).AnyTimes()
				mockClient.EXPECT().GetTLSConnectionState().Return(&tls.ConnectionState{})
				Expect(auth.Check(mockClient)).To(BeFalse())
			})

			It("Should verify the password correctly", func() {
				auth.provisioningTokenSigner = "testdata/ssl/certs/rip.mcollective.pem"
				auth.provPass = "other"
				mockClient.EXPECT().RemoteAddress().Return(&net.IPAddr{IP: net.ParseIP("192.168.0.1"), Zone: ""}).AnyTimes()
				mockClient.EXPECT().GetTLSConnectionState().Return(&tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{nil}}).AnyTimes()

				Expect(auth.Check(mockClient)).To(BeFalse())
			})

			It("Should do provision auth for verified connections", func() {
				auth.provisioningTokenSigner = "testdata/ssl/certs/rip.mcollective.pem"

				mockClient.EXPECT().RemoteAddress().Return(&net.IPAddr{IP: net.ParseIP("192.168.0.1"), Zone: ""}).AnyTimes()
				mockClient.EXPECT().GetTLSConnectionState().Return(&tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{nil}}).AnyTimes()
				mockClient.EXPECT().RegisterUser(gomock.Any()).Do(func(user *server.User) {
					Expect(user.Account).To(Equal(auth.provisioningAccount))
				})

				Expect(auth.Check(mockClient)).To(BeTrue())
			})
		})

		Describe("system user", func() {
			BeforeEach(func() {
				auth.isTLS = true
				auth.systemAccount = &server.Account{Name: "system"}
				auth.systemUser = "system"
				auth.systemPass = "sysTem"

				copts := &server.ClientOpts{Username: "system", Password: "sysTem"}
				mockClient.EXPECT().GetOpts().Return(copts).AnyTimes()
				mockClient.EXPECT().RemoteAddress().Return(&net.IPAddr{IP: net.ParseIP("192.168.0.1"), Zone: ""}).AnyTimes()
			})

			It("Should reject non mTLS system users", func() {
				mockClient.EXPECT().GetTLSConnectionState().Return(&tls.ConnectionState{}).AnyTimes()
				Expect(auth.Check(mockClient)).To(BeFalse())
			})

			It("Should verify the password correctly", func() {
				auth.systemPass = "other"
				mockClient.EXPECT().GetTLSConnectionState().Return(&tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{nil}}).AnyTimes()

				Expect(auth.Check(mockClient)).To(BeFalse())
			})

			It("Should register mTLS system users", func() {
				mockClient.EXPECT().GetTLSConnectionState().Return(&tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{nil}}).AnyTimes()
				mockClient.EXPECT().RegisterUser(gomock.Any()).Do(func(user *server.User) {
					Expect(user.Account).To(Equal(auth.systemAccount))
				})

				Expect(auth.Check(mockClient)).To(BeTrue())
			})
		})
	})

	Describe("handleDefaultConnection", func() {
		var (
			td           string
			privateKey   *rsa.PrivateKey
			copts        *server.ClientOpts
			verifiedConn *tls.ConnectionState
		)

		BeforeEach(func() {
			td, privateKey = createKeyPair()
			auth.jwtSigner = filepath.Join(td, "public.pem")
			copts = &server.ClientOpts{
				Token: createSignedJWT(privateKey, map[string]interface{}{
					"purpose": "choria_client_id",
				}),
			}
			verifiedConn = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{nil}}
			mockClient.EXPECT().GetOpts().Return(copts).AnyTimes()
		})

		AfterEach(func() {
			os.RemoveAll(td)
		})

		It("Should require remote info in anon TLS or JWT modes", func() {
			auth.anonTLS = true
			mockClient.EXPECT().RemoteAddress().Return(nil)
			verified, err := auth.handleDefaultConnection(mockClient, verifiedConn, true)
			Expect(verified).To(BeFalse())
			Expect(err).To(MatchError("remote client information is required in anonymous TLS or JWT signing modes"))
		})

		It("Should not access a JWT in non TLS mode", func() {
			auth.jwtSigner = ""
			auth.anonTLS = false
			mockClient.GetOpts().Token = ""

			mockClient.EXPECT().RemoteAddress().Return(&net.IPAddr{IP: net.ParseIP("192.168.0.1"), Zone: ""})
			mockClient.EXPECT().RegisterUser(gomock.Any()).Do(func(user *server.User) {
				Expect(user.Username).To(Equal("")) // caller would be set from the jwt
				Expect(user.Account).To(Equal(auth.choriaAccount))
			})

			verified, err := auth.handleDefaultConnection(mockClient, verifiedConn, true)
			Expect(verified).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should set strict permissions for a client JWT user", func() {
			mockClient.EXPECT().RemoteAddress().Return(&net.IPAddr{IP: net.ParseIP("192.168.0.1"), Zone: ""})
			mockClient.EXPECT().RegisterUser(gomock.Any()).Do(func(user *server.User) {
				Expect(user.Username).To(Equal("up=ginkgo"))
				Expect(user.Account).To(Equal(auth.choriaAccount))
				Expect(user.Permissions.Subscribe).To(Equal(&server.SubjectPermission{
					Allow: []string{"*.reply.e33bf0376d4accbb4a8fd24b2f840b2e.>"},
				}))
				Expect(user.Permissions.Publish).To(Equal(&server.SubjectPermission{
					Allow: []string{
						"*.broadcast.agent.>",
						"*.broadcast.service.>",
						"*.node.>",
						"choria.federation.*.federation",
					},
				}))
			})

			verified, err := auth.handleDefaultConnection(mockClient, verifiedConn, true)
			Expect(err).ToNot(HaveOccurred())
			Expect(verified).To(BeTrue())
		})

		It("Should deny servers when allow list is set and servers are not allowed", func() {
			auth.clientAllowList = []string{"10.0.0.0/24"}
			auth.denyServers = true
			mockClient.GetOpts().Token = ""
			mockClient.EXPECT().RemoteAddress().Return(&net.IPAddr{IP: net.ParseIP("192.168.0.1"), Zone: ""})
			mockClient.EXPECT().RegisterUser(gomock.Any()).Do(func(user *server.User) {
				Expect(user.Username).To(Equal(""))
				Expect(user.Account).To(Equal(auth.choriaAccount))
				Expect(user.Permissions.Subscribe).To(Equal(&server.SubjectPermission{
					Deny: []string{">"},
				}))
				Expect(user.Permissions.Publish).To(Equal(&server.SubjectPermission{
					Deny: []string{">"}},
				))
			})

			verified, err := auth.handleDefaultConnection(mockClient, verifiedConn, true)
			Expect(err).ToNot(HaveOccurred())
			Expect(verified).To(BeTrue())
		})

		It("Should register other clients without restriction", func() {
			mockClient.GetOpts().Token = ""
			mockClient.EXPECT().RemoteAddress().Return(&net.IPAddr{IP: net.ParseIP("192.168.0.1"), Zone: ""})
			mockClient.EXPECT().RegisterUser(gomock.Any()).Do(func(user *server.User) {
				Expect(user.Username).To(Equal(""))
				Expect(user.Account).To(Equal(auth.choriaAccount))
				Expect(user.Permissions.Subscribe).To(BeNil())
				Expect(user.Permissions.Publish).To(BeNil())
			})

			verified, err := auth.handleDefaultConnection(mockClient, verifiedConn, true)
			Expect(err).ToNot(HaveOccurred())
			Expect(verified).To(BeTrue())
		})
	})

	Describe("handleSystemAccount", func() {
		It("Should fail without a password", func() {
			auth.systemUser = ""
			auth.systemPass = ""

			verified, err := auth.handleSystemAccount(mockClient)
			Expect(err).To(MatchError("system user is required"))
			Expect(verified).To(BeFalse())

			auth.systemUser = "system"
			verified, err = auth.handleSystemAccount(mockClient)
			Expect(err).To(MatchError("system password is required"))
			Expect(verified).To(BeFalse())
		})

		It("Should fail without an account", func() {
			auth.systemUser = "system"
			auth.systemPass = "s3cret"

			verified, err := auth.handleSystemAccount(mockClient)
			Expect(err).To(MatchError("system account is not set"))
			Expect(verified).To(BeFalse())
		})

		It("Should verify the password", func() {
			auth.systemUser = "system"
			auth.systemPass = "other"
			auth.systemAccount = &server.Account{Name: "system"}

			mockClient.EXPECT().GetOpts().Return(&server.ClientOpts{Username: "system", Password: "s3cret"}).AnyTimes()
			verified, err := auth.handleSystemAccount(mockClient)
			Expect(err).To(MatchError("invalid system credentials"))
			Expect(verified).To(BeFalse())
		})

		It("Should correctly verify the password and register the user", func() {
			auth.systemUser = "system"
			auth.systemPass = "s3cret"
			auth.systemAccount = &server.Account{Name: "system"}

			mockClient.EXPECT().GetOpts().Return(&server.ClientOpts{Username: "system", Password: "s3cret"}).AnyTimes()
			mockClient.EXPECT().RegisterUser(gomock.Any()).Do(func(user *server.User) {
				Expect(user.Username).To(Equal("system"))
				Expect(user.Password).To(Equal("s3cret"))
				Expect(user.Account).To(Equal(auth.systemAccount))
				Expect(user.Permissions).To(Not(BeNil()))
				Expect(user.Permissions.Publish).To(BeNil())
				Expect(user.Permissions.Subscribe).To(BeNil())
				Expect(user.Permissions.Response).To(BeNil())
			})

			verified, err := auth.handleSystemAccount(mockClient)
			Expect(err).ToNot(HaveOccurred())
			Expect(verified).To(BeTrue())
		})
	})

	Describe("handleProvisioningUserConnection", func() {
		It("Should fail without a password", func() {
			auth.provPass = ""

			verified, err := auth.handleProvisioningUserConnection(mockClient)
			Expect(err).To(MatchError("provisioning user password not enabled"))
			Expect(verified).To(BeFalse())
		})

		It("Should fail without an account", func() {
			auth.provPass = "s3cret"

			verified, err := auth.handleProvisioningUserConnection(mockClient)
			Expect(err).To(MatchError("provisioning account is not set"))
			Expect(verified).To(BeFalse())
		})

		It("Should fail when server not in TLS mode", func() {
			auth.provPass = "s3cret"
			auth.isTLS = false
			auth.provisioningAccount = &server.Account{Name: provisioningUser}

			verified, err := auth.handleProvisioningUserConnection(mockClient)
			Expect(err).To(MatchError("provisioning user access requires TLS"))
			Expect(verified).To(BeFalse())
		})

		It("Should fail when client is not using TLS", func() {
			auth.provPass = "s3cret"
			auth.isTLS = true
			auth.provisioningAccount = &server.Account{Name: provisioningUser}
			mockClient.EXPECT().GetTLSConnectionState().Return(nil).AnyTimes()

			verified, err := auth.handleProvisioningUserConnection(mockClient)
			Expect(err).To(MatchError("provisioning user can only connect over tls"))
			Expect(verified).To(BeFalse())
		})

		It("Should correctly verify the password and register the user", func() {
			auth.provPass = "s3cret"
			auth.isTLS = true
			auth.provisioningAccount = &server.Account{Name: provisioningUser}
			mockClient.EXPECT().GetTLSConnectionState().Return(&tls.ConnectionState{}).AnyTimes()
			mockClient.EXPECT().GetOpts().Return(&server.ClientOpts{Username: provisioningUser, Password: "s3cret"}).AnyTimes()
			mockClient.EXPECT().RegisterUser(gomock.Any()).Do(func(user *server.User) {
				Expect(user.Username).To(Equal(provisioningUser))
				Expect(user.Password).To(Equal("s3cret"))
				Expect(user.Account).To(Equal(auth.provisioningAccount))
				Expect(user.Permissions).To(Not(BeNil()))
				Expect(user.Permissions.Publish).To(BeNil())
				Expect(user.Permissions.Subscribe).To(BeNil())
				Expect(user.Permissions.Response).To(BeNil())
			})

			verified, err := auth.handleProvisioningUserConnection(mockClient)
			Expect(err).ToNot(HaveOccurred())
			Expect(verified).To(BeTrue())
		})
	})

	Describe("handleUnverifiedProvisioningConnection", func() {
		It("Should fail without a signer cert set or present", func() {
			validated, err := auth.handleUnverifiedProvisioningConnection(mockClient)
			Expect(validated).To(BeFalse())
			Expect(err).To(MatchError("provisioning is not enabled"))

			auth.provisioningTokenSigner = "/nonexisting"
			validated, err = auth.handleUnverifiedProvisioningConnection(mockClient)
			Expect(validated).To(BeFalse())
			Expect(err).To(MatchError("provisioning signer certificate /nonexisting does not exist"))
		})

		It("Should fail without a provisioner account", func() {
			auth.provisioningTokenSigner = "testdata/ssl/certs/rip.mcollective.pem"

			validated, err := auth.handleUnverifiedProvisioningConnection(mockClient)
			Expect(validated).To(BeFalse())
			Expect(err).To(MatchError("provisioning account is not set"))
		})

		Describe("Servers", func() {
			BeforeEach(func() {
				auth.provisioningTokenSigner = "testdata/ssl/certs/rip.mcollective.pem"
				auth.provisioningAccount = &server.Account{Name: "provisioning"}
			})

			It("Should fail for invalid tokens", func() {
				t, err := os.ReadFile("testdata/provisioning/invalid.jwt")
				Expect(err).ToNot(HaveOccurred())

				mockClient.EXPECT().GetOpts().Return(&server.ClientOpts{Token: string(t)}).AnyTimes()

				validated, err := auth.handleUnverifiedProvisioningConnection(mockClient)
				Expect(validated).To(BeFalse())
				Expect(err).To(MatchError("could not parse provisioner token: crypto/rsa: verification error"))
			})

			It("Should set server permissions and register", func() {
				t, err := os.ReadFile("testdata/provisioning/secure.jwt")
				Expect(err).ToNot(HaveOccurred())

				mockClient.EXPECT().GetOpts().Return(&server.ClientOpts{Token: string(t)}).AnyTimes()
				mockClient.EXPECT().RemoteAddress().Return(&net.IPAddr{IP: net.ParseIP("192.168.0.1"), Zone: ""})
				mockClient.EXPECT().RegisterUser(gomock.Any()).Do(func(user *server.User) {
					Expect(user.Username).To(Equal(""))
					Expect(user.Password).To(Equal(""))
					Expect(user.Account).To(Equal(auth.provisioningAccount))
					Expect(user.Permissions).To(Not(BeNil()))
					Expect(user.Permissions.Subscribe.Allow).To(Equal([]string{
						"provisioning.node.>",
						"provisioning.broadcast.agent.discovery",
						"provisioning.broadcast.agent.rpcutil",
						"provisioning.broadcast.agent.choria_util",
						"provisioning.broadcast.agent.choria_provision",
					}))
					Expect(user.Permissions.Publish.Allow).To(Equal([]string{
						"choria.lifecycle.>",
						"provisioning.reply.>",
						"provisioning.registration.>",
					}))
				})

				validated, err := auth.handleUnverifiedProvisioningConnection(mockClient)
				Expect(validated).To(BeTrue())
				Expect(err).To(BeNil())
			})
		})

		Describe("Provisioner Client", func() {
			It("Should not accept connections from the provisioning user without verified TLS", func() {
				auth.provisioningTokenSigner = "testdata/ssl/certs/rip.mcollective.pem"
				auth.provisioningAccount = &server.Account{Name: "provisioning"}

				copts := &server.ClientOpts{Username: "provisioner", Password: "s3cret"}
				mockClient.EXPECT().GetOpts().Return(copts).AnyTimes()

				validated, err := auth.handleUnverifiedProvisioningConnection(mockClient)
				Expect(err).To(MatchError("provisioning user requires verified TLS"))
				Expect(validated).To(BeFalse())
			})
		})
	})

	Describe("remoteInClientAllowList", func() {
		It("Should allow all when no allowlist is set", func() {
			ipv4Addr, _, err := net.ParseCIDR("192.0.2.1/24")
			Expect(err).ToNot(HaveOccurred())

			Expect(auth.remoteInClientAllowList(&net.IPAddr{IP: ipv4Addr})).To(BeTrue())
		})

		It("Should handle nil remotes", func() {
			Expect(auth.remoteInClientAllowList(nil)).To(BeTrue())
		})

		It("Should handle invalid remotes", func() {
			ipv4Addr, _, err := net.ParseCIDR("192.0.2.1/24")
			Expect(err).ToNot(HaveOccurred())

			auth.clientAllowList = []string{"192.0.2.1/24"}
			Expect(auth.remoteInClientAllowList(&net.IPAddr{IP: ipv4Addr})).To(BeFalse())
		})

		It("Should handle simple strings", func() {
			ipv4Addr, _, err := net.ParseCIDR("192.0.2.1/24")
			Expect(err).ToNot(HaveOccurred())

			auth.clientAllowList = []string{"192.0.2.1"}
			Expect(auth.remoteInClientAllowList(&net.TCPAddr{IP: ipv4Addr, Port: 1232})).To(BeTrue())
		})

		It("Should handle subnets", func() {
			ipv4Addr, _, err := net.ParseCIDR("192.0.2.1/24")
			Expect(err).ToNot(HaveOccurred())

			auth.clientAllowList = []string{"192.0.0.0/8"}
			Expect(auth.remoteInClientAllowList(&net.TCPAddr{IP: ipv4Addr, Port: 1232})).To(BeTrue())
		})

		It("Should support IPv6", func() {
			auth.clientAllowList = []string{
				"2a00:1450::/32",
				"2a01:1450:4002:801::200e",
			}

			ipv6Addr, _, err := net.ParseCIDR("2a00:1450:4002:801::200e/64")
			Expect(err).ToNot(HaveOccurred())
			Expect(auth.remoteInClientAllowList(&net.TCPAddr{IP: ipv6Addr, Port: 1232})).To(BeTrue())

			ipv6Addr, _, err = net.ParseCIDR("2a01:1450:4002:801::200e/64")
			Expect(err).ToNot(HaveOccurred())
			Expect(auth.remoteInClientAllowList(&net.TCPAddr{IP: ipv6Addr, Port: 1232})).To(BeTrue())

			ipv6Addr, _, err = net.ParseCIDR("2a02:1450:4002:801::200e/64")
			Expect(err).ToNot(HaveOccurred())
			Expect(auth.remoteInClientAllowList(&net.TCPAddr{IP: ipv6Addr, Port: 1232})).To(BeFalse())
		})

		It("Should be false for un matched nodes", func() {
			ipv4Addr, _, err := net.ParseCIDR("192.0.2.1/24")
			Expect(err).ToNot(HaveOccurred())

			auth.clientAllowList = []string{"127.0.0.0/8"}
			Expect(auth.remoteInClientAllowList(&net.TCPAddr{IP: ipv4Addr, Port: 1232})).To(BeFalse())

			ipv4Addr, _, err = net.ParseCIDR("127.0.2.1/24")
			Expect(err).ToNot(HaveOccurred())
			Expect(auth.remoteInClientAllowList(&net.TCPAddr{IP: ipv4Addr, Port: 1232})).To(BeTrue())
		})
	})

	Describe("parseClientIDJWT", func() {
		var (
			td         string
			privateKey *rsa.PrivateKey
		)

		BeforeEach(func() {
			td, privateKey = createKeyPair()
		})

		AfterEach(func() {
			os.RemoveAll(td)
		})

		It("Should fail without a cert", func() {
			_, _, err := auth.parseClientIDJWT("")
			Expect(err).To(MatchError("JWT Signer not set in plugin.choria.network.client_signer_cert, denying all clients"))
		})

		It("Should fail for empty JWTs", func() {
			auth.jwtSigner = "testdata/public.pem"
			_, _, err := auth.parseClientIDJWT("")
			Expect(err).To(MatchError("no JWT received"))
		})

		It("Should verify JWTs", func() {
			auth.jwtSigner = filepath.Join(td, "public.pem")
			signed := createSignedJWT(privateKey, map[string]interface{}{
				"exp": time.Now().UTC().Add(-time.Hour).Unix(),
			})

			caller, perms, err := auth.parseClientIDJWT(signed)
			Expect(err.Error()).To(MatchRegexp("token is expired by"))
			Expect(perms).To(BeNil())
			Expect(caller).To(Equal(""))
		})

		It("Should detect missing callers", func() {
			auth.jwtSigner = filepath.Join(td, "public.pem")
			signed := createSignedJWT(privateKey, map[string]interface{}{
				"callerid": "",
				"purpose":  "choria_client_id",
			})

			caller, perms, err := auth.parseClientIDJWT(signed)
			Expect(err).To(MatchError("no callerid in claims"))
			Expect(perms).To(BeNil())
			Expect(caller).To(Equal(""))
		})

		It("Should expect a purpose field", func() {
			auth.jwtSigner = filepath.Join(td, "public.pem")
			signed := createSignedJWT(privateKey, nil)
			_, _, err := auth.parseClientIDJWT(signed)
			Expect(err).To(MatchError("not a client id token"))

			signed = createSignedJWT(privateKey, map[string]interface{}{
				"purpose": "wrong",
			})
			_, _, err = auth.parseClientIDJWT(signed)
			Expect(err).To(MatchError("not a client id token"))
		})

		It("Should extract the caller", func() {
			auth.jwtSigner = filepath.Join(td, "public.pem")
			signed := createSignedJWT(privateKey, map[string]interface{}{
				"purpose": "choria_client_id",
			})

			caller, perms, err := auth.parseClientIDJWT(signed)
			Expect(err).ToNot(HaveOccurred())
			Expect(perms).To(BeNil())
			Expect(caller).To(Equal("up=ginkgo"))
		})
	})

	Describe("setClientPermissions", func() {
		var (
			log    *logrus.Entry
			minSub []string
			minPub []string
		)

		BeforeEach(func() {
			log = logrus.NewEntry(logrus.New())
			log.Logger.SetOutput(GinkgoWriter)

			minSub = []string{"*.reply.>"}
			minPub = []string{
				"*.broadcast.agent.>",
				"*.broadcast.service.>",
				"*.node.>",
				"choria.federation.*.federation"}
		})

		Describe("System User", func() {
			It("Should should set correct permissions", func() {
				auth.anonTLS = true
				auth.setClientPermissions(user, "", &tokens.ClientPermissions{SystemUser: true}, log)
				Expect(user.Permissions.Subscribe).To(Equal(&server.SubjectPermission{
					Allow: []string{">"},
				}))
				Expect(user.Permissions.Publish).To(Equal(&server.SubjectPermission{
					Allow: []string{">"},
				}))
			})
		})

		Describe("Stream Users", func() {
			It("Should set no permissions for non choria users", func() {
				user.Account = auth.provisioningAccount
				auth.setClientPermissions(user, "", &tokens.ClientPermissions{StreamsUser: true}, log)
				Expect(user.Permissions.Subscribe).To(Equal(&server.SubjectPermission{
					Allow: minSub,
				}))
				Expect(user.Permissions.Publish).To(Equal(&server.SubjectPermission{
					Allow: minPub,
				}))
			})

			It("Should set correct permissions for the choria user", func() {
				user.Account = auth.choriaAccount
				auth.setClientPermissions(user, "", &tokens.ClientPermissions{StreamsUser: true}, log)
				Expect(user.Permissions.Subscribe).To(Equal(&server.SubjectPermission{
					Allow: minSub,
				}))
				Expect(user.Permissions.Publish).To(Equal(&server.SubjectPermission{
					Allow: append(minPub,
						"$JS.API.STREAM.NAMES",
						"$JS.API.STREAM.LIST",
						"$JS.API.STREAM.INFO.*",
						"$JS.API.STREAM.MSG.GET.*",
						"$JS.API.CONSUMER.CREATE.*",
						"$JS.API.CONSUMER.DURABLE.CREATE.*.*",
						"$JS.API.CONSUMER.NAMES.*",
						"$JS.API.CONSUMER.LIST.*",
						"$JS.API.CONSUMER.INFO.*.*",
						"$JS.API.CONSUMER.MSG.NEXT.*.*",
						"$JS.ACK.>",
						"$JS.FC."),
				}))
			})
		})

		Describe("Event Viewers", func() {
			It("Should set provisioning permissions", func() {
				user.Account = auth.provisioningAccount
				auth.setClientPermissions(user, "", &tokens.ClientPermissions{EventsViewer: true}, log)
				Expect(user.Permissions.Subscribe).To(Equal(&server.SubjectPermission{
					Allow: append(minSub, "choria.lifecycle.event.*.provision_mode_server"),
				}))
				Expect(user.Permissions.Publish).To(Equal(&server.SubjectPermission{
					Allow: minPub,
				}))
			})

			It("Should set choria permissions", func() {
				user.Account = auth.choriaAccount
				auth.setClientPermissions(user, "", &tokens.ClientPermissions{EventsViewer: true}, log)
				Expect(user.Permissions.Subscribe).To(Equal(&server.SubjectPermission{
					Allow: append(minSub, "choria.lifecycle.event.>",
						"choria.machine.watcher.>",
						"choria.machine.transition"),
				}))
				Expect(user.Permissions.Publish).To(Equal(&server.SubjectPermission{
					Allow: minPub,
				}))
			})
		})

		Describe("Election Users", func() {
			It("Should set provisioning permissions", func() {
				user.Account = auth.provisioningAccount
				auth.setClientPermissions(user, "", &tokens.ClientPermissions{ElectionUser: true}, log)
				Expect(user.Permissions.Subscribe).To(Equal(&server.SubjectPermission{
					Allow: minSub,
				}))
				Expect(user.Permissions.Publish).To(Equal(&server.SubjectPermission{
					Allow: append(minPub,
						"choria.streams.STREAM.INFO.KV_CHORIA_LEADER_ELECTION",
						"$KV.CHORIA_LEADER_ELECTION.provisioner"),
				}))
			})

			It("Should set choria permissions", func() {
				user.Account = auth.choriaAccount
				auth.setClientPermissions(user, "", &tokens.ClientPermissions{ElectionUser: true}, log)
				Expect(user.Permissions.Subscribe).To(Equal(&server.SubjectPermission{
					Allow: minSub,
				}))
				Expect(user.Permissions.Publish).To(Equal(&server.SubjectPermission{
					Allow: append(minPub,
						"$JS.API.STREAM.INFO.KV_CHORIA_LEADER_ELECTION",
						"$KV.CHORIA_LEADER_ELECTION.>"),
				}))
			})
		})
		Describe("Streams Admin", func() {
			It("Should set no permissions for non choria users", func() {
				user.Account = auth.provisioningAccount
				auth.setClientPermissions(user, "", &tokens.ClientPermissions{StreamsAdmin: true}, log)
				Expect(user.Permissions.Subscribe).To(Equal(&server.SubjectPermission{
					Allow: minSub,
				}))
				Expect(user.Permissions.Publish).To(Equal(&server.SubjectPermission{
					Allow: minPub,
				}))
			})

			It("Should set correct permissions for choria user", func() {
				user.Account = auth.choriaAccount
				auth.setClientPermissions(user, "", &tokens.ClientPermissions{StreamsAdmin: true}, log)
				Expect(user.Permissions.Subscribe).To(Equal(&server.SubjectPermission{
					Allow: append(minSub, "$JS.EVENT.>"),
				}))
				Expect(user.Permissions.Publish).To(Equal(&server.SubjectPermission{
					Allow: append(minPub, "$JS.>"),
				}))
			})
		})

		Describe("Minimal Permissions", func() {
			It("Should support caller private reply subjects", func() {
				auth.anonTLS = true
				auth.setClientPermissions(user, "u=ginkgo", nil, log)
				Expect(user.Permissions.Subscribe).To(Equal(&server.SubjectPermission{
					Allow: []string{"*.reply.0f47cbbd2accc01a51e57261d6e64b8b.>"},
				}))
				Expect(user.Permissions.Publish).To(Equal(&server.SubjectPermission{
					Allow: minPub,
				}))
			})

			It("Should support standard reply subjects", func() {
				auth.anonTLS = true
				auth.setClientPermissions(user, "", nil, log)
				Expect(user.Permissions.Subscribe).To(Equal(&server.SubjectPermission{
					Allow: []string{"*.reply.>"},
				}))
				Expect(user.Permissions.Publish).To(Equal(&server.SubjectPermission{
					Allow: []string{
						"*.broadcast.agent.>",
						"*.broadcast.service.>",
						"*.node.>",
						"choria.federation.*.federation",
					},
				}))
			})
		})
	})

	Describe("setServerPermissions", func() {
		It("Should set correct permissions", func() {
			auth.setServerPermissions(user)

			Expect(user.Permissions.Publish.Allow).To(Equal([]string{
				">",
			}))

			Expect(user.Permissions.Publish.Deny).To(Equal([]string{
				"*.broadcast.agent.>",
				"*.broadcast.service.>",
				"*.node.>",
				"choria.federation.*.federation",
			}))

			Expect(user.Permissions.Subscribe.Allow).To(HaveLen(0))
			Expect(user.Permissions.Subscribe.Deny).To(Equal([]string{
				"*.reply.>",
				"choria.federation.>",
				"choria.lifecycle.>",
			}))
		})

		It("Should support denying servers", func() {
			auth.denyServers = true
			auth.setServerPermissions(user)
			Expect(user.Permissions.Publish.Deny).To(Equal([]string{">"}))
			Expect(user.Permissions.Publish.Allow).To(BeNil())
		})
	})
})
