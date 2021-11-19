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
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/tokens"
	. "github.com/onsi/ginkgo"
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

			jwtpath := filepath.Join(td, "good.jwt")

			edPub, _, err := Ed25519KeyPair()
			Expect(err).ToNot(HaveOccurred())
			claims, err := tokens.NewClientIDClaims("up=ginkgo", nil, "choria", nil, "", "Ginkgo", time.Hour, nil, edPub)
			Expect(err).ToNot(HaveOccurred())

			signed, err := tokens.SignToken(claims, privateKey)
			Expect(err).ToNot(HaveOccurred())

			err = os.WriteFile(jwtpath, []byte(signed), 0600)
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			os.RemoveAll(td)
		})

		Describe("UniqueIDFromUnverifiedToken", func() {
			It("Should extract the correct items", func() {
				cfg.Choria.RemoteSignerTokenFile = filepath.Join(td, "good.jwt")
				caller, id, token, err := fw.UniqueIDFromUnverifiedToken()
				Expect(err).ToNot(HaveOccurred())

				expectedT, err := os.ReadFile(cfg.Choria.RemoteSignerTokenFile)
				Expect(err).ToNot(HaveOccurred())

				Expect(token).To(Equal(strings.TrimSpace(string(expectedT))))
				Expect(id).To(Equal("e33bf0376d4accbb4a8fd24b2f840b2e"))
				Expect(caller).To(Equal("up=ginkgo"))
			})
		})

		Describe("SignerToken", func() {
			It("Should error when there is no way to find a token", func() {
				t, err := fw.SignerToken()
				Expect(t).To(BeEmpty())
				Expect(err).To(MatchError("no token file defined"))
			})

			It("Should support file tokens", func() {
				tf, err := os.CreateTemp("", "")
				Expect(err).ToNot(HaveOccurred())
				defer os.Remove(tf.Name())

				fmt.Fprintf(tf, "FOOFOO")
				cfg.Choria.RemoteSignerTokenFile = tf.Name()
				t, err := fw.SignerToken()
				Expect(err).ToNot(HaveOccurred())
				Expect(t).To(Equal("FOOFOO"))
			})
		})
	})

	Describe("ProvisionMode", func() {
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

		It("Should be false if there are no brokers", func() {
			c, err := config.NewConfig("testdata/provision.cfg")
			Expect(err).ToNot(HaveOccurred())
			c.DisableTLS = true
			c.Choria.SSLDir = "/nonexisting"

			fw, err := NewWithConfig(c)
			Expect(err).ToNot(HaveOccurred())

			build.ProvisionBrokerURLs = ""
			c.InitiatedByServer = true

			Expect(fw.ProvisionMode()).To(Equal(false))
		})
	})
})
