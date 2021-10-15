// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
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

	"github.com/golang-jwt/jwt"
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
			allowList:     []string{},
			log:           log,
			choriaAccount: &server.Account{Name: "choria"},
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

	Describe("Check", func() {
		Describe("Unverified connections", func() {
			It("Should only prov auth when tls is enabled", func() {
				auth.isTLS = false
				auth.provPass = "s3cret"
				auth.provisioningAccount = &server.Account{Name: provisioningUser}
				copts := &server.ClientOpts{Username: provisioningUser, Password: "s3cret"}
				mockClient.EXPECT().GetOpts().Return(copts).AnyTimes()
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
				auth.isTLS = true
				auth.provisioningTokenSigner = "testdata/ssl/certs/rip.mcollective.pem"
				auth.provPass = "s3cret"
				auth.provisioningAccount = &server.Account{Name: "provision"}

				copts := &server.ClientOpts{Username: "provisioner", Password: "s3cret"}
				mockClient.EXPECT().GetOpts().Return(copts).AnyTimes()
				mockClient.EXPECT().GetTLSConnectionState().Return(nil)
				mockClient.EXPECT().RemoteAddress().Return(&net.IPAddr{IP: net.ParseIP("192.168.0.1"), Zone: ""})

				Expect(auth.Check(mockClient)).To(BeFalse())
			})

			It("Should not do provision auth for unverified connections", func() {
				auth.isTLS = true
				auth.provisioningTokenSigner = "testdata/ssl/certs/rip.mcollective.pem"
				auth.provPass = "s3cret"
				auth.provisioningAccount = &server.Account{Name: "provision"}

				copts := &server.ClientOpts{Username: "provisioner", Password: "s3cret"}
				mockClient.EXPECT().GetOpts().Return(copts).AnyTimes()
				mockClient.EXPECT().RemoteAddress().Return(&net.IPAddr{IP: net.ParseIP("192.168.0.1"), Zone: ""}).AnyTimes()
				mockClient.EXPECT().GetTLSConnectionState().Return(&tls.ConnectionState{})
				Expect(auth.Check(mockClient)).To(BeFalse())
			})

			It("Should do provision auth for verified connections", func() {
				auth.isTLS = true
				auth.provisioningTokenSigner = "testdata/ssl/certs/rip.mcollective.pem"
				auth.provPass = "s3cret"
				auth.provisioningAccount = &server.Account{Name: "provision"}

				copts := &server.ClientOpts{Username: "provisioner", Password: "s3cret"}
				mockClient.EXPECT().GetOpts().Return(copts).AnyTimes()
				mockClient.EXPECT().RemoteAddress().Return(&net.IPAddr{IP: net.ParseIP("192.168.0.1"), Zone: ""}).AnyTimes()
				mockClient.EXPECT().GetTLSConnectionState().Return(&tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{nil}}).AnyTimes()
				mockClient.EXPECT().RegisterUser(gomock.Any()).Do(func(user *server.User) {
					Expect(user.Account).To(Equal(auth.provisioningAccount))
				})

				Expect(auth.Check(mockClient)).To(BeTrue())
			})
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
				auth.isTLS = true
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

			auth.allowList = []string{"192.0.2.1/24"}
			Expect(auth.remoteInClientAllowList(&net.IPAddr{IP: ipv4Addr})).To(BeFalse())
		})

		It("Should handle simple strings", func() {
			ipv4Addr, _, err := net.ParseCIDR("192.0.2.1/24")
			Expect(err).ToNot(HaveOccurred())

			auth.allowList = []string{"192.0.2.1"}
			Expect(auth.remoteInClientAllowList(&net.TCPAddr{IP: ipv4Addr, Port: 1232})).To(BeTrue())
		})

		It("Should handle subnets", func() {
			ipv4Addr, _, err := net.ParseCIDR("192.0.2.1/24")
			Expect(err).ToNot(HaveOccurred())

			auth.allowList = []string{"192.0.0.0/8"}
			Expect(auth.remoteInClientAllowList(&net.TCPAddr{IP: ipv4Addr, Port: 1232})).To(BeTrue())
		})

		It("Should support IPv6", func() {
			auth.allowList = []string{
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

			auth.allowList = []string{"127.0.0.0/8"}
			Expect(auth.remoteInClientAllowList(&net.TCPAddr{IP: ipv4Addr, Port: 1232})).To(BeFalse())

			ipv4Addr, _, err = net.ParseCIDR("127.0.2.1/24")
			Expect(err).ToNot(HaveOccurred())
			Expect(auth.remoteInClientAllowList(&net.TCPAddr{IP: ipv4Addr, Port: 1232})).To(BeTrue())
		})
	})

	Describe("parseAnonTLSJWTUser", func() {
		var (
			td         string
			err        error
			privateKey *rsa.PrivateKey
		)

		BeforeEach(func() {
			td, err = os.MkdirTemp("", "")
			Expect(err).ToNot(HaveOccurred())

			privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
			Expect(err).ToNot(HaveOccurred())

			template := x509.Certificate{
				SerialNumber: big.NewInt(1),
				Subject: pkix.Name{
					Organization: []string{"Acme Co"},
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
		})

		It("Should fail without a cert", func() {
			_, err := auth.parseAnonTLSJWTUser("")
			Expect(err).To(MatchError("anonymous TLS JWT Signer not set in plugin.choria.security.request_signing_certificate, denying all clients"))
		})

		It("Should fail for empty JWTs", func() {
			auth.jwtSigner = "testdata/public.pem"
			_, err := auth.parseAnonTLSJWTUser("")
			Expect(err).To(MatchError("no JWT received"))
		})

		It("Should verify JWTs", func() {
			auth.jwtSigner = filepath.Join(td, "public.pem")
			claims := map[string]interface{}{
				"exp":      time.Now().UTC().Add(-time.Hour).Unix(),
				"nbf":      time.Now().UTC().Add(-1 * time.Minute).Unix(),
				"iat":      time.Now().UTC().Unix(),
				"iss":      "Ginkgo",
				"callerid": "up=ginkgo",
				"sub":      "up=ginkgo",
			}

			token := jwt.NewWithClaims(jwt.GetSigningMethod("RS512"), jwt.MapClaims(claims))
			signed, err := token.SignedString(privateKey)
			Expect(err).ToNot(HaveOccurred())
			caller, err := auth.parseAnonTLSJWTUser(signed)
			Expect(err).To(MatchError("invalid JWT: Token is expired"))
			Expect(caller).To(Equal(""))
		})

		It("Should detect missing callers", func() {
			auth.jwtSigner = filepath.Join(td, "public.pem")
			claims := map[string]interface{}{
				"exp": time.Now().UTC().Add(time.Hour).Unix(),
				"nbf": time.Now().UTC().Add(-1 * time.Minute).Unix(),
				"iat": time.Now().UTC().Unix(),
				"iss": "Ginkgo",
				"sub": "up=ginkgo",
			}

			token := jwt.NewWithClaims(jwt.GetSigningMethod("RS512"), jwt.MapClaims(claims))
			signed, err := token.SignedString(privateKey)
			Expect(err).ToNot(HaveOccurred())
			caller, err := auth.parseAnonTLSJWTUser(signed)
			Expect(err).To(MatchError("no callerid in claims"))
			Expect(caller).To(Equal(""))
		})

		It("Should extract the caller", func() {
			auth.jwtSigner = filepath.Join(td, "public.pem")
			claims := map[string]interface{}{
				"exp":      time.Now().UTC().Add(time.Hour).Unix(),
				"nbf":      time.Now().UTC().Add(-1 * time.Minute).Unix(),
				"iat":      time.Now().UTC().Unix(),
				"iss":      "Ginkgo",
				"callerid": "up=ginkgo",
				"sub":      "up=ginkgo",
			}

			token := jwt.NewWithClaims(jwt.GetSigningMethod("RS512"), jwt.MapClaims(claims))
			signed, err := token.SignedString(privateKey)
			Expect(err).ToNot(HaveOccurred())
			caller, err := auth.parseAnonTLSJWTUser(signed)
			Expect(err).ToNot(HaveOccurred())
			Expect(caller).To(Equal("up=ginkgo"))
		})
	})

	Describe("setClientPermissions", func() {
		It("Should do nothing when not in anonymous tls mode", func() {
			auth.anonTLS = false
			auth.setClientPermissions(user, "")
			Expect(user.Permissions.Subscribe).To(BeNil())
			Expect(user.Permissions.Publish).To(BeNil())
		})

		It("Should support caller private reply subjects", func() {
			auth.anonTLS = true
			auth.setClientPermissions(user, "u=ginkgo")
			Expect(user.Permissions.Subscribe).To(Equal(&server.SubjectPermission{
				Allow: []string{"*.reply.0f47cbbd2accc01a51e57261d6e64b8b.>"},
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

		It("Should support standard reply subjects", func() {
			auth.anonTLS = true
			auth.setClientPermissions(user, "")
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
