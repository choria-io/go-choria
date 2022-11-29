// Copyright (c) 2020-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package network

import (
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/integration/testutil"
	iu "github.com/choria-io/go-choria/internal/util"
	"github.com/choria-io/go-choria/tokens"
	"github.com/golang-jwt/jwt/v4"
	"github.com/golang/mock/gomock"
	"github.com/nats-io/nats-server/v2/server"
	. "github.com/onsi/ginkgo/v2"
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
			issuerTokens:    map[string]string{},
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

		pri, err = testutil.CreateRSAKeyAndCert(td)
		Expect(err).ToNot(HaveOccurred())

		return td, pri
	}

	createSignedServerJWT := func(pk any, pubK []byte, claims map[string]any) string {
		signed, err := testutil.CreateSignedServerJWT(pk, pubK, claims)
		Expect(err).ToNot(HaveOccurred())

		return signed
	}

	createSignedClientJWT := func(pk any, claims map[string]any) string {
		signed, err := testutil.CreateSignedClientJWT(pk, claims)
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
			var copts *server.ClientOpts

			BeforeEach(func() {
				auth.isTLS = true
				auth.systemAccount = &server.Account{Name: "system"}
				auth.systemUser = "system"
				auth.systemPass = "sysTem"

				copts = &server.ClientOpts{Username: "system", Password: "sysTem"}
				mockClient.EXPECT().GetOpts().Return(copts).AnyTimes()
				mockClient.EXPECT().RemoteAddress().Return(&net.IPAddr{IP: net.ParseIP("192.168.0.1"), Zone: ""}).AnyTimes()
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

			It("Should reject non mTLS system users that has no JWT", func() {
				mockClient.EXPECT().GetTLSConnectionState().Return(&tls.ConnectionState{}).AnyTimes()
				mockClient.EXPECT().RegisterUser(gomock.Any()).Times(0)
				Expect(auth.Check(mockClient)).To(BeFalse())
			})

			Describe("JWT based system access", func() {
				var (
					td           string
					privateKey   *rsa.PrivateKey
					edPrivateKey ed25519.PrivateKey
					edPublicKey  ed25519.PublicKey
					// verifiedConn *tls.ConnectionState
					err error
				)

				BeforeEach(func() {
					td, privateKey = createKeyPair()
					auth.clientJwtSigners = []string{filepath.Join(td, "public.pem")}
					edPublicKey, edPrivateKey, err = choria.Ed25519KeyPair()
					Expect(err).ToNot(HaveOccurred())
					sig, err := choria.Ed25519Sign(edPrivateKey, []byte("toomanysecrets"))
					mockClient.EXPECT().GetNonce().Return([]byte("toomanysecrets")).AnyTimes()
					Expect(err).ToNot(HaveOccurred())
					copts.Sig = base64.RawURLEncoding.EncodeToString(sig)
					mockClient.EXPECT().Kind().Return(server.CLIENT).AnyTimes()
				})

				AfterEach(func() {
					os.RemoveAll(td)
				})

				It("Should reject non mTLS system users with a JWT but without the needed permissions", func() {
					mockClient.EXPECT().RegisterUser(gomock.Any()).Times(0)
					mockClient.EXPECT().GetTLSConnectionState().Return(&tls.ConnectionState{}).AnyTimes()
					copts.Token = createSignedClientJWT(privateKey, map[string]any{
						"purpose":    tokens.ClientIDPurpose,
						"public_key": hex.EncodeToString(edPublicKey),
					})

					Expect(auth.Check(mockClient)).To(BeFalse())
				})

				It("Should reject non mTLS system users with a JWT that does not allow system access", func() {
					mockClient.EXPECT().GetTLSConnectionState().Return(&tls.ConnectionState{}).AnyTimes()
					mockClient.EXPECT().RegisterUser(gomock.Any()).Times(0)

					copts.Token = createSignedClientJWT(privateKey, map[string]any{
						"purpose":     tokens.ClientIDPurpose,
						"public_key":  hex.EncodeToString(edPublicKey),
						"permissions": map[string]bool{},
					})

					Expect(auth.Check(mockClient)).To(BeFalse())
				})

				It("Should only accept system users with a JWT over TLS", func() {
					mockClient.EXPECT().GetTLSConnectionState().Return(nil).AnyTimes()
					mockClient.EXPECT().RegisterUser(gomock.Any()).Times(0)

					copts.Token = createSignedClientJWT(privateKey, map[string]any{
						"purpose":    tokens.ClientIDPurpose,
						"public_key": hex.EncodeToString(edPublicKey),
						"permissions": map[string]bool{
							"system_user": true,
						},
					})

					Expect(auth.Check(mockClient)).To(BeFalse())
				})

				It("Should accept non mTLS system users with a correct JWT", func() {
					mockClient.EXPECT().GetTLSConnectionState().Return(&tls.ConnectionState{}).AnyTimes()
					mockClient.EXPECT().RegisterUser(gomock.Any()).Do(func(user *server.User) {
						Expect(user.Account).To(Equal(auth.systemAccount))
					})

					copts.Token = createSignedClientJWT(privateKey, map[string]any{
						"purpose":    tokens.ClientIDPurpose,
						"public_key": hex.EncodeToString(edPublicKey),
						"permissions": map[string]bool{
							"system_user": true,
						},
					})

					Expect(auth.Check(mockClient)).To(BeTrue())
				})
			})
		})
	})

	Describe("handleDefaultConnection", func() {
		var (
			td           string
			privateKey   *rsa.PrivateKey
			edPrivateKey ed25519.PrivateKey
			edPublicKey  ed25519.PublicKey
			copts        *server.ClientOpts
			verifiedConn *tls.ConnectionState
			err          error
		)

		BeforeEach(func() {
			td, privateKey = createKeyPair()
			auth.serverJwtSigners = []string{filepath.Join(td, "public.pem")}
			edPublicKey, edPrivateKey, err = choria.Ed25519KeyPair()
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			os.RemoveAll(td)
		})

		Describe("Servers", func() {
			BeforeEach(func() {
				auth.serverJwtSigners = []string{filepath.Join(td, "public.pem")}
				auth.clientAllowList = nil
				auth.denyServers = false

				copts = &server.ClientOpts{
					Token: createSignedServerJWT(privateKey, edPublicKey, map[string]any{
						"purpose":     tokens.ServerPurpose,
						"public_key":  hex.EncodeToString(edPublicKey),
						"collectives": []string{"c1", "c2"},
					}),
				}
				verifiedConn = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{nil}}
				mockClient.EXPECT().GetOpts().Return(copts).AnyTimes()
				mockClient.EXPECT().Kind().Return(server.CLIENT).AnyTimes()
			})

			It("Should require a remote", func() {
				_, err := auth.verifyServerJWTBasedAuth(nil, "", nil, "", log)
				Expect(err).To(MatchError("remote client information is required in anonymous TLS or JWT signing modes"))
			})

			It("Should fail on invalid jwt", func() {
				_, err := auth.verifyServerJWTBasedAuth(&net.IPAddr{IP: net.ParseIP("192.168.0.1"), Zone: ""}, "x", nil, "", log)
				Expect(err).To(MatchError("invalid JWT token"))
			})

			It("Should fail for invalid nonce", func() {
				copts.Sig = "wrong"
				mockClient.EXPECT().RemoteAddress().Return(&net.IPAddr{IP: net.ParseIP("192.168.0.1"), Zone: ""})
				mockClient.EXPECT().GetNonce().Return([]byte("toomanysecrets"))

				verified, err := auth.handleDefaultConnection(mockClient, verifiedConn, true, log)
				Expect(err).To(MatchError("invalid nonce signature or jwt token"))
				Expect(verified).To(BeFalse())
			})

			It("Should deny servers when allow list is set and servers are not allowed", func() {
				auth.clientAllowList = []string{"10.0.0.0/24"}
				auth.denyServers = true
				mockClient.GetOpts().Token = ""
				mockClient.EXPECT().RemoteAddress().Return(&net.IPAddr{IP: net.ParseIP("192.168.0.1"), Zone: ""})
				mockClient.EXPECT().GetNonce().Return(nil)
				mockClient.EXPECT().RegisterUser(gomock.Any()).Do(func(user *server.User) {
					Expect(user.Username).To(BeEmpty())
					Expect(user.Account).To(Equal(auth.choriaAccount))
					Expect(user.Permissions.Subscribe).To(Equal(&server.SubjectPermission{
						Deny: []string{">"},
					}))
					Expect(user.Permissions.Publish).To(Equal(&server.SubjectPermission{
						Deny: []string{">"}},
					))
				})

				verified, err := auth.handleDefaultConnection(mockClient, verifiedConn, true, log)
				Expect(err).ToNot(HaveOccurred())
				Expect(verified).To(BeTrue())
			})

			Describe("Server Permissions", func() {
				BeforeEach(func() {
					auth.denyServers = false
					sig, err := choria.Ed25519Sign(edPrivateKey, []byte("toomanysecrets"))
					Expect(err).ToNot(HaveOccurred())
					copts.Sig = base64.RawURLEncoding.EncodeToString(sig)

					mockClient.EXPECT().RemoteAddress().Return(&net.IPAddr{IP: net.ParseIP("192.168.0.1"), Zone: ""})
					mockClient.EXPECT().GetNonce().Return([]byte("toomanysecrets"))
				})

				It("Should set strict permissions for a server JWT user", func() {
					mockClient.EXPECT().RegisterUser(gomock.Any()).Do(func(user *server.User) {
						Expect(user.Username).To(Equal("ginkgo.example.net"))
						Expect(user.Account).To(Equal(auth.choriaAccount))
						Expect(user.Permissions.Subscribe).To(Equal(&server.SubjectPermission{
							Allow: []string{
								"c1.broadcast.agent.>",
								"c1.node.ginkgo.example.net",
								"c1.reply.3f7c3a791b0eb10da51dca4cdedb9418.>",
								"c2.broadcast.agent.>",
								"c2.node.ginkgo.example.net",
								"c2.reply.3f7c3a791b0eb10da51dca4cdedb9418.>",
							},
						}))
						Expect(user.Permissions.Publish).To(Equal(&server.SubjectPermission{
							Allow: []string{
								"choria.lifecycle.>",
								"choria.machine.transition",
								"choria.machine.watcher.>",
								"c1.reply.>",
								"c1.broadcast.agent.registration",
								"choria.federation.c1.collective",
								"c2.reply.>",
								"c2.broadcast.agent.registration",
								"choria.federation.c2.collective",
							},
						}))
					})

					verified, err := auth.handleDefaultConnection(mockClient, verifiedConn, true, log)
					Expect(err).ToNot(HaveOccurred())
					Expect(verified).To(BeTrue())
				})

				It("Should support denying servers", func() {
					auth.denyServers = true
					mockClient.EXPECT().RegisterUser(gomock.Any()).Do(func(user *server.User) {
						Expect(user.Username).To(Equal("ginkgo.example.net"))
						Expect(user.Account).To(Equal(auth.choriaAccount))
						Expect(user.Permissions.Subscribe).To(Equal(&server.SubjectPermission{
							Deny: []string{">"},
						}))
						Expect(user.Permissions.Publish).To(Equal(&server.SubjectPermission{
							Deny: []string{">"},
						}))
					})

					verified, err := auth.handleDefaultConnection(mockClient, verifiedConn, true, log)
					Expect(err).ToNot(HaveOccurred())
					Expect(verified).To(BeTrue())
				})

				It("Should handle no collectives being set", func() {
					copts.Token = createSignedServerJWT(privateKey, edPublicKey, map[string]any{
						"purpose":    tokens.ServerPurpose,
						"public_key": hex.EncodeToString(edPublicKey),
					})

					mockClient.EXPECT().RegisterUser(gomock.Any()).Do(func(user *server.User) {
						Expect(user.Username).To(Equal("ginkgo.example.net"))
						Expect(user.Account).To(Equal(auth.choriaAccount))
						Expect(user.Permissions.Subscribe).To(Equal(&server.SubjectPermission{
							Deny: []string{">"},
						}))
						Expect(user.Permissions.Publish).To(Equal(&server.SubjectPermission{
							Deny: []string{">"},
						}))
					})

					verified, err := auth.handleDefaultConnection(mockClient, verifiedConn, true, log)
					Expect(err).ToNot(HaveOccurred())
					Expect(verified).To(BeTrue())
				})

				It("Should support service hosts", func() {
					copts.Token = createSignedServerJWT(privateKey, edPublicKey, map[string]any{
						"purpose":     tokens.ServerPurpose,
						"public_key":  hex.EncodeToString(edPublicKey),
						"collectives": []string{"c1", "c2"},
						"permissions": &tokens.ServerPermissions{ServiceHost: true},
					})

					mockClient.EXPECT().RegisterUser(gomock.Any()).Do(func(user *server.User) {
						Expect(user.Username).To(Equal("ginkgo.example.net"))
						Expect(user.Account).To(Equal(auth.choriaAccount))
						Expect(user.Permissions.Subscribe).To(Equal(&server.SubjectPermission{
							Allow: []string{
								"c1.broadcast.agent.>",
								"c1.node.ginkgo.example.net",
								"c1.reply.3f7c3a791b0eb10da51dca4cdedb9418.>",
								"c1.broadcast.service.>",
								"c2.broadcast.agent.>",
								"c2.node.ginkgo.example.net",
								"c2.reply.3f7c3a791b0eb10da51dca4cdedb9418.>",
								"c2.broadcast.service.>",
							},
						}))
					})

					verified, err := auth.handleDefaultConnection(mockClient, verifiedConn, true, log)
					Expect(err).ToNot(HaveOccurred())
					Expect(verified).To(BeTrue())
				})

				It("Should support Governors", func() {
					copts.Token = createSignedServerJWT(privateKey, edPublicKey, map[string]any{
						"purpose":     tokens.ServerPurpose,
						"public_key":  hex.EncodeToString(edPublicKey),
						"collectives": []string{"c1", "c2"},
						"permissions": &tokens.ServerPermissions{Governor: true, Streams: true},
					})

					mockClient.EXPECT().RegisterUser(gomock.Any()).Do(func(user *server.User) {
						Expect(user.Username).To(Equal("ginkgo.example.net"))
						Expect(user.Account).To(Equal(auth.choriaAccount))
						Expect(user.Permissions.Publish).To(Equal(&server.SubjectPermission{
							Allow: []string{
								"choria.lifecycle.>",
								"choria.machine.transition",
								"choria.machine.watcher.>",
								"c1.reply.>",
								"c1.broadcast.agent.registration",
								"choria.federation.c1.collective",
								"c1.governor.*",
								"c2.reply.>",
								"c2.broadcast.agent.registration",
								"choria.federation.c2.collective",
								"c2.governor.*",
								"$JS.API.STREAM.INFO.*",
								"$JS.API.STREAM.MSG.GET.*",
								"$JS.API.STREAM.MSG.DELETE.*",
								"$JS.API.DIRECT.GET.*",
								"$JS.API.DIRECT.GET.*.>",
								"$JS.API.CONSUMER.CREATE.*",
								"$JS.API.CONSUMER.CREATE.*.>",
								"$JS.API.CONSUMER.DURABLE.CREATE.*.*",
								"$JS.API.CONSUMER.INFO.*.*",
								"$JS.API.CONSUMER.MSG.NEXT.*.*",
								"$JS.ACK.>",
								"$JS.FC.>",
							},
						}))
					})

					verified, err := auth.handleDefaultConnection(mockClient, verifiedConn, true, log)
					Expect(err).ToNot(HaveOccurred())
					Expect(verified).To(BeTrue())
				})

				It("Should support Submission", func() {
					copts.Token = createSignedServerJWT(privateKey, edPublicKey, map[string]any{
						"purpose":     tokens.ServerPurpose,
						"public_key":  hex.EncodeToString(edPublicKey),
						"collectives": []string{"c1", "c2"},
						"permissions": &tokens.ServerPermissions{Submission: true},
					})

					mockClient.EXPECT().RegisterUser(gomock.Any()).Do(func(user *server.User) {
						Expect(user.Username).To(Equal("ginkgo.example.net"))
						Expect(user.Account).To(Equal(auth.choriaAccount))
						Expect(user.Permissions.Publish).To(Equal(&server.SubjectPermission{
							Allow: []string{
								"choria.lifecycle.>",
								"choria.machine.transition",
								"choria.machine.watcher.>",
								"c1.reply.>",
								"c1.broadcast.agent.registration",
								"choria.federation.c1.collective",
								"c1.submission.in.>",
								"c2.reply.>",
								"c2.broadcast.agent.registration",
								"choria.federation.c2.collective",
								"c2.submission.in.>",
							},
						}))
					})

					verified, err := auth.handleDefaultConnection(mockClient, verifiedConn, true, log)
					Expect(err).ToNot(HaveOccurred())
					Expect(verified).To(BeTrue())
				})

				Describe("Should support Streams", func() {
					It("Should support Streams in the choria org", func() {
						copts.Token = createSignedServerJWT(privateKey, edPublicKey, map[string]any{
							"purpose":     tokens.ServerPurpose,
							"public_key":  hex.EncodeToString(edPublicKey),
							"collectives": []string{"c1", "c2"},
							"permissions": &tokens.ServerPermissions{Streams: true},
						})

						mockClient.EXPECT().RegisterUser(gomock.Any()).Do(func(user *server.User) {
							Expect(user.Username).To(Equal("ginkgo.example.net"))
							Expect(user.Account).To(Equal(auth.choriaAccount))
							Expect(user.Permissions.Publish).To(Equal(&server.SubjectPermission{
								Allow: []string{
									"choria.lifecycle.>",
									"choria.machine.transition",
									"choria.machine.watcher.>",
									"c1.reply.>",
									"c1.broadcast.agent.registration",
									"choria.federation.c1.collective",
									"c2.reply.>",
									"c2.broadcast.agent.registration",
									"choria.federation.c2.collective",
									"$JS.API.STREAM.INFO.*",
									"$JS.API.STREAM.MSG.GET.*",
									"$JS.API.STREAM.MSG.DELETE.*",
									"$JS.API.DIRECT.GET.*",
									"$JS.API.DIRECT.GET.*.>",
									"$JS.API.CONSUMER.CREATE.*",
									"$JS.API.CONSUMER.CREATE.*.>",
									"$JS.API.CONSUMER.DURABLE.CREATE.*.*",
									"$JS.API.CONSUMER.INFO.*.*",
									"$JS.API.CONSUMER.MSG.NEXT.*.*",
									"$JS.ACK.>",
									"$JS.FC.>",
								},
							}))
						})

						verified, err := auth.handleDefaultConnection(mockClient, verifiedConn, true, log)
						Expect(err).ToNot(HaveOccurred())
						Expect(verified).To(BeTrue())
					})
					It("Should support Streams in other orgs", func() {
						copts.Token = createSignedServerJWT(privateKey, edPublicKey, map[string]any{
							"purpose":     tokens.ServerPurpose,
							"public_key":  hex.EncodeToString(edPublicKey),
							"collectives": []string{"c1", "c2"},
							"ou":          "other",
							"permissions": &tokens.ServerPermissions{Streams: true},
						})

						mockClient.EXPECT().RegisterUser(gomock.Any()).Do(func(user *server.User) {
							Expect(user.Username).To(Equal("ginkgo.example.net"))
							Expect(user.Account).To(Equal(auth.choriaAccount))
							Expect(user.Permissions.Publish).To(Equal(&server.SubjectPermission{
								Allow: []string{
									"choria.lifecycle.>",
									"choria.machine.transition",
									"choria.machine.watcher.>",
									"c1.reply.>",
									"c1.broadcast.agent.registration",
									"choria.federation.c1.collective",
									"c2.reply.>",
									"c2.broadcast.agent.registration",
									"choria.federation.c2.collective",
									"choria.streams.STREAM.INFO.*",
									"choria.streams.STREAM.MSG.GET.*",
									"choria.streams.STREAM.MSG.DELETE.*",
									"choria.streams.DIRECT.GET.*",
									"choria.streams.DIRECT.GET.*.>",
									"choria.streams.CONSUMER.CREATE.*",
									"choria.streams.CONSUMER.CREATE.*.>",
									"choria.streams.CONSUMER.DURABLE.CREATE.*.*",
									"choria.streams.CONSUMER.INFO.*.*",
									"choria.streams.CONSUMER.MSG.NEXT.*.*",
									"$JS.ACK.>",
									"$JS.FC.>",
								},
							}))
						})

						verified, err := auth.handleDefaultConnection(mockClient, verifiedConn, true, log)
						Expect(err).ToNot(HaveOccurred())
						Expect(verified).To(BeTrue())
					})
				})

				It("Should support additional subjects", func() {
					copts.Token = createSignedServerJWT(privateKey, edPublicKey, map[string]any{
						"purpose":      tokens.ServerPurpose,
						"public_key":   hex.EncodeToString(edPublicKey),
						"collectives":  []string{"c1", "c2"},
						"pub_subjects": []string{"other", "subject"},
					})

					mockClient.EXPECT().RegisterUser(gomock.Any()).Do(func(user *server.User) {
						Expect(user.Username).To(Equal("ginkgo.example.net"))
						Expect(user.Account).To(Equal(auth.choriaAccount))
						Expect(user.Permissions.Publish).To(Equal(&server.SubjectPermission{
							Allow: []string{
								"choria.lifecycle.>",
								"choria.machine.transition",
								"choria.machine.watcher.>",
								"other",
								"subject",
								"c1.reply.>",
								"c1.broadcast.agent.registration",
								"choria.federation.c1.collective",
								"c2.reply.>",
								"c2.broadcast.agent.registration",
								"choria.federation.c2.collective",
							},
						}))
					})

					verified, err := auth.handleDefaultConnection(mockClient, verifiedConn, true, log)
					Expect(err).ToNot(HaveOccurred())
					Expect(verified).To(BeTrue())

				})
			})
		})

		Describe("Clients", func() {
			BeforeEach(func() {
				auth.clientJwtSigners = []string{filepath.Join(td, "public.pem")}
				copts = &server.ClientOpts{
					Token: createSignedClientJWT(privateKey, map[string]any{
						"purpose":    tokens.ClientIDPurpose,
						"public_key": hex.EncodeToString(edPublicKey),
					}),
				}
				verifiedConn = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{nil}}
				mockClient.EXPECT().GetOpts().Return(copts).AnyTimes()
				mockClient.EXPECT().Kind().Return(server.CLIENT).AnyTimes()
			})

			It("Should require a remote", func() {
				_, err := auth.verifyClientJWTBasedAuth(nil, "", nil, "", log)
				Expect(err).To(MatchError("remote client information is required in anonymous TLS or JWT signing modes"))
			})

			It("Should fail on invalid jwt", func() {
				_, err := auth.verifyClientJWTBasedAuth(&net.IPAddr{IP: net.ParseIP("192.168.0.1"), Zone: ""}, "x", nil, "", log)
				Expect(err).To(MatchError("invalid JWT token"))
			})

			It("Should fail for invalid nonce", func() {
				copts.Sig = "wrong"
				mockClient.EXPECT().RemoteAddress().Return(&net.IPAddr{IP: net.ParseIP("192.168.0.1"), Zone: ""})
				mockClient.EXPECT().GetNonce().Return([]byte("toomanysecrets"))

				verified, err := auth.handleDefaultConnection(mockClient, verifiedConn, true, log)
				Expect(err).To(MatchError("invalid nonce signature or jwt token"))
				Expect(verified).To(BeFalse())
			})

			It("Should set strict permissions for a client JWT user", func() {
				sig, err := choria.Ed25519Sign(edPrivateKey, []byte("toomanysecrets"))
				Expect(err).ToNot(HaveOccurred())
				copts.Sig = base64.RawURLEncoding.EncodeToString(sig)

				mockClient.EXPECT().RemoteAddress().Return(&net.IPAddr{IP: net.ParseIP("192.168.0.1"), Zone: ""})
				mockClient.EXPECT().GetNonce().Return([]byte("toomanysecrets"))
				mockClient.EXPECT().RegisterUser(gomock.Any()).Do(func(user *server.User) {
					Expect(user.Username).To(Equal("up=ginkgo"))
					Expect(user.Account).To(Equal(auth.choriaAccount))
					Expect(user.Permissions.Subscribe).To(Equal(&server.SubjectPermission{
						Allow: []string{"*.reply.e33bf0376d4accbb4a8fd24b2f840b2e.>"},
					}))
					Expect(user.Permissions.Publish).To(Equal(&server.SubjectPermission{}))
				})

				verified, err := auth.handleDefaultConnection(mockClient, verifiedConn, true, log)
				Expect(err).ToNot(HaveOccurred())
				Expect(verified).To(BeTrue())
			})

			It("Should register other clients without restriction", func() {
				mockClient.GetOpts().Token = ""
				mockClient.EXPECT().RemoteAddress().Return(&net.IPAddr{IP: net.ParseIP("192.168.0.1"), Zone: ""})
				mockClient.EXPECT().GetNonce().Return(nil)
				mockClient.EXPECT().RegisterUser(gomock.Any()).Do(func(user *server.User) {
					Expect(user.Username).To(BeEmpty())
					Expect(user.Account).To(Equal(auth.choriaAccount))
					Expect(user.Permissions.Subscribe).To(BeNil())
					Expect(user.Permissions.Publish).To(BeNil())
				})

				verified, err := auth.handleDefaultConnection(mockClient, verifiedConn, true, log)
				Expect(err).ToNot(HaveOccurred())
				Expect(verified).To(BeTrue())
			})
		})

		Describe("verifyNonceSignature", func() {
			It("Should fail when no signature is given", func() {
				ok, err := auth.verifyNonceSignature(nil, "", "", log)
				Expect(ok).To(BeFalse())
				Expect(err).To(MatchError("connection nonce was not signed"))
			})

			It("Should fail when no public key is in the jwt", func() {
				ok, err := auth.verifyNonceSignature(nil, "x", "", log)
				Expect(ok).To(BeFalse())
				Expect(err).To(MatchError("no public key found in the JWT to verify nonce signature"))
			})

			It("Should fail when the server did not set a nonce", func() {
				ok, err := auth.verifyNonceSignature(nil, "x", "x", log)
				Expect(ok).To(BeFalse())
				Expect(err).To(MatchError("server did not generate a nonce to verify"))
			})

			It("Should fail for invalid nonce signatures", func() {
				ok, err := auth.verifyNonceSignature([]byte("toomanysecrets"), "x", hex.EncodeToString(edPublicKey), log)
				Expect(ok).To(BeFalse())
				Expect(err).To(MatchError("invalid url encoded signature: illegal base64 data at input byte 0"))
			})

			It("Should not panic for invalid length public keys", func() {
				nonce := []byte("toomanysecrets")

				sig, err := choria.Ed25519Sign(edPrivateKey, nonce)
				Expect(err).ToNot(HaveOccurred())
				Expect(sig).To(HaveLen(64))

				ok, err := auth.verifyNonceSignature(nonce, base64.RawURLEncoding.EncodeToString(sig), hex.EncodeToString([]byte(hex.EncodeToString(edPublicKey))), log)
				Expect(err).To(MatchError("could not verify nonce signature: invalid public key length 64"))
				Expect(ok).To(BeFalse())
			})

			It("Should pass correct signatures", func() {
				nonce := []byte("toomanysecrets")

				sig, err := choria.Ed25519Sign(edPrivateKey, nonce)
				Expect(err).ToNot(HaveOccurred())
				Expect(sig).To(HaveLen(64))

				ok, err := auth.verifyNonceSignature(nonce, base64.RawURLEncoding.EncodeToString(sig), hex.EncodeToString(edPublicKey), log)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
	})

	Describe("handleVerifiedSystemAccount", func() {
		It("Should fail without a password", func() {
			auth.systemUser = ""
			auth.systemPass = ""

			verified, err := auth.handleVerifiedSystemAccount(mockClient, log)
			Expect(err).To(MatchError("system user is required"))
			Expect(verified).To(BeFalse())

			auth.systemUser = "system"
			verified, err = auth.handleVerifiedSystemAccount(mockClient, log)
			Expect(err).To(MatchError("system password is required"))
			Expect(verified).To(BeFalse())
		})

		It("Should fail without an account", func() {
			auth.systemUser = "system"
			auth.systemPass = "s3cret"

			verified, err := auth.handleVerifiedSystemAccount(mockClient, log)
			Expect(err).To(MatchError("system account is not set"))
			Expect(verified).To(BeFalse())
		})

		It("Should verify the password", func() {
			auth.systemUser = "system"
			auth.systemPass = "other"
			auth.systemAccount = &server.Account{Name: "system"}

			mockClient.EXPECT().GetOpts().Return(&server.ClientOpts{Username: "system", Password: "s3cret"}).AnyTimes()
			verified, err := auth.handleVerifiedSystemAccount(mockClient, log)
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

			verified, err := auth.handleVerifiedSystemAccount(mockClient, log)
			Expect(err).ToNot(HaveOccurred())
			Expect(verified).To(BeTrue())
		})
	})

	Describe("handleProvisioningUserConnection", func() {
		It("Should fail without a password", func() {
			auth.provPass = ""

			verified, err := auth.handleProvisioningUserConnection(mockClient, true)
			Expect(err).To(MatchError("provisioning user password not enabled"))
			Expect(verified).To(BeFalse())
		})

		It("Should fail without an account", func() {
			auth.provPass = "s3cret"

			verified, err := auth.handleProvisioningUserConnection(mockClient, true)
			Expect(err).To(MatchError("provisioning account is not set"))
			Expect(verified).To(BeFalse())
		})

		Context("Using Issuers", func() {
			var td string
			var err error
			var issuerPubk ed25519.PublicKey
			var issuerPrik ed25519.PrivateKey

			BeforeEach(func() {
				td, err = os.MkdirTemp("", "")
				Expect(err).ToNot(HaveOccurred())

				issuerPubk, issuerPrik, err = iu.Ed25519KeyPair()
				Expect(err).ToNot(HaveOccurred())

				auth.issuerTokens = map[string]string{"choria": hex.EncodeToString(issuerPubk)}

				DeferCleanup(func() {
					os.RemoveAll(td)
				})
			})

			It("Should require a token", func() {
				auth.provPass = "s3cret"
				auth.provisioningAccount = &server.Account{Name: provisioningUser}

				mockClient.EXPECT().GetOpts().Return(&server.ClientOpts{Username: provisioningUser, Password: "s3cret"}).AnyTimes()

				verified, err := auth.handleProvisioningUserConnection(mockClient, true)
				Expect(err).To(MatchError("no token provided in connection"))
				Expect(verified).To(BeFalse())
			})

			It("Should handle tokens that do not pass validation", func() {
				auth.provPass = "s3cret"
				auth.provisioningAccount = &server.Account{Name: provisioningUser}

				provPubk, _, err := iu.Ed25519KeyPair()
				Expect(err).ToNot(HaveOccurred())

				provClaims, err := tokens.NewClientIDClaims("provisioner", nil, "choria", nil, "", "", time.Hour, nil, provPubk)
				Expect(err).ToNot(HaveOccurred())

				// make sure its expired
				provClaims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Hour))
				signed, err := tokens.SignToken(provClaims, issuerPrik)
				Expect(err).ToNot(HaveOccurred())

				mockClient.EXPECT().GetOpts().Return(&server.ClientOpts{Username: provisioningUser, Password: "s3cret", Token: signed}).AnyTimes()

				verified, err := auth.handleProvisioningUserConnection(mockClient, true)
				Expect(err.Error()).To(MatchRegexp("token is expired by 1h"))
				Expect(verified).To(BeFalse())
			})

			It("Should require the provisioner permission", func() {
				auth.provPass = "s3cret"
				auth.provisioningAccount = &server.Account{Name: provisioningUser}

				provPubk, _, err := iu.Ed25519KeyPair()
				Expect(err).ToNot(HaveOccurred())

				provClaims, err := tokens.NewClientIDClaims("provisioner", nil, "choria", nil, "", "", time.Hour, nil, provPubk)
				Expect(err).ToNot(HaveOccurred())
				signed, err := tokens.SignToken(provClaims, issuerPrik)
				Expect(err).ToNot(HaveOccurred())

				mockClient.EXPECT().GetOpts().Return(&server.ClientOpts{Username: provisioningUser, Password: "s3cret", Token: signed}).AnyTimes()

				verified, err := auth.handleProvisioningUserConnection(mockClient, true)
				Expect(err).To(MatchError("provisioner claim is false in token with caller id 'provisioner'"))
				Expect(verified).To(BeFalse())
			})

			It("Should correctly verify the password and register the user", func() {
				auth.provPass = "s3cret"
				auth.provisioningAccount = &server.Account{Name: provisioningUser}

				provPubk, _, err := iu.Ed25519KeyPair()
				Expect(err).ToNot(HaveOccurred())

				provClaims, err := tokens.NewClientIDClaims("provisioner", nil, "choria", nil, "", "", time.Hour, &tokens.ClientPermissions{ServerProvisioner: true}, provPubk)
				Expect(err).ToNot(HaveOccurred())
				signed, err := tokens.SignToken(provClaims, issuerPrik)
				Expect(err).ToNot(HaveOccurred())

				mockClient.EXPECT().GetOpts().Return(&server.ClientOpts{Username: provisioningUser, Password: "s3cret", Token: signed}).AnyTimes()
				mockClient.EXPECT().RegisterUser(gomock.Any()).Do(func(user *server.User) {
					Expect(user.Username).To(Equal(provisioningUser))
					Expect(user.Password).To(Equal("s3cret"))
					Expect(user.Account).To(Equal(auth.provisioningAccount))
					Expect(user.Permissions).To(Not(BeNil()))
					Expect(user.Permissions.Publish).To(BeNil())
					Expect(user.Permissions.Subscribe).To(BeNil())
					Expect(user.Permissions.Response).To(BeNil())
				})

				verified, err := auth.handleProvisioningUserConnection(mockClient, true)
				Expect(err).ToNot(HaveOccurred())
				Expect(verified).To(BeTrue())
			})
		})

		Context("Using mTLS", func() {
			It("Should fail when server not in TLS mode", func() {
				auth.provPass = "s3cret"
				auth.isTLS = false
				auth.provisioningAccount = &server.Account{Name: provisioningUser}

				verified, err := auth.handleProvisioningUserConnection(mockClient, true)
				Expect(err).To(MatchError("provisioning user access requires TLS"))
				Expect(verified).To(BeFalse())
			})

			It("Should fail when client is not using TLS", func() {
				auth.provPass = "s3cret"
				auth.isTLS = true
				auth.provisioningAccount = &server.Account{Name: provisioningUser}
				mockClient.EXPECT().GetTLSConnectionState().Return(nil).AnyTimes()

				verified, err := auth.handleProvisioningUserConnection(mockClient, false)
				Expect(err).To(MatchError("provisioning user is only allowed over verified TLS connections"))
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

				verified, err := auth.handleProvisioningUserConnection(mockClient, true)
				Expect(err).ToNot(HaveOccurred())
				Expect(verified).To(BeTrue())
			})
		})
	})

	Describe("handleUnverifiedProvisioningConnection", func() {
		Describe("Provisioner Client", func() {
			It("Should not accept connections from the provisioning user without verified TLS", func() {
				auth.provisioningTokenSigner = "testdata/ssl/certs/rip.mcollective.pem"
				auth.provisioningAccount = &server.Account{Name: "provisioning"}

				copts := &server.ClientOpts{Username: "provisioner", Password: "s3cret"}
				mockClient.EXPECT().GetOpts().Return(copts).AnyTimes()

				validated, err := auth.handleUnverifiedProvisioningConnection(mockClient)
				Expect(err).To(MatchError("provisioning user requires a verified connection"))
				Expect(validated).To(BeFalse())
			})
		})

		Context("Org Issuers", func() {
			var (
				issuerPubk                    ed25519.PublicKey
				issuerPrik                    ed25519.PrivateKey
				valid, invalid, wrongOU, noOU string
				err                           error
			)

			BeforeEach(func() {
				issuerPubk, issuerPrik, err = iu.Ed25519KeyPair()
				Expect(err).ToNot(HaveOccurred())

				td, err := os.MkdirTemp("", "")
				Expect(err).ToNot(HaveOccurred())
				DeferCleanup(func() { os.RemoveAll(td) })

				token, err := tokens.NewProvisioningClaims(false, true, "s3cret", "", "", nil, "example.net", "", "", "choria", "xxx", time.Hour)
				Expect(err).ToNot(HaveOccurred())
				valid, err = tokens.SignToken(token, issuerPrik)
				Expect(err).ToNot(HaveOccurred())

				token, err = tokens.NewProvisioningClaims(false, true, "s3cret", "", "", nil, "example.net", "", "", "choria", "xxx", time.Hour)
				Expect(err).ToNot(HaveOccurred())
				token.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-1 * time.Hour))
				invalid, err = tokens.SignToken(token, issuerPrik)
				Expect(err).ToNot(HaveOccurred())

				token, err = tokens.NewProvisioningClaims(false, true, "s3cret", "", "", nil, "example.net", "", "", "other", "xxx", time.Hour)
				Expect(err).ToNot(HaveOccurred())
				wrongOU, err = tokens.SignToken(token, issuerPrik)
				Expect(err).ToNot(HaveOccurred())

				token, err = tokens.NewProvisioningClaims(false, true, "s3cret", "", "", nil, "example.net", "", "", "other", "xxx", time.Hour)
				token.OrganizationUnit = ""
				Expect(err).ToNot(HaveOccurred())
				noOU, err = tokens.SignToken(token, issuerPrik)
				Expect(err).ToNot(HaveOccurred())

				auth.issuerTokens = map[string]string{"choria": hex.EncodeToString(issuerPubk)}
				auth.provPass = "s3cret"
			})

			It("Should fail without a provisioner account", func() {
				mockClient.EXPECT().GetOpts().Return(&server.ClientOpts{}).AnyTimes()

				validated, err := auth.handleUnverifiedProvisioningConnection(mockClient)
				Expect(validated).To(BeFalse())
				Expect(err).To(MatchError("provisioning account is not set"))
			})

			Describe("Servers", func() {
				BeforeEach(func() {
					auth.provisioningAccount = &server.Account{Name: "provisioning"}
				})

				It("Should fail for invalid tokens", func() {
					mockClient.EXPECT().GetOpts().Return(&server.ClientOpts{Token: invalid}).AnyTimes()

					validated, err := auth.handleUnverifiedProvisioningConnection(mockClient)
					Expect(validated).To(BeFalse())
					Expect(err.Error()).To(MatchRegexp("token is expired by"))
				})

				It("Should detect missing ou claims", func() {
					mockClient.EXPECT().GetOpts().Return(&server.ClientOpts{Token: noOU}).AnyTimes()

					validated, err := auth.handleUnverifiedProvisioningConnection(mockClient)
					Expect(validated).To(BeFalse())
					Expect(err.Error()).To(MatchRegexp("no ou claim in token"))
				})

				It("Should detect unconfigured Issuers", func() {
					mockClient.EXPECT().GetOpts().Return(&server.ClientOpts{Token: wrongOU}).AnyTimes()

					validated, err := auth.handleUnverifiedProvisioningConnection(mockClient)
					Expect(validated).To(BeFalse())
					Expect(err.Error()).To(MatchRegexp("no issuer found for ou other"))
				})

				It("Should set server permissions and register", func() {
					mockClient.EXPECT().GetOpts().Return(&server.ClientOpts{Token: valid}).AnyTimes()
					mockClient.EXPECT().RemoteAddress().Return(&net.IPAddr{IP: net.ParseIP("192.168.0.1"), Zone: ""})
					mockClient.EXPECT().RegisterUser(gomock.Any()).Do(func(user *server.User) {
						Expect(user.Username).To(BeEmpty())
						Expect(user.Password).To(BeEmpty())
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
		})

		Context("mTLS", func() {
			It("Should fail without a signer cert set or present", func() {
				mockClient.EXPECT().GetOpts().Return(&server.ClientOpts{}).AnyTimes()

				validated, err := auth.handleUnverifiedProvisioningConnection(mockClient)
				Expect(validated).To(BeFalse())
				Expect(err).To(MatchError("provisioning is not enabled"))

				auth.provisioningTokenSigner = "/nonexisting"
				validated, err = auth.handleUnverifiedProvisioningConnection(mockClient)
				Expect(validated).To(BeFalse())
				Expect(err).To(MatchError("provisioning signer certificate /nonexisting does not exist"))
			})

			It("Should fail without a provisioner account", func() {
				mockClient.EXPECT().GetOpts().Return(&server.ClientOpts{}).AnyTimes()

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
						Expect(user.Username).To(BeEmpty())
						Expect(user.Password).To(BeEmpty())
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

	Describe("parseServerJWT", func() {
		It("Should fail without a cert", func() {
			_, err := auth.parseServerJWT("")
			Expect(err).To(MatchError("no Server JWT signer or Organization Issuer set, denying all servers"))
		})

		It("Should fail for empty JWTs", func() {
			auth.serverJwtSigners = []string{"testdata/public.pem"}
			_, err := auth.parseServerJWT("")
			Expect(err).To(MatchError("no JWT received"))
		})

		Describe("Issuers", func() {
			var (
				issuerPubk, serverPubk ed25519.PublicKey
				issuerPrik             ed25519.PrivateKey
				err                    error
			)

			BeforeEach(func() {
				issuerPubk, issuerPrik, err = iu.Ed25519KeyPair()
				Expect(err).ToNot(HaveOccurred())
				serverPubk, _, err = iu.Ed25519KeyPair()
				Expect(err).ToNot(HaveOccurred())

				auth.issuerTokens = map[string]string{"choria": hex.EncodeToString(issuerPubk)}
			})

			It("Should detect missing ou claims", func() {
				signed := createSignedServerJWT(issuerPrik, serverPubk, map[string]any{
					"ou": nil,
				})

				_, err = auth.parseServerJWT(signed)
				Expect(err.Error()).To(MatchRegexp("no ou claim in token"))
			})

			It("Should detect unconfigured Issuers", func() {
				signed := createSignedServerJWT(issuerPrik, serverPubk, map[string]any{
					"ou": "other",
				})

				_, err = auth.parseServerJWT(signed)
				Expect(err.Error()).To(MatchRegexp("no issuer found for ou other"))
			})

			It("Should parse the token and handle failures", func() {
				signed := createSignedClientJWT(issuerPrik, map[string]any{
					"ou": "choria",
				})

				_, err := auth.parseServerJWT(signed)
				Expect(err).To(MatchError("failed to parse token issued by the choria chain: not a server token"))
			})

			It("Should handle valid tokens issued by a chain issuer", func() {
				// this is for provisioner signing servers
				chainIssuerPubk, chainIssuerPrik, err := iu.Ed25519KeyPair()
				Expect(err).ToNot(HaveOccurred())
				chainIssuer, err := tokens.NewClientIDClaims("chain_issuer", nil, "choria", nil, "", "", time.Hour, nil, chainIssuerPubk)
				Expect(err).ToNot(HaveOccurred())
				Expect(chainIssuer.AddOrgIssuerData(issuerPrik)).To(Succeed())

				serverPubk, _, err := iu.Ed25519KeyPair()
				Expect(err).ToNot(HaveOccurred())
				server, err := tokens.NewServerClaims("ginkgo.example.net", []string{"choria"}, "choria", nil, nil, serverPubk, "", time.Hour)
				Expect(err).ToNot(HaveOccurred())
				Expect(server.AddChainIssuerData(chainIssuer, chainIssuerPrik)).To(Succeed())
				signed, err := tokens.SignToken(server, chainIssuerPrik)
				Expect(err).ToNot(HaveOccurred())

				claims, err := auth.parseServerJWT(signed)
				Expect(err).ToNot(HaveOccurred())
				Expect(claims.ChoriaIdentity).To(Equal("ginkgo.example.net"))
			})

			It("Should handle valid tokens issued by the org issuer", func() {
				pubk, _, err := iu.Ed25519KeyPair()
				Expect(err).ToNot(HaveOccurred())
				server, err := tokens.NewServerClaims("ginkgo.example.net", []string{"choria"}, "choria", nil, nil, pubk, "", time.Hour)
				Expect(err).ToNot(HaveOccurred())
				signed, err := tokens.SignToken(server, issuerPrik)
				Expect(err).ToNot(HaveOccurred())

				claims, err := auth.parseServerJWT(signed)
				Expect(err).ToNot(HaveOccurred())
				Expect(claims.ChoriaIdentity).To(Equal("ginkgo.example.net"))
			})
		})

		Describe("Trusted Signers", func() {
			It("Should verify JWTs", func() {
				edPublicKey, edPriKey, err := choria.Ed25519KeyPair()
				Expect(err).ToNot(HaveOccurred())

				auth.serverJwtSigners = []string{hex.EncodeToString(edPublicKey)}
				signed := createSignedClientJWT(edPriKey, map[string]any{
					"exp": time.Now().UTC().Add(-time.Hour).Unix(),
				})

				_, err = auth.parseServerJWT(signed)
				Expect(err).To(MatchError(jwt.ErrTokenExpired))
			})

			It("Should check a purpose field", func() {
				edPublicKey, edPriKey, err := choria.Ed25519KeyPair()
				Expect(err).ToNot(HaveOccurred())
				auth.serverJwtSigners = []string{hex.EncodeToString(edPublicKey)}

				signed := createSignedClientJWT(edPriKey, nil)
				_, err = auth.parseServerJWT(signed)
				Expect(err).To(MatchError(tokens.ErrNotAServerToken))

				signed = createSignedClientJWT(edPriKey, map[string]any{
					"purpose": "wrong",
				})
				_, err = auth.parseServerJWT(signed)
				Expect(err).To(MatchError(tokens.ErrNotAServerToken))
			})

			It("Should check the identity", func() {
				edPublicKey, edPriKey, err := choria.Ed25519KeyPair()
				Expect(err).ToNot(HaveOccurred())
				auth.serverJwtSigners = []string{hex.EncodeToString(edPublicKey)}

				signed := createSignedClientJWT(edPriKey, map[string]any{
					"purpose": tokens.ServerPurpose,
				})
				_, err = auth.parseServerJWT(signed)
				Expect(err).To(MatchError("identity not in claims"))
			})

			It("Should check the public key", func() {
				edPublicKey, edPriKey, err := choria.Ed25519KeyPair()
				Expect(err).ToNot(HaveOccurred())
				auth.serverJwtSigners = []string{hex.EncodeToString(edPublicKey)}

				signed := createSignedClientJWT(edPriKey, map[string]any{
					"purpose":  tokens.ServerPurpose,
					"identity": "ginkgo.example.net",
				})
				_, err = auth.parseServerJWT(signed)
				Expect(err).To(MatchError("no public key in claims"))
			})

			It("Should handle multiple public identifiers", func() {
				edPublicKey, edPriKey, err := choria.Ed25519KeyPair()
				Expect(err).ToNot(HaveOccurred())
				auth.serverJwtSigners = []string{"/nonexisting", hex.EncodeToString(edPublicKey)}

				signed := createSignedServerJWT(edPriKey, edPublicKey, map[string]any{
					"purpose":    tokens.ServerPurpose,
					"identity":   "ginkgo.example.net",
					"public_key": hex.EncodeToString(edPublicKey),
				})

				_, err = auth.parseServerJWT(signed)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("parseClientIDJWT", func() {
		var td string
		var privateKey *rsa.PrivateKey

		BeforeEach(func() {
			td, privateKey = createKeyPair()
		})

		AfterEach(func() {
			os.RemoveAll(td)
		})

		It("Should fail without a cert", func() {
			_, err := auth.parseClientIDJWT("")
			Expect(err).To(MatchError("no Client JWT signer or Organization Issuer set, denying all clients"))
		})

		It("Should fail for empty JWTs", func() {
			auth.clientJwtSigners = []string{"testdata/public.pem"}
			_, err := auth.parseClientIDJWT("")
			Expect(err).To(MatchError("no JWT received"))
		})

		Describe("Issuers", func() {
			var (
				issuerPubk ed25519.PublicKey
				issuerPrik ed25519.PrivateKey
				err        error
			)

			BeforeEach(func() {
				issuerPubk, issuerPrik, err = iu.Ed25519KeyPair()
				Expect(err).ToNot(HaveOccurred())
				auth.issuerTokens = map[string]string{"choria": hex.EncodeToString(issuerPubk)}
			})

			It("Should detect missing ou claims", func() {
				signed := createSignedClientJWT(privateKey, nil)

				_, err := auth.parseClientIDJWT(signed)
				Expect(err.Error()).To(MatchRegexp("no ou claim in token"))
			})

			It("Should detect unconfigured Issuers", func() {
				signed := createSignedClientJWT(privateKey, map[string]any{
					"exp": time.Now().UTC().Add(-time.Hour).Unix(),
					"ou":  "other",
				})

				_, err := auth.parseClientIDJWT(signed)
				Expect(err.Error()).To(MatchRegexp("no issuer configured for ou 'other'"))
			})

			It("Should parse the token and handle failures", func() {
				pubk, _, err := iu.Ed25519KeyPair()
				Expect(err).ToNot(HaveOccurred())

				signed := createSignedServerJWT(issuerPrik, pubk, map[string]any{
					"ou": "choria",
				})
				_, err = auth.parseClientIDJWT(signed)
				Expect(err.Error()).To(MatchRegexp("failed to parse client token issued by the choria chain: not a client token"))
			})

			It("Should handle valid tokens issued by a chain issuer", func() {
				chainIssuerPubk, chainIssuerPrik, err := iu.Ed25519KeyPair()
				Expect(err).ToNot(HaveOccurred())
				chainIssuer, err := tokens.NewClientIDClaims("chain_issuer", nil, "choria", nil, "", "", time.Hour, nil, chainIssuerPubk)
				Expect(err).ToNot(HaveOccurred())
				Expect(chainIssuer.AddOrgIssuerData(issuerPrik)).To(Succeed())

				clientPubk, _, err := iu.Ed25519KeyPair()
				Expect(err).ToNot(HaveOccurred())
				client, err := tokens.NewClientIDClaims("ginkgo", nil, "choria", nil, "", "", time.Hour, nil, clientPubk)
				Expect(err).ToNot(HaveOccurred())
				Expect(client.AddChainIssuerData(chainIssuer, chainIssuerPrik)).To(Succeed())
				signed, err := tokens.SignToken(client, chainIssuerPrik)
				Expect(err).ToNot(HaveOccurred())

				claims, err := auth.parseClientIDJWT(signed)
				Expect(err).ToNot(HaveOccurred())
				Expect(claims.CallerID).To(Equal("ginkgo"))
			})

			It("Should handle valid tokens issued by the org issuer", func() {
				pubk, _, err := iu.Ed25519KeyPair()
				Expect(err).ToNot(HaveOccurred())
				client, err := tokens.NewClientIDClaims("ginkgo", nil, "choria", nil, "", "", time.Hour, nil, pubk)
				Expect(err).ToNot(HaveOccurred())
				// Expect(client.AddOrgIssuerData(issuerPrik)).To(Succeed())

				signed, err := tokens.SignToken(client, issuerPrik)
				Expect(err).ToNot(HaveOccurred())

				claims, err := auth.parseClientIDJWT(signed)
				Expect(err).ToNot(HaveOccurred())
				Expect(claims.CallerID).To(Equal("ginkgo"))
			})
		})

		Describe("Trusted Signers", func() {
			It("Should verify JWTs", func() {
				auth.clientJwtSigners = []string{filepath.Join(td, "public.pem")}
				signed := createSignedClientJWT(privateKey, map[string]any{
					"exp": time.Now().UTC().Add(-time.Hour).Unix(),
				})

				_, err := auth.parseClientIDJWT(signed)
				Expect(err.Error()).To(MatchRegexp("token is expired by"))
			})

			It("Should detect missing callers", func() {
				auth.clientJwtSigners = []string{filepath.Join(td, "public.pem")}
				signed := createSignedClientJWT(privateKey, map[string]any{
					"callerid": "",
					"purpose":  tokens.ClientIDPurpose,
				})

				_, err := auth.parseClientIDJWT(signed)
				Expect(err).To(MatchError("no callerid in claims"))
			})

			It("Should check the purpose field", func() {
				auth.clientJwtSigners = []string{filepath.Join(td, "public.pem")}
				signed := createSignedClientJWT(privateKey, nil)
				_, err := auth.parseClientIDJWT(signed)
				Expect(err).To(MatchError(tokens.ErrNotAClientToken))

				signed = createSignedClientJWT(privateKey, map[string]any{
					"purpose": "wrong",
				})
				_, err = auth.parseClientIDJWT(signed)
				Expect(err).To(MatchError(tokens.ErrNotAClientToken))
			})

			It("Should check the caller", func() {
				edPublicKey, _, err := choria.Ed25519KeyPair()
				Expect(err).ToNot(HaveOccurred())

				auth.clientJwtSigners = []string{filepath.Join(td, "public.pem")}
				signed := createSignedClientJWT(privateKey, map[string]any{
					"purpose":    tokens.ClientIDPurpose,
					"public_key": hex.EncodeToString(edPublicKey),
				})

				claims, err := auth.parseClientIDJWT(signed)
				Expect(err).ToNot(HaveOccurred())
				Expect(claims.CallerID).To(Equal("up=ginkgo"))
			})

			It("Should check the public key", func() {
				auth.clientJwtSigners = []string{filepath.Join(td, "public.pem")}
				signed := createSignedClientJWT(privateKey, map[string]any{
					"purpose": tokens.ClientIDPurpose,
				})

				claims, err := auth.parseClientIDJWT(signed)
				Expect(err).To(MatchError("no public key in claims"))
				Expect(claims).To(BeNil())
			})

			It("Should handle multiple public identifiers", func() {
				edPublicKey, edPriKey, err := choria.Ed25519KeyPair()
				Expect(err).ToNot(HaveOccurred())
				signed := createSignedClientJWT(edPriKey, map[string]any{
					"purpose":    tokens.ClientIDPurpose,
					"public_key": hex.EncodeToString(edPublicKey),
				})

				// should fail the public key not there
				auth.clientJwtSigners = []string{filepath.Join(td, "public.pem")}
				claims, err := auth.parseClientIDJWT(signed)
				Expect(err).To(MatchError("could not parse client id token: ed25519 public key required"))
				Expect(claims).To(BeNil())

				// should now pass after having done a multi check
				auth.clientJwtSigners = []string{filepath.Join(td, "public.pem"), "/nonexisting", hex.EncodeToString(edPublicKey)}
				claims, err = auth.parseClientIDJWT(signed)
				Expect(err).ToNot(HaveOccurred())
				Expect(claims.CallerID).To(Equal("up=ginkgo"))
			})
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
		})

		Describe("System User", func() {
			It("Should should set correct permissions", func() {
				auth.setClientPermissions(user, "", &tokens.ClientIDClaims{Permissions: &tokens.ClientPermissions{OrgAdmin: true}}, log)
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
				auth.setClientPermissions(user, "", &tokens.ClientIDClaims{Permissions: &tokens.ClientPermissions{StreamsUser: true}}, log)
				Expect(user.Permissions.Subscribe).To(Equal(&server.SubjectPermission{
					Allow: minSub,
				}))
				Expect(user.Permissions.Publish).To(Equal(&server.SubjectPermission{}))
			})

			It("Should set correct permissions for the choria user", func() {
				user.Account = auth.choriaAccount
				auth.setClientPermissions(user, "", &tokens.ClientIDClaims{Permissions: &tokens.ClientPermissions{StreamsUser: true}}, log)
				Expect(user.Permissions.Subscribe).To(Equal(&server.SubjectPermission{
					Allow: minSub,
				}))
				Expect(user.Permissions.Publish).To(Equal(&server.SubjectPermission{
					Allow: append(minPub,
						"$JS.API.INFO",
						"$JS.API.STREAM.NAMES",
						"$JS.API.STREAM.LIST",
						"$JS.API.STREAM.INFO.*",
						"$JS.API.STREAM.MSG.GET.*",
						"$JS.API.STREAM.MSG.DELETE.*",
						"$JS.API.DIRECT.GET.*",
						"$JS.API.DIRECT.GET.*.>",
						"$JS.API.CONSUMER.CREATE.*",
						"$JS.API.CONSUMER.CREATE.*.>",
						"$JS.API.CONSUMER.DURABLE.CREATE.*.*",
						"$JS.API.CONSUMER.DELETE.*.*",
						"$JS.API.CONSUMER.NAMES.*",
						"$JS.API.CONSUMER.LIST.*",
						"$JS.API.CONSUMER.INFO.*.*",
						"$JS.API.CONSUMER.MSG.NEXT.*.*",
						"$JS.ACK.>",
						"$JS.FC.>"),
				}))
			})
		})

		Describe("Governor Users", func() {
			It("Should not set provisioner permissions", func() {
				user.Account = auth.provisioningAccount
				auth.setClientPermissions(user, "", &tokens.ClientIDClaims{Permissions: &tokens.ClientPermissions{StreamsUser: true, Governor: true}}, log)
				Expect(user.Permissions.Subscribe).To(Equal(&server.SubjectPermission{
					Allow: minSub,
				}))
				Expect(user.Permissions.Publish).To(Equal(&server.SubjectPermission{
					Allow: minPub,
				}))
			})

			It("Should set choria permissions", func() {
				user.Account = auth.choriaAccount
				auth.setClientPermissions(user, "", &tokens.ClientIDClaims{Permissions: &tokens.ClientPermissions{StreamsUser: true, Governor: true}}, log)
				Expect(user.Permissions.Subscribe).To(Equal(&server.SubjectPermission{
					Allow: minSub,
				}))
				Expect(user.Permissions.Publish).To(Equal(&server.SubjectPermission{
					Allow: append(minPub, []string{
						"$JS.API.INFO",
						"$JS.API.STREAM.NAMES",
						"$JS.API.STREAM.LIST",
						"$JS.API.STREAM.INFO.*",
						"$JS.API.STREAM.MSG.GET.*",
						"$JS.API.STREAM.MSG.DELETE.*",
						"$JS.API.DIRECT.GET.*",
						"$JS.API.DIRECT.GET.*.>",
						"$JS.API.CONSUMER.CREATE.*",
						"$JS.API.CONSUMER.CREATE.*.>",
						"$JS.API.CONSUMER.DURABLE.CREATE.*.*",
						"$JS.API.CONSUMER.DELETE.*.*",
						"$JS.API.CONSUMER.NAMES.*",
						"$JS.API.CONSUMER.LIST.*",
						"$JS.API.CONSUMER.INFO.*.*",
						"$JS.API.CONSUMER.MSG.NEXT.*.*",
						"$JS.ACK.>",
						"$JS.FC.>",
						"*.governor.*",
					}...),
				}))
			})

		})

		Describe("Event Viewers", func() {
			It("Should set provisioning permissions", func() {
				user.Account = auth.provisioningAccount
				auth.setClientPermissions(user, "", &tokens.ClientIDClaims{Permissions: &tokens.ClientPermissions{EventsViewer: true}}, log)
				Expect(user.Permissions.Subscribe).To(Equal(&server.SubjectPermission{
					Allow: append(minSub, "choria.lifecycle.event.*.provision_mode_server"),
				}))
				Expect(user.Permissions.Publish).To(Equal(&server.SubjectPermission{
					Allow: minPub,
				}))
			})

			It("Should set choria permissions", func() {
				user.Account = auth.choriaAccount
				auth.setClientPermissions(user, "", &tokens.ClientIDClaims{Permissions: &tokens.ClientPermissions{EventsViewer: true}}, log)
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
				auth.setClientPermissions(user, "", &tokens.ClientIDClaims{Permissions: &tokens.ClientPermissions{ElectionUser: true}}, log)
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
				auth.setClientPermissions(user, "", &tokens.ClientIDClaims{Permissions: &tokens.ClientPermissions{ElectionUser: true}}, log)
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
				auth.setClientPermissions(user, "", &tokens.ClientIDClaims{Permissions: &tokens.ClientPermissions{StreamsAdmin: true}}, log)
				Expect(user.Permissions.Subscribe).To(Equal(&server.SubjectPermission{
					Allow: minSub,
				}))
				Expect(user.Permissions.Publish).To(Equal(&server.SubjectPermission{
					Allow: minPub,
				}))
			})

			It("Should set correct permissions for choria user", func() {
				user.Account = auth.choriaAccount
				auth.setClientPermissions(user, "", &tokens.ClientIDClaims{Permissions: &tokens.ClientPermissions{StreamsAdmin: true}}, log)
				Expect(user.Permissions.Subscribe).To(Equal(&server.SubjectPermission{
					Allow: append(minSub, "$JS.EVENT.>"),
				}))
				Expect(user.Permissions.Publish).To(Equal(&server.SubjectPermission{
					Allow: append(minPub, "$JS.>"),
				}))
			})
		})

		Describe("Fleet Management", func() {
			It("Should set correct permissions for fleet management users", func() {
				user.Account = auth.choriaAccount
				auth.setClientPermissions(user, "", &tokens.ClientIDClaims{Permissions: &tokens.ClientPermissions{FleetManagement: true}}, log)
				Expect(user.Permissions.Subscribe).To(Equal(&server.SubjectPermission{
					Allow: minSub,
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

		Describe("Additional subjects", func() {
			It("Should set the permissions correctly", func() {
				user.Account = auth.choriaAccount
				auth.setClientPermissions(user, "", &tokens.ClientIDClaims{
					AdditionalSubscribeSubjects: []string{"sub.>"},
					AdditionalPublishSubjects:   []string{"pub.>"},
				}, log)
				Expect(user.Permissions.Subscribe).To(Equal(&server.SubjectPermission{
					Allow: append(minSub, "sub.>"),
				}))
				Expect(user.Permissions.Publish).To(Equal(&server.SubjectPermission{
					Allow: []string{"pub.>"},
				}))
			})
		})

		Describe("Minimal Permissions", func() {
			It("Should support caller private reply subjects", func() {
				auth.setClientPermissions(user, "u=ginkgo", nil, log)
				Expect(user.Permissions.Subscribe).To(Equal(&server.SubjectPermission{
					Allow: []string{"*.reply.0f47cbbd2accc01a51e57261d6e64b8b.>"},
				}))
				Expect(user.Permissions.Publish).To(Equal(&server.SubjectPermission{}))
			})

			It("Should support standard reply subjects", func() {
				auth.setClientPermissions(user, "", nil, log)
				Expect(user.Permissions.Subscribe).To(Equal(&server.SubjectPermission{
					Allow: []string{"*.reply.>"},
				}))
				Expect(user.Permissions.Publish).To(Equal(&server.SubjectPermission{}))
			})
		})
	})

	Describe("setServerPermissions", func() {
		It("Should set correct permissions", func() {
			auth.setServerPermissions(user, nil, log)

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
			auth.setServerPermissions(user, nil, log)
			Expect(user.Permissions.Publish.Deny).To(Equal([]string{">"}))
			Expect(user.Permissions.Publish.Allow).To(BeNil())
		})
	})
})
