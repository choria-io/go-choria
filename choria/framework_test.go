// Copyright (c) 2017-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package choria

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/tokens"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestChoria(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Choria")
}

var _ = Describe("Choria", func() {
	Describe("NewChoria", func() {
		It("Should initialize choria correctly", func() {
			cfg := config.NewConfigForTests()
			cfg.Choria.SSLDir = "/nonexisting"

			c := cfg.Choria
			Expect(c.UseSRVRecords).To(BeTrue())
		})
	})

	Describe("JWT", func() {
		var (
			fw         *Framework
			cfg        *config.Config
			err        error
			privateKey *rsa.PrivateKey
			td         string
		)

		BeforeEach(func() {
			td, err = os.MkdirTemp("", "")
			Expect(err).ToNot(HaveOccurred())

			cfg = config.NewConfigForTests()
			cfg.Choria.SSLDir = "/nonexisting"
			cfg.DisableSecurityProviderVerify = true

			fw, err = NewWithConfig(cfg)
			Expect(err).ToNot(HaveOccurred())

			privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
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

			edPub, _, err := Ed25519KeyPair()
			Expect(err).ToNot(HaveOccurred())

			clientJwtPath := filepath.Join(td, "good-client.jwt")
			claims, err := tokens.NewClientIDClaims("up=ginkgo", nil, "choria", nil, "", "Ginkgo", time.Hour, nil, edPub)
			Expect(err).ToNot(HaveOccurred())
			signed, err := tokens.SignToken(claims, privateKey)
			Expect(err).ToNot(HaveOccurred())
			err = os.WriteFile(clientJwtPath, []byte(signed), 0600)
			Expect(err).ToNot(HaveOccurred())

			serverJwtPath := filepath.Join(td, "good-server.jwt")
			sclaims, err := tokens.NewServerClaims("ginkgo.example.net", []string{"c"}, "choria", nil, nil, edPub, "Ginkgo", time.Hour)
			Expect(err).ToNot(HaveOccurred())
			signed, err = tokens.SignToken(sclaims, privateKey)
			Expect(err).ToNot(HaveOccurred())
			err = os.WriteFile(serverJwtPath, []byte(signed), 0600)
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			os.RemoveAll(td)
		})

		Describe("UniqueIDFromUnverifiedToken", func() {
			It("Should extract the correct items for servers", func() {
				cfg.Choria.ServerAnonTLS = true
				fw.Config.InitiatedByServer = true
				cfg.Choria.ServerTokenFile = filepath.Join(td, "good-server.jwt")
				caller, id, exp, token, err := fw.UniqueIDFromUnverifiedToken()
				Expect(err).ToNot(HaveOccurred())
				Expect(exp).To(BeTemporally("~", time.Now().Add(time.Hour), time.Second))

				expectedT, err := os.ReadFile(cfg.Choria.ServerTokenFile)
				Expect(err).ToNot(HaveOccurred())

				Expect(token).To(Equal(strings.TrimSpace(string(expectedT))))
				Expect(id).To(Equal("3f7c3a791b0eb10da51dca4cdedb9418"))
				Expect(caller).To(Equal("ginkgo.example.net"))
			})

			It("Should extract the correct items for clients", func() {
				cfg.Choria.ClientAnonTLS = true
				cfg.Choria.RemoteSignerTokenFile = filepath.Join(td, "good-client.jwt")
				caller, id, exp, token, err := fw.UniqueIDFromUnverifiedToken()
				Expect(err).ToNot(HaveOccurred())
				Expect(exp).To(BeTemporally("~", time.Now().Add(time.Hour), time.Second))

				expectedT, err := os.ReadFile(cfg.Choria.RemoteSignerTokenFile)
				Expect(err).ToNot(HaveOccurred())

				Expect(token).To(Equal(strings.TrimSpace(string(expectedT))))
				Expect(id).To(Equal("e33bf0376d4accbb4a8fd24b2f840b2e"))
				Expect(caller).To(Equal("up=ginkgo"))
			})
		})

		Describe("SignerTokenFile", func() {
			BeforeEach(func() {
				cfg.Choria.ChoriaSecurityTokenFile = "stf"
				cfg.Choria.RemoteSignerTokenFile = "rstf"
				cfg.Choria.ServerAnonTLS = true
				cfg.Choria.ServerTokenFile = "stf"
			})

			It("Should prefer choria security token file", func() {
				tf, err := fw.SignerTokenFile()
				Expect(err).ToNot(HaveOccurred())
				Expect(tf).To(Equal("stf"))
			})

			It("Should support remote signer token files", func() {
				cfg.Choria.ChoriaSecurityTokenFile = ""
				tf, err := fw.SignerTokenFile()
				Expect(err).ToNot(HaveOccurred())
				Expect(tf).To(Equal("rstf"))
			})

			It("Should support server anon tls", func() {
				cfg.Choria.ChoriaSecurityTokenFile = ""
				cfg.Choria.RemoteSignerTokenFile = ""
				tf, err := fw.SignerTokenFile()
				Expect(err).ToNot(HaveOccurred())
				Expect(tf).To(Equal("stf"))
			})

			It("Should fail without a token", func() {
				cfg.Choria.ChoriaSecurityTokenFile = ""
				cfg.Choria.RemoteSignerTokenFile = ""
				cfg.Choria.ServerAnonTLS = false
				tf, err := fw.SignerTokenFile()
				Expect(err).To(MatchError("no token file defined"))
				Expect(tf).To(Equal(""))
			})
		})

		Describe("SignerToken", func() {
			It("Should error when there is no way to find a token", func() {
				t, _, err := fw.SignerToken()
				Expect(t).To(BeEmpty())
				Expect(err).To(MatchError("no token file defined"))
			})

			It("Should prefer choria security seed file", func() {
				cfg.Choria.ChoriaSecurityTokenFile = filepath.Join(td, "good-server.jwt")
				t, _, err := fw.SignerToken()
				Expect(err).ToNot(HaveOccurred())
				dt, err := os.ReadFile(cfg.Choria.ChoriaSecurityTokenFile)
				Expect(err).ToNot(HaveOccurred())
				Expect(t).To(Equal(string(dt)))
			})

			It("Should support server file tokens", func() {
				cfg.Choria.ServerAnonTLS = true
				cfg.Choria.ServerTokenFile = filepath.Join(td, "good-server.jwt")
				t, _, err := fw.SignerToken()
				Expect(err).ToNot(HaveOccurred())
				dt, err := os.ReadFile(cfg.Choria.ServerTokenFile)
				Expect(err).ToNot(HaveOccurred())
				Expect(t).To(Equal(string(dt)))
			})

			It("Should support client file tokens", func() {
				cfg.Choria.ClientAnonTLS = true
				cfg.Choria.RemoteSignerTokenFile = filepath.Join(td, "good-client.jwt")
				t, _, err := fw.SignerToken()
				Expect(err).ToNot(HaveOccurred())
				dt, err := os.ReadFile(cfg.Choria.RemoteSignerTokenFile)
				Expect(err).ToNot(HaveOccurred())
				Expect(t).To(Equal(string(dt)))
			})
		})
	})

	Describe("ProvisionMode", func() {
		BeforeEach(func() {
			build.ProvisionBrokerURLs = ""
			build.ProvisionModeDefault = ""
		})

		It("Should be on only in the Server", func() {
			c := config.NewConfigForTests()
			c.Choria.SSLDir = "/nonexisting"
			c.DisableTLS = true

			fw, err := NewWithConfig(c)
			Expect(err).ToNot(HaveOccurred())

			build.ProvisionBrokerURLs = "nats://n1:4222"
			build.ProvisionModeDefault = "true"
			Expect(fw.ProvisionMode()).To(Equal(false))
			c.InitiatedByServer = true
			Expect(fw.ProvisionMode()).To(Equal(true))
		})

		It("Should use the default when not configured and brokers are compiled in", func() {
			c := config.NewConfigForTests()
			c.DisableTLS = true
			c.Choria.SSLDir = "/nonexisting"

			fw, err := NewWithConfig(c)
			Expect(err).ToNot(HaveOccurred())

			Expect(fw.ProvisionMode()).To(Equal(false))

			build.ProvisionBrokerURLs = "nats://n1:4222"
			build.ProvisionModeDefault = "true"
			c.InitiatedByServer = true

			Expect(fw.ProvisionMode()).To(Equal(true))
		})

		It("Should use the configured value when set and when brokers are compiled in", func() {
			c, err := config.NewConfig("testdata/provision.cfg")
			Expect(err).ToNot(HaveOccurred())
			c.DisableTLS = true
			c.Choria.SSLDir = "/nonexisting"

			fw, err := NewWithConfig(c)
			Expect(err).ToNot(HaveOccurred())

			build.ProvisionBrokerURLs = "nats://n1:4222"
			c.InitiatedByServer = true

			Expect(fw.ProvisionMode()).To(Equal(true))

			c.Choria.Provision = false
			build.ProvisionModeDefault = "true"

			Expect(fw.ProvisionMode()).To(Equal(false))
		})

		It("Should be true when the JWT token does not match the running identity", func() {
			c := config.NewConfigForTests()
			c.DisableTLS = true
			c.Choria.SSLDir = "/nonexisting"
			c.InitiatedByServer = true
			c.Choria.ServerAnonTLS = true
			build.ProvisionBrokerURLs = "nats://n1:4222"
			build.ProvisionModeDefault = "false"

			// no token path set here so token won't be considered
			// even though we are in server anon tls mode
			fw, err := NewWithConfig(c)
			Expect(err).ToNot(HaveOccurred())
			Expect(fw.ProvisionMode()).To(Equal(false))

			// now we create a token file with other.example.net as id and config with
			// ginkgo.example.net as id so at this point it should trigger true
			tf, err := os.CreateTemp("", "")
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove(tf.Name())

			pk, _, err := Ed25519KeyPair()
			Expect(err).ToNot(HaveOccurred())

			t, err := tokens.NewServerClaims("other.example.net", []string{"choria"}, "choria", nil, []string{}, pk, "ginkgo", time.Hour)
			Expect(err).ToNot(HaveOccurred())
			s, err := tokens.SignTokenWithKeyFile(t, "../tokens/testdata/rsa/signer-key.pem")
			Expect(err).ToNot(HaveOccurred())

			_, err = tf.WriteString(s)
			Expect(err).ToNot(HaveOccurred())
			tf.Close()

			c.Choria.ServerTokenFile = tf.Name()

			Expect(fw.ProvisionMode()).To(Equal(true))
		})

		It("Should be false if there are no brokers", func() {
			c, err := config.NewConfig("testdata/provision.cfg")
			Expect(err).ToNot(HaveOccurred())
			c.DisableTLS = true
			c.Choria.SSLDir = "/nonexisting"

			fw, err := NewWithConfig(c)
			Expect(err).ToNot(HaveOccurred())

			c.InitiatedByServer = true

			Expect(fw.ProvisionMode()).To(Equal(false))
		})
	})
})
