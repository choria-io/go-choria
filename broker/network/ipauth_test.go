package network

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io/ioutil"
	"math/big"
	"net"
	"path/filepath"
	"time"

	"github.com/form3tech-oss/jwt-go"
	"github.com/nats-io/nats-server/v2/server"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

var _ = Describe("Network Broker/IPAuth", func() {
	var (
		log  *logrus.Entry
		auth *IPAuth
		user *server.User
	)

	BeforeEach(func() {
		logger := logrus.New()
		logger.Out = ioutil.Discard
		log = logrus.NewEntry(logger)
		auth = &IPAuth{
			allowList: []string{},
			log:       log,
		}
		user = &server.User{
			Username:    "bob",
			Password:    "secret",
			Permissions: &server.Permissions{},
		}
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
			td, err = ioutil.TempDir("", "")
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
			err = ioutil.WriteFile(filepath.Join(td, "public.pem"), out.Bytes(), 0600)
			Expect(err).ToNot(HaveOccurred())

			out.Reset()

			blk := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}
			pem.Encode(out, blk)

			err = ioutil.WriteFile(filepath.Join(td, "private.pem"), out.Bytes(), 0600)
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
