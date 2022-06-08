package broker_auth

import (
	"context"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/tls"
	"encoding/hex"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/integration/testbroker"
	"github.com/choria-io/go-choria/integration/testutil"
	"github.com/choria-io/go-choria/tokens"
	"github.com/nats-io/nats.go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/sirupsen/logrus"
)

// TestBrokerAuthentication is a number of tests that essentially test out broker.ChoriaAuth class
// by starting a broker with a specific configuration file and then using the nats.go client to attempt
// to connect to it, bypass restrictions and more
func TestBrokerAuthentication(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Broker Authentication")
}

var _ = Describe("Authentication", func() {
	var (
		ctx     context.Context
		cancel  context.CancelFunc
		wg      sync.WaitGroup
		logger  *logrus.Logger
		logbuff *gbytes.Buffer
	)

	BeforeEach(func() {
		ctx, cancel = context.WithTimeout(context.Background(), 45*time.Second)
		DeferCleanup(func() {
			cancel()
			Eventually(logbuff, 5).Should(gbytes.Say("Choria Network Broker shut down"))
		})

		logbuff, logger = testutil.GbytesLogger(logrus.DebugLevel)
	})

	Describe("JWT Auth", func() {
		var (
			edPrivateKey   ed25519.PrivateKey
			edPublicKey    ed25519.PublicKey
			nodeSignerPK   *rsa.PrivateKey
			clientSignerPK *rsa.PrivateKey
		)

		BeforeEach(func() {
			_, err := testbroker.StartNetworkBrokerWithConfigFile(ctx, &wg, "testdata/anontls.conf", logger)
			Expect(err).ToNot(HaveOccurred())

			Eventually(logbuff, 1).Should(gbytes.Say("Server is ready"))

			Expect(logbuff.Contents()).To(ContainSubstring("Allowing unverified TLS connections for AAA signed clients"))
			Expect(logbuff.Contents()).To(ContainSubstring("Allowing unverified TLS connections for Provisioner signed servers"))
			Expect(logbuff.Contents()).To(ContainSubstring("TLS required for client connections"))

			edPublicKey, edPrivateKey, err = choria.Ed25519KeyPair()
			Expect(err).ToNot(HaveOccurred())

			nodeSignerPK, err = testutil.LoadRSAKey("../../ca/node-signer-key.pem")
			Expect(err).ToNot(HaveOccurred())
			clientSignerPK, err = testutil.LoadRSAKey("../../ca/client-signer-key.pem")
			Expect(err).ToNot(HaveOccurred())
		})

		Describe("JWT Token Servers", func() {
			It("Should fail for invalid server tokens", func() {
				// signing the server jwt with the client signer which will yield an invalid connection
				jwt, err := testutil.CreateSignedServerJWT(clientSignerPK, edPublicKey, map[string]interface{}{
					"purpose":     tokens.ServerPurpose,
					"public_key":  hex.EncodeToString(edPublicKey),
					"collectives": []string{"c1"},
				})
				Expect(err).ToNot(HaveOccurred())

				_, err = nats.Connect("nats://localhost:4222",
					nats.Secure(&tls.Config{InsecureSkipVerify: true}),
					nats.Token(jwt),
					nats.UserJWT(func() (string, error) {
						return jwt, nil
					}, func(n []byte) ([]byte, error) {
						return choria.Ed25519Sign(edPrivateKey, n)
					}),
				)
				Expect(err).To(MatchError("nats: Authorization Violation"))
				Eventually(logbuff, 5).Should(gbytes.Say("Performing JWT based authentication verification"))
				Eventually(logbuff, 1).Should(gbytes.Say("could not parse server id token: crypto/rsa: verification error"))
				Eventually(logbuff, 1).Should(gbytes.Say("invalid nonce signature or jwt token"))
				Eventually(logbuff, 1).ShouldNot(gbytes.Say("Registering user"))
			})

			It("Should fail for invalid nonce signatures", func() {
				jwt, err := testutil.CreateSignedServerJWT(nodeSignerPK, edPublicKey, map[string]interface{}{
					"purpose":     tokens.ServerPurpose,
					"public_key":  hex.EncodeToString(edPublicKey),
					"collectives": []string{"c1"},
				})
				Expect(err).ToNot(HaveOccurred())

				_, err = nats.Connect("nats://localhost:4222",
					nats.Secure(&tls.Config{InsecureSkipVerify: true}),
					nats.Token(jwt),
					nats.UserJWT(func() (string, error) {
						return jwt, nil
					}, func(n []byte) ([]byte, error) {
						// we create an invalid nonce signature so this must fail
						return []byte("invalid signature"), nil
					}),
				)
				Expect(err).To(MatchError("nats: Authorization Violation"))
				Eventually(logbuff, 5).Should(gbytes.Say("Performing JWT based authentication verification"))
				Eventually(logbuff, 1).Should(gbytes.Say("nonce signature verification failed: nonce signature did not verify using pub key in the jwt"))
				Eventually(logbuff, 1).ShouldNot(gbytes.Say("Registering user"))
			})

			It("Should accept valid servers and set permissions", func() {
				jwt, err := testutil.CreateSignedServerJWT(nodeSignerPK, edPublicKey, map[string]interface{}{
					"purpose":     tokens.ServerPurpose,
					"public_key":  hex.EncodeToString(edPublicKey),
					"collectives": []string{"c1"},
				})
				Expect(err).ToNot(HaveOccurred())

				clBuffer, clLogger := testutil.GbytesLogger(logrus.DebugLevel)

				nc, err := nats.Connect("nats://localhost:4222",
					nats.ErrorHandler(func(_ *nats.Conn, _ *nats.Subscription, err error) {
						clLogger.Errorf(strings.ReplaceAll(err.Error(), `"`, ``))
					}),
					nats.Secure(&tls.Config{InsecureSkipVerify: true}),
					nats.Token(jwt),
					nats.UserJWT(func() (string, error) {
						return jwt, nil
					}, func(n []byte) ([]byte, error) {
						return choria.Ed25519Sign(edPrivateKey, n)
					}),
				)
				Expect(err).ToNot(HaveOccurred())
				defer nc.Close()

				Eventually(logbuff, 5).Should(gbytes.Say("Performing JWT based authentication verification"))
				Eventually(logbuff, 1).Should(gbytes.Say("Successfully verified nonce signature"))
				Eventually(logbuff, 1).Should(gbytes.Say("Extracted remote identity ginkgo.example.net from JWT token"))
				Eventually(logbuff, 1).Should(gbytes.Say("Setting server permissions based on token claims"))
				Eventually(logbuff, 1).Should(gbytes.Say("Registering user 'ginkgo.example.net' in account 'choria'"))

				Expect(nc.ConnectedUrl()).To(Equal("nats://localhost:4222"))
				Expect(nc.Publish("choria.lifecycle.x", []byte("x"))).ToNot(HaveOccurred())
				Expect(nc.Publish("choria.machine.transition", []byte("x"))).ToNot(HaveOccurred())
				Expect(nc.Publish("choria.machine.watcher.x", []byte("x"))).ToNot(HaveOccurred())

				// should not allow submission by default
				Expect(nc.Publish("c1.submission.in.x", []byte("x"))).ToNot(HaveOccurred())
				Eventually(clBuffer, 1).Should(gbytes.Say(`Permissions Violation for Publish to c1.submission.in.x`))

				// should only allow us to sub to our id
				_, err = nc.Subscribe("c1.node.other.node", func(m *nats.Msg) {})
				Expect(err).ToNot(HaveOccurred())
				Eventually(clBuffer, 1).Should(gbytes.Say("Permissions Violation for Subscription to c1.node.other.node"))

				_, err = nc.Subscribe("c1.node.ginkgo.example.net", func(m *nats.Msg) {})
				Expect(err).ToNot(HaveOccurred())
				Eventually(clBuffer, 1).ShouldNot(gbytes.Say("Permissions Violation for Subscription"))

				// should not allow sub collective escape
				_, err = nc.Subscribe("other.node.ginkgo.example.net", func(m *nats.Msg) {})
				Expect(err).ToNot(HaveOccurred())
				Eventually(clBuffer, 1).Should(gbytes.Say("Permissions Violation for Subscription to other.node.ginkgo.example.net"))

			})
		})

		Describe("JWT Token Clients", func() {
			It("Should fail for invalid client tokens", func() {
				// signing the client jwt with the server signer which will yield an invalid connection
				jwt, err := testutil.CreateSignedServerJWT(nodeSignerPK, edPublicKey, map[string]interface{}{
					"purpose":    tokens.ClientIDPurpose,
					"callerid":   "up=ginkgo",
					"public_key": hex.EncodeToString(edPublicKey),
				})
				Expect(err).ToNot(HaveOccurred())

				_, err = nats.Connect("nats://localhost:4222",
					nats.Secure(&tls.Config{InsecureSkipVerify: true}),
					nats.Token(jwt),
					nats.UserJWT(func() (string, error) {
						return jwt, nil
					}, func(n []byte) ([]byte, error) {
						return choria.Ed25519Sign(edPrivateKey, n)
					}),
				)
				Expect(err).To(MatchError("nats: Authorization Violation"))
				Eventually(logbuff, 5).Should(gbytes.Say("Performing JWT based authentication verification"))
				Eventually(logbuff, 1).Should(gbytes.Say("could not parse client id token: crypto/rsa: verification error"))
				Eventually(logbuff, 1).Should(gbytes.Say("invalid nonce signature or jwt token"))
				Eventually(logbuff, 1).ShouldNot(gbytes.Say("Registering user"))
			})

			It("Should fail for invalid nonce signatures", func() {
				jwt, err := testutil.CreateSignedServerJWT(clientSignerPK, edPublicKey, map[string]interface{}{
					"purpose":    tokens.ClientIDPurpose,
					"callerid":   "up=ginkgo",
					"public_key": hex.EncodeToString(edPublicKey),
				})
				Expect(err).ToNot(HaveOccurred())

				_, err = nats.Connect("nats://localhost:4222",
					nats.Secure(&tls.Config{InsecureSkipVerify: true}),
					nats.Token(jwt),
					nats.UserJWT(func() (string, error) {
						return jwt, nil
					}, func(n []byte) ([]byte, error) {
						// we create an invalid nonce signature so this must fail
						return []byte("invalid signature"), nil
					}),
				)
				Expect(err).To(MatchError("nats: Authorization Violation"))

				Eventually(logbuff, 5).Should(gbytes.Say("Performing JWT based authentication verification"))
				Eventually(logbuff, 1).Should(gbytes.Say("nonce signature verification failed: nonce signature did not verify using pub key in the jwt"))
				Eventually(logbuff, 1).ShouldNot(gbytes.Say("Registering user"))
			})

			It("Should require a caller id claim", func() {
				jwt, err := testutil.CreateSignedServerJWT(clientSignerPK, edPublicKey, map[string]interface{}{
					"purpose":    tokens.ClientIDPurpose,
					"public_key": hex.EncodeToString(edPublicKey),
				})
				Expect(err).ToNot(HaveOccurred())

				_, err = nats.Connect("nats://localhost:4222",
					nats.Secure(&tls.Config{InsecureSkipVerify: true}),
					nats.Token(jwt),
					nats.UserJWT(func() (string, error) {
						return jwt, nil
					}, func(n []byte) ([]byte, error) {
						return choria.Ed25519Sign(edPrivateKey, n)
					}),
				)
				Expect(err).To(MatchError("nats: Authorization Violation"))

				Eventually(logbuff, 5).Should(gbytes.Say("Performing JWT based authentication verification"))
				Eventually(logbuff, 1).Should(gbytes.Say("no callerid in claims"))
				Eventually(logbuff, 1).ShouldNot(gbytes.Say("Registering user"))
			})

			It("Should accept valid clients and set permissions", func() {
				jwt, err := testutil.CreateSignedServerJWT(clientSignerPK, edPublicKey, map[string]interface{}{
					"purpose":    tokens.ClientIDPurpose,
					"callerid":   "up=ginkgo",
					"public_key": hex.EncodeToString(edPublicKey),
				})
				Expect(err).ToNot(HaveOccurred())

				clBuffer, clLogger := testutil.GbytesLogger(logrus.DebugLevel)

				nc, err := nats.Connect("nats://localhost:4222",
					nats.ErrorHandler(func(_ *nats.Conn, _ *nats.Subscription, err error) {
						clLogger.Errorf(strings.ReplaceAll(err.Error(), `"`, ``))
					}),
					nats.Secure(&tls.Config{InsecureSkipVerify: true}),
					nats.Token(jwt),
					nats.UserJWT(func() (string, error) {
						return jwt, nil
					}, func(n []byte) ([]byte, error) {
						return choria.Ed25519Sign(edPrivateKey, n)
					}),
				)
				Expect(err).ToNot(HaveOccurred())
				defer nc.Close()

				Eventually(logbuff, 5).Should(gbytes.Say("Performing JWT based authentication verification"))
				Eventually(logbuff, 1).Should(gbytes.Say("Successfully verified nonce signature"))
				Eventually(logbuff, 1).Should(gbytes.Say("Extracted caller id up=ginkgo from JWT token"))
				Eventually(logbuff, 1).Should(gbytes.Say("Creating ACLs for a private reply subject on \\*.reply.e33bf0376d4accbb4a8fd24b2f840b2e.>"))
				Eventually(logbuff, 1).Should(gbytes.Say("Registering user 'up=ginkgo' in account 'choria'"))

				// should only access its own replies
				_, err = nc.Subscribe("c1.reply.xxxx.>", func(_ *nats.Msg) {})
				Expect(err).ToNot(HaveOccurred())
				Eventually(logbuff, 1).Should(gbytes.Say("Subscription Violation - User .+up=ginkgo.+, Subject .+c1.reply.xxxx.>"))
				Eventually(clBuffer, 1).Should(gbytes.Say("Permissions Violation for Subscription to c1.reply.xxxx.>"))

				_, err = nc.Subscribe("c1.reply.e33bf0376d4accbb4a8fd24b2f840b2e.>", func(_ *nats.Msg) {})
				Expect(err).ToNot(HaveOccurred())
				Eventually(logbuff, 1).ShouldNot(gbytes.Say("c1.reply.e33bf0376d4accbb4a8fd24b2f840b2e"))
				Eventually(clBuffer, 1).ShouldNot(gbytes.Say("c1.reply.e33bf0376d4accbb4a8fd24b2f840b2e"))

				// should not be able to be a node
				_, err = nc.Subscribe("c1.node.other.node", func(m *nats.Msg) {})
				Expect(err).ToNot(HaveOccurred())
				Eventually(clBuffer, 1).Should(gbytes.Say("Permissions Violation for Subscription to c1.node.other.node"))

			})
		})
	})

	Describe("mTLS Connections", FlakeAttempts(3), func() {
		BeforeEach(func() {
			_, err := testbroker.StartNetworkBrokerWithConfigFile(ctx, &wg, "testdata/mtls.conf", logger)
			Expect(err).ToNot(HaveOccurred())

			Eventually(logbuff, 2).Should(gbytes.Say("TLS required for client connections"))
			Eventually(logbuff, 1).Should(gbytes.Say("Server is ready"))
		})

		It("Should reject unverified TLS connections", func() {
			_, err := nats.Connect("nats://localhost:4222",
				nats.Secure(&tls.Config{InsecureSkipVerify: true}),
			)
			Expect(err).To(MatchError("nats: Authorization Violation"))
			Eventually(logbuff, 2).Should(gbytes.Say("Rejecting unverified connection without token"))
			Expect(logbuff).ToNot(gbytes.Say("Registering user"))
		})

		It("Should be rejected using the certs from an unknown CA", func() {
			nc, err := nats.Connect("tls://localhost:4222",
				nats.ClientCert(testutil.CertPath("two", "rip.mcollective"), testutil.KeyPath("two", "rip.mcollective")),
				nats.RootCAs(testutil.CertPath("one", "ca")),
			)
			Eventually(logbuff, 5).Should(gbytes.Say("failed to verify client certificate: x509: certificate signed by unknown authority"))
			Expect(err).To(MatchError("remote error: tls: bad certificate"))
			Expect(nc).To(BeNil())
			Expect(logbuff).ToNot(gbytes.Say("Registering user"))
		})

		It("Should allow connections using the right CA and valid keys", func() {
			nc, err := nats.Connect("tls://localhost:4222",
				nats.ClientCert(testutil.CertPath("one", "rip.mcollective"), testutil.KeyPath("one", "rip.mcollective")),
				nats.RootCAs(testutil.CertPath("one", "ca")),
			)
			Expect(err).ToNot(HaveOccurred())
			defer nc.Close()

			Eventually(logbuff, 1).Should(gbytes.Say("Registering user '' in account 'choria'"))
			Expect(nc.ConnectedUrl()).To(Equal("tls://localhost:4222"))
		})
	})

	Describe("System Account Connections", func() {
		BeforeEach(func() {
			_, err := testbroker.StartNetworkBrokerWithConfigFile(ctx, &wg, "testdata/provisioning.conf", logger)
			Expect(err).ToNot(HaveOccurred())
			Eventually(logbuff, 1).Should(gbytes.Say("Allowing unverified TLS connections for provisioning purposes"))
			Eventually(logbuff, 2).Should(gbytes.Say("TLS required for client connections"))
			Eventually(logbuff, 1).Should(gbytes.Say("Server is ready"))
		})

		It("Should prevent the system user from connecting without full TLS", func() {
			_, err := nats.Connect("nats://localhost:4222",
				nats.UserInfo("system", "systemS3cret"),
				nats.Secure(&tls.Config{InsecureSkipVerify: true}))
			Expect(err).To(MatchError("nats: Authorization Violation"))
			Expect(logbuff).To(gbytes.Say("Handling unverified TLS system user failed, denying: no JWT token received"))
			Expect(logbuff).ToNot(gbytes.Say("Registering user"))
		})

		It("Should verify credentials", func() {
			_, err := nats.Connect("nats://localhost:4222",
				nats.UserInfo("system", "s3cret"),
				nats.ClientCert(testutil.CertPath("one", "rip.mcollective"), testutil.KeyPath("one", "rip.mcollective")),
				nats.RootCAs(testutil.CertPath("one", "ca")))
			Expect(err).To(MatchError("nats: Authorization Violation"))
			Expect(logbuff).To(gbytes.Say("Handling system user failed, denying: invalid system credentials"))
			Expect(logbuff).ToNot(gbytes.Say("Registering user"))
		})

		It("Should register correct connections", func() {
			nc, err := nats.Connect("nats://localhost:4222",
				nats.UserInfo("system", "systemS3cret"),
				nats.ClientCert(testutil.CertPath("one", "rip.mcollective"), testutil.KeyPath("one", "rip.mcollective")),
				nats.RootCAs(testutil.CertPath("one", "ca")))
			Expect(err).ToNot(HaveOccurred())
			Expect(nc.ConnectedUrl()).To(Equal("nats://localhost:4222"))
			Expect(logbuff).To(gbytes.Say("Registering user 'system' in account 'system'"))
		})
	})

	Describe("Provisioning Mode Server Connections", func() {
		var (
			jwt        []byte
			invalidJwt []byte
		)

		BeforeEach(func() {
			_, err := testbroker.StartNetworkBrokerWithConfigFile(ctx, &wg, "testdata/provisioning.conf", logger)
			Expect(err).ToNot(HaveOccurred())
			Eventually(logbuff, 1).Should(gbytes.Say("Allowing unverified TLS connections for provisioning purposes"))
			Eventually(logbuff, 2).Should(gbytes.Say("TLS required for client connections"))
			Eventually(logbuff, 1).Should(gbytes.Say("Server is ready"))

			invalidJwt, err = os.ReadFile("../../ca/invalid-provisioning.jwt")
			Expect(err).ToNot(HaveOccurred())

			jwt, err = os.ReadFile("../../ca/provisioning.jwt")
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should fail without a token", func() {
			_, err := nats.Connect("nats://localhost:4222",
				nats.Secure(&tls.Config{InsecureSkipVerify: true}),
			)
			Expect(err).To(MatchError("nats: Authorization Violation"))
			Expect(logbuff).To(gbytes.Say("Rejecting unverified connection without token"))
			Expect(logbuff).To(gbytes.Say("unverified connection without JWT token"))
			Expect(logbuff).To(gbytes.Say("provisioning requires a token"))
			Expect(logbuff).ToNot(gbytes.Say("Registering user"))
		})

		It("Should verify the token", func() {
			_, err := nats.Connect("nats://localhost:4222",
				nats.Secure(&tls.Config{InsecureSkipVerify: true}),
				nats.Token(string(invalidJwt)),
			)
			Expect(err).To(MatchError("nats: Authorization Violation"))

			Expect(logbuff).To(gbytes.Say("Performing JWT based authentication verification"))
			Expect(logbuff).To(gbytes.Say("could not parse provisioner token: crypto/rsa: verification error"))
			Expect(logbuff).ToNot(gbytes.Say("Registering user"))
		})

		It("Should accept correctly configured servers", func() {
			nc, err := nats.Connect("nats://localhost:4222",
				nats.Secure(&tls.Config{InsecureSkipVerify: true}),
				nats.Token(string(jwt)),
			)
			Expect(err).ToNot(HaveOccurred())

			Expect(nc.ConnectedUrl()).To(Equal("nats://localhost:4222"))
			Expect(logbuff).To(gbytes.Say("Performing JWT based authentication verification"))
			Expect(logbuff).To(gbytes.Say("Allowing a provisioning server from using unverified TLS connection from"))
			Expect(logbuff).To(gbytes.Say("Registering user '' in account 'provisioning'"))
			Expect(logbuff).ToNot(gbytes.Say("in account 'choria'"))
		})
	})

	Describe("Server Provisioner connections", func() {
		It("Should deny the provisioner user over unverified tls", func() {
			_, err := testbroker.StartNetworkBrokerWithConfigFile(ctx, &wg, "testdata/provisioning.conf", logger)
			Expect(err).ToNot(HaveOccurred())
			Eventually(logbuff, 1).Should(gbytes.Say("Server is ready"))

			_, err = nats.Connect("nats://localhost:4222",
				nats.Secure(&tls.Config{InsecureSkipVerify: true}),
				nats.UserInfo("provisioner", "s3cret"),
			)

			Expect(err).To(MatchError("nats: Authorization Violation"))
			Expect(logbuff).To(gbytes.Say("provisioning user is only allowed over verified TLS connections"))
			Expect(logbuff).ToNot(gbytes.Say("Registering user"))
		})

		It("Should only accept provisioning connections when configured", func() {
			_, err := testbroker.StartNetworkBrokerWithConfigFile(ctx, &wg, "testdata/mtls.conf", logger)
			Expect(err).ToNot(HaveOccurred())
			Eventually(logbuff, 1).Should(gbytes.Say("Server is ready"))

			_, err = nats.Connect("tls://localhost:4222",
				nats.UserInfo("provisioner", "s3cret"),
				nats.ClientCert(testutil.CertPath("one", "rip.mcollective"), testutil.KeyPath("one", "rip.mcollective")),
				nats.RootCAs(testutil.CertPath("one", "ca")),
			)
			Expect(err).To(MatchError("nats: Authorization Violation"))
			Eventually(logbuff, 1).Should(gbytes.Say("provisioning user password not enabled"))
			Expect(logbuff).ToNot(gbytes.Say("Registering user"))
		})

		It("Should require a password on the connection", func() {
			_, err := testbroker.StartNetworkBrokerWithConfigFile(ctx, &wg, "testdata/provisioning.conf", logger)
			Expect(err).ToNot(HaveOccurred())
			Eventually(logbuff, 1).Should(gbytes.Say("Server is ready"))

			_, err = nats.Connect("tls://localhost:4222",
				nats.UserInfo("provisioner", ""),
				nats.ClientCert(testutil.CertPath("one", "rip.mcollective"), testutil.KeyPath("one", "rip.mcollective")),
				nats.RootCAs(testutil.CertPath("one", "ca")),
			)
			Expect(err).To(MatchError("nats: Authorization Violation"))
			Expect(string(logbuff.Contents())).To(MatchRegexp("Handling provisioning user connection failed, denying.+: password required"))
		})

		It("Should register the connection into the provisioning account", func() {
			_, err := testbroker.StartNetworkBrokerWithConfigFile(ctx, &wg, "testdata/provisioning.conf", logger)
			Expect(err).ToNot(HaveOccurred())
			Eventually(logbuff, 1).Should(gbytes.Say("Allowing unverified TLS connections for provisioning purposes"))
			Eventually(logbuff, 1).Should(gbytes.Say("TLS required for client connections"))
			Eventually(logbuff, 1).Should(gbytes.Say("Server is ready"))

			nc, err := nats.Connect("tls://localhost:4222",
				nats.UserInfo("provisioner", "s3cret"),
				nats.ClientCert(testutil.CertPath("one", "rip.mcollective"), testutil.KeyPath("one", "rip.mcollective")),
				nats.RootCAs(testutil.CertPath("one", "ca")),
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(nc.ConnectedUrl()).To(Equal("tls://localhost:4222"))
			Expect(logbuff).To(gbytes.Say(`Registering user 'provisioner' in account 'provisioning'`))
		})
	})
})
