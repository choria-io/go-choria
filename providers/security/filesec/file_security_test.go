// Copyright (c) 2020-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package filesec

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/tlssetup"

	"github.com/choria-io/go-choria/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestFileSecurity(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Providers/Security/File")
}

func setSSL(c *Config, parent string, id string, private_extension string) {
	if private_extension == "" {
		private_extension = "pem"
	}
	c.Certificate = filepath.Join(parent, "certs", fmt.Sprintf("%s.pem", id))
	c.CA = filepath.Join(parent, "certs", "ca.pem")
	c.Key = filepath.Join(parent, "private_keys", fmt.Sprintf("%s.%s", id, private_extension))
	c.AllowList = []string{"\\.mcollective$"}
	c.PrivilegedUsers = []string{"\\.privileged.mcollective$"}
	c.DisableTLSVerify = false
	c.Identity = id

	useFakeUID = true
	fakeUID = 500
}

var _ = Describe("FileSecurity", func() {
	var cfg *Config
	var err error
	var prov *FileSecurity
	var l *logrus.Logger

	var goodStub string
	var nonexistingStub string

	BeforeEach(func() {
		os.Setenv("MCOLLECTIVE_CERTNAME", "rip.mcollective")

		goodStub = filepath.Join("..", "testdata", "good")
		nonexistingStub = filepath.Join("..", "testdata", "nonexisting")

		cfg = &Config{}
		Expect(err).ToNot(HaveOccurred())
		setSSL(cfg, goodStub, "rip.mcollective", "")

		l = logrus.New()

		l.Out = io.Discard

		prov, err = New(WithConfig(cfg), WithLog(l.WithFields(logrus.Fields{})))
		Expect(err).ToNot(HaveOccurred())
	})

	It("Should implement the provider interface", func() {
		f := func(p inter.SecurityProvider) {}
		f(prov)
		Expect(prov.Provider()).To(Equal("file"))
	})

	Describe("WithChoriaConfig", func() {
		BeforeEach(func() {
			os.Unsetenv("MCOLLECTIVE_CERTNAME")
		})

		It("Should support OverrideCertname", func() {
			c, err := config.NewDefaultConfig()
			Expect(err).ToNot(HaveOccurred())
			c.OverrideCertname = "override.choria"
			prov, err := New(WithChoriaConfig(&build.Info{}, c), WithLog(l.WithFields(logrus.Fields{})))
			Expect(err).ToNot(HaveOccurred())

			Expect(prov.conf.Identity).To(Equal("override.choria"))
		})

		It("Should support MCOLLECTIVE_CERTNAME", func() {
			os.Setenv("MCOLLECTIVE_CERTNAME", "bob.mcollective")
			c, err := config.NewDefaultConfig()
			Expect(err).ToNot(HaveOccurred())
			prov, err := New(WithChoriaConfig(&build.Info{}, c), WithLog(l.WithFields(logrus.Fields{})))
			Expect(err).ToNot(HaveOccurred())

			Expect(prov.conf.Identity).To(Equal("bob.mcollective"))
		})

		It("Should copy all the relevant settings", func() {
			c, err := config.NewDefaultConfig()
			Expect(err).ToNot(HaveOccurred())

			fakeUID = 0
			c.Choria.FileSecurityCA = "stub/ca.pem"
			c.Choria.FileSecurityCertificate = "stub/cert.pem"
			c.Choria.FileSecurityKey = "stub/key.pem"
			c.DisableTLSVerify = true
			c.Identity = "test.identity"

			prov, err := New(WithChoriaConfig(&build.Info{}, c), WithLog(l.WithFields(logrus.Fields{})))
			Expect(err).ToNot(HaveOccurred())

			Expect(prov.conf.AllowList).To(Equal([]string{"\\.mcollective$", "\\.choria$"}))
			Expect(prov.conf.PrivilegedUsers).To(Equal([]string{"\\.privileged.mcollective$", "\\.privileged.choria$"}))
			Expect(prov.conf.CA).To(Equal("stub/ca.pem"))
			Expect(prov.conf.Certificate).To(Equal("stub/cert.pem"))
			Expect(prov.conf.Key).To(Equal("stub/key.pem"))
			Expect(prov.conf.DisableTLSVerify).To(BeTrue())
			Expect(prov.conf.Identity).To(Equal("test.identity"))
		})

		It("Should support override certname", func() {
			c, err := config.NewDefaultConfig()
			Expect(err).ToNot(HaveOccurred())

			c.Choria.FileSecurityCA = "stub/ca.pem"
			c.Choria.FileSecurityCertificate = "stub/cert.pem"
			c.Choria.FileSecurityKey = "stub/key.pem"
			c.DisableTLSVerify = true
			c.Identity = "test.identity"
			c.OverrideCertname = "bob.identity"

			prov, err := New(WithChoriaConfig(&build.Info{}, c), WithLog(l.WithFields(logrus.Fields{})))
			Expect(err).ToNot(HaveOccurred())

			Expect(prov.conf.Identity).To(Equal("bob.identity"))
		})

		It("Should support root and windows", func() {
			c, err := config.NewDefaultConfig()
			Expect(err).ToNot(HaveOccurred())

			c.Choria.FileSecurityCA = "stub/ca.pem"
			c.Choria.FileSecurityCertificate = "stub/cert.pem"
			c.Choria.FileSecurityKey = "stub/key.pem"
			c.DisableTLSVerify = true
			c.Identity = "test.identity"

			useFakeOS = true
			defer func() { useFakeOS = false }()
			fakeOS = "windows"
			Expect(runtimeOs()).To(Equal("windows"))

			prov, err = New(WithChoriaConfig(&build.Info{}, c), WithLog(l.WithFields(logrus.Fields{})))
			Expect(err).ToNot(HaveOccurred())

			Expect(prov.conf.Identity).To(Equal("test.identity"))
		})
	})

	Describe("Validate", func() {
		It("Should handle missing files", func() {
			setSSL(cfg, nonexistingStub, "test.mcollective", "")
			prov, err = New(WithConfig(cfg), WithLog(l.WithFields(logrus.Fields{})))
			Expect(err).ToNot(HaveOccurred())

			errs, ok := prov.Validate()

			Expect(ok).To(BeFalse())
			Expect(errs).To(HaveLen(3))
			Expect(errs[0]).To(Equal(fmt.Sprintf("public certificate %s does not exist", cfg.Certificate)))
			Expect(errs[1]).To(Equal(fmt.Sprintf("private key %s does not exist", cfg.Key)))
			Expect(errs[2]).To(Equal(fmt.Sprintf("CA %s does not exist", cfg.CA)))
		})

		It("Should accept valid directories", func() {
			setSSL(cfg, goodStub, "rip.mcollective", "")

			errs, ok := prov.Validate()
			Expect(errs).To(HaveLen(0))
			Expect(ok).To(BeTrue())
		})
	})

	Describe("Identity", func() {
		It("Should return the identity", func() {
			cfg.Identity = "bob.choria"
			Expect(prov.Identity()).To(Equal("bob.choria"))
		})
	})

	Describe("CallerName", func() {
		It("Should return the right caller name", func() {
			cfg.Identity = "test.choria"
			Expect(prov.CallerName()).To(Equal("choria=test.choria"))
		})
	})

	Describe("CallerIdentity", func() {
		It("Should return the right caller ident", func() {
			Expect(prov.CallerIdentity("choria=test.choria")).To(Equal("test.choria"))
			Expect(prov.CallerIdentity("foo=test1.choria")).To(Equal("test1.choria"))
		})

		It("Should handle invalid caller ident", func() {
			_, err := prov.CallerIdentity("test.choria")
			Expect(err).To(MatchError("could not find a valid caller identity name in test.choria"))

			_, err = prov.CallerIdentity("fooBar=test.choria")
			Expect(err).To(MatchError("could not find a valid caller identity name in fooBar=test.choria"))
		})
	})

	Describe("SignBytes", func() {
		It("Should produce the right signature", func() {
			sig, err := prov.SignBytes([]byte("too many secrets"))
			Expect(err).ToNot(HaveOccurred())
			Expect(base64.StdEncoding.EncodeToString(sig)).To(Equal("PXj4RDHHt1oS1zF7r6EKiPyQ9oHlY4qyDP4DemZT26Hcr1A84l1p3nOVNMoksACrCdB1mW47FAwatgCB7cfCaOHsIiGOW/LQsmyE8eRpCYrV2gAHNsU6hA/CeIATwCq0Wtzp7Vc4PWR2VgrlSmihuK7sYGBJHEkillUG7F+P9c+epGJvLleM+nP7pTZVkrPqzwQ1tXFHgCNS2di5wTc5tCoJ0HHU3b31tuLGwROny3g3SsOjirrqdLDxciHYe/WzOGKByzTiqj1jjPZuuvkCzL9myr4anMBkwn1qtuqGtQ8FSwXLfgOKEwlLyf83rQ1OYWQFP+hdPJHaOlBm4iuVGjDEjla6MG081W8wpho6SqwhD1x2U9CUofQj2e0kNLQmjNK0xbIJUGSiStMcNFhIx5qoJYub40uJZkbfTE3hVp6cuOk9+yswGxfRO/RA88DBW679v8QoGeB+3RehggL2qGyRjdiPtxJj4Jt/pUAgBofrbausiIi8SUOnRSgYqpt0CLeYIiVgiNHa2EbYRfLgCsGGdVb+owAQ2Xh2VpMCelakgEBLXxBDBQ5CU8a+K992eUqDCWN6k70hDAsxXqjL+Li1J6yFjg8mAIaPLBUYgbttu47wItFZPpqlJ82cM01mELc2LyS1mChZHlo+h1q4GEbUevt0Q/VMpGNaa/WyeSQ="))
		})

		It("Should work with PKCS8 files", func() {
			setSSL(cfg, goodStub, "rip.mcollective", "p8")
			sig, err := prov.SignBytes([]byte("too many secrets"))
			Expect(err).ToNot(HaveOccurred())
			Expect(base64.StdEncoding.EncodeToString(sig)).To(Equal("PXj4RDHHt1oS1zF7r6EKiPyQ9oHlY4qyDP4DemZT26Hcr1A84l1p3nOVNMoksACrCdB1mW47FAwatgCB7cfCaOHsIiGOW/LQsmyE8eRpCYrV2gAHNsU6hA/CeIATwCq0Wtzp7Vc4PWR2VgrlSmihuK7sYGBJHEkillUG7F+P9c+epGJvLleM+nP7pTZVkrPqzwQ1tXFHgCNS2di5wTc5tCoJ0HHU3b31tuLGwROny3g3SsOjirrqdLDxciHYe/WzOGKByzTiqj1jjPZuuvkCzL9myr4anMBkwn1qtuqGtQ8FSwXLfgOKEwlLyf83rQ1OYWQFP+hdPJHaOlBm4iuVGjDEjla6MG081W8wpho6SqwhD1x2U9CUofQj2e0kNLQmjNK0xbIJUGSiStMcNFhIx5qoJYub40uJZkbfTE3hVp6cuOk9+yswGxfRO/RA88DBW679v8QoGeB+3RehggL2qGyRjdiPtxJj4Jt/pUAgBofrbausiIi8SUOnRSgYqpt0CLeYIiVgiNHa2EbYRfLgCsGGdVb+owAQ2Xh2VpMCelakgEBLXxBDBQ5CU8a+K992eUqDCWN6k70hDAsxXqjL+Li1J6yFjg8mAIaPLBUYgbttu47wItFZPpqlJ82cM01mELc2LyS1mChZHlo+h1q4GEbUevt0Q/VMpGNaa/WyeSQ="))
		})
	})

	Describe("VerifyByteSignature", func() {
		It("Should validate correctly", func() {
			sig, err := base64.StdEncoding.DecodeString("PXj4RDHHt1oS1zF7r6EKiPyQ9oHlY4qyDP4DemZT26Hcr1A84l1p3nOVNMoksACrCdB1mW47FAwatgCB7cfCaOHsIiGOW/LQsmyE8eRpCYrV2gAHNsU6hA/CeIATwCq0Wtzp7Vc4PWR2VgrlSmihuK7sYGBJHEkillUG7F+P9c+epGJvLleM+nP7pTZVkrPqzwQ1tXFHgCNS2di5wTc5tCoJ0HHU3b31tuLGwROny3g3SsOjirrqdLDxciHYe/WzOGKByzTiqj1jjPZuuvkCzL9myr4anMBkwn1qtuqGtQ8FSwXLfgOKEwlLyf83rQ1OYWQFP+hdPJHaOlBm4iuVGjDEjla6MG081W8wpho6SqwhD1x2U9CUofQj2e0kNLQmjNK0xbIJUGSiStMcNFhIx5qoJYub40uJZkbfTE3hVp6cuOk9+yswGxfRO/RA88DBW679v8QoGeB+3RehggL2qGyRjdiPtxJj4Jt/pUAgBofrbausiIi8SUOnRSgYqpt0CLeYIiVgiNHa2EbYRfLgCsGGdVb+owAQ2Xh2VpMCelakgEBLXxBDBQ5CU8a+K992eUqDCWN6k70hDAsxXqjL+Li1J6yFjg8mAIaPLBUYgbttu47wItFZPpqlJ82cM01mELc2LyS1mChZHlo+h1q4GEbUevt0Q/VMpGNaa/WyeSQ=")
			Expect(err).ToNot(HaveOccurred())

			valid, _ := prov.VerifyByteSignature([]byte("too many secrets"), sig, nil)
			Expect(valid).To(BeTrue())
		})

		It("Should fail for invalid sigs", func() {
			valid, _ := prov.VerifyByteSignature([]byte("too many secrets"), []byte("meh"), nil)
			Expect(valid).To(BeFalse())
		})

		It("Should support cached certificates", func() {
			cfg.Identity = "2.mcollective"

			sig, err := base64.StdEncoding.DecodeString("Zq1F2bdXOAvB5Ca+iYCZ/BLYz2ZzbQP/V8kwQY0E3cuDrBDArX7UhUnBakzN+Msr7UyF+EkYmzvIi4KHnFBrgi7otM8Q5YMh5IT+IPaoHj3Rj/jorqD4g8ltZINqCUBWDN4wvSG98SxLyawV69gAK4SnP+oy7SU7zxuQiPwIMJ7lVoiQ3t+tiQAHUxeykQPw7WElLb+wPTb1k4DM3yRkijA9OeUk+3SVyl2sTCu5h/Lg0lcI372bkLDESlnhnvw7yuLD2SSncrEQrBdv/N2yEpY2fx1UKGlTrn9GH4MGA1GuzE1F87RH9P8ieeul6vI13BkBAlMk5KaGlmWpgiGri5UjCHHXMxEnXfwUcKFE+E6yVg4SbrJknkuJzNJduypMIep7YOnPHVLNIBZLuOUdJrRgBQ+Yb9mxPnEQHhOHeN0XHUcseRJEISqPkagpNx1xhOb7g3hsNyEvqibT/DZsc/2hyU2I/wG9fl26CnN9c12r1zInyCQYsU/wuIvjDtRZvTpLGJSJdgjSmTPzGmA/fKpAfOWObdsoLeorjF/pNweuc0x0JZMsBrZauldLL53wnnvllsFEmIAxs+RusoJ2UfW7WugZ7lXGISHTef6IHjukHgDBSbeGawVCnAgPbPz1dy42x04koUW3Bmz89fJ4/j+e49ijz7z3W/IercNeke4=")
			Expect(err).ToNot(HaveOccurred())

			cert, err := os.ReadFile("../testdata/good/certs/2.mcollective.pem")
			Expect(err).ToNot(HaveOccurred())

			valid, _ := prov.VerifyByteSignature([]byte("too many secrets"), sig, cert)
			Expect(valid).To(BeTrue())
		})
	})

	Describe("ChecksumBytes", func() {
		It("Should produce the right checksum", func() {
			sum, err := base64.StdEncoding.DecodeString("Yk+jdKdZ3v8E2p6dmbfn+ZN9lBBAHEIcOMp4lzuYKTo=")
			Expect(err).ToNot(HaveOccurred())

			Expect(prov.ChecksumBytes([]byte("too many secrets"))).To(Equal(sum))
		})
	})

	Describe("TLSConfig", func() {
		It("Should produce a valid TLS Config", func() {
			c, err := prov.TLSConfig()
			Expect(err).ToNot(HaveOccurred())

			Expect(c.InsecureSkipVerify).To(BeFalse())
			Expect(err).ToNot(HaveOccurred())

			pub := prov.publicCertPath()
			pri := prov.privateKeyPath()

			cert, err := tls.LoadX509KeyPair(pub, pri)
			Expect(err).ToNot(HaveOccurred())

			Expect(c.Certificates).To(HaveLen(1))
			Expect(c.Certificates[0].Certificate).To(Equal(cert.Certificate))
		})

		It("Should support disabling tls verify", func() {
			cfg.DisableTLSVerify = true

			c, err := prov.TLSConfig()
			Expect(err).ToNot(HaveOccurred())

			Expect(c.InsecureSkipVerify).To(BeTrue())

		})
	})

	Describe("VerifyCertificate", func() {
		var pem []byte

		BeforeEach(func() {
			pub := prov.publicCertPath()
			pem, err = os.ReadFile(pub)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should fail for foreign certs", func() {
			pem, err = os.ReadFile(filepath.Join("..", "testdata", "foreign.pem"))
			Expect(err).ToNot(HaveOccurred())
			err := prov.VerifyCertificate(pem, "rip.mcollective")
			Expect(err).To(MatchError("x509: certificate signed by unknown authority"))

		})

		It("Should fail for invalid names", func() {
			err := prov.VerifyCertificate(pem, "bob")
			Expect(err).To(MatchError("x509: certificate is valid for rip.mcollective, not bob"))
		})

		It("Should accept valid certs", func() {
			err := prov.VerifyCertificate(pem, "rip.mcollective")
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should work with client provided intermediate chains", func() {
			c, err := config.NewDefaultConfig()
			Expect(err).ToNot(HaveOccurred())

			c.Choria.FileSecurityCA = filepath.Join("..", "testdata", "intermediate", "certs", "ca.pem")

			prov, err := New(WithChoriaConfig(&build.Info{}, c), WithLog(l.WithFields(logrus.Fields{})))
			Expect(err).ToNot(HaveOccurred())

			pem, err = os.ReadFile(filepath.Join("..", "testdata", "intermediate", "certs", "rip.mcollective.pem"))
			Expect(err).ToNot(HaveOccurred())

			err = prov.VerifyCertificate(pem, "rip.mcollective")
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should work with server side ca intermediate chains", func() {
			c, err := config.NewDefaultConfig()
			Expect(err).ToNot(HaveOccurred())

			c.Choria.FileSecurityCA = filepath.Join("..", "testdata", "intermediate", "certs", "ca_chain_ca.pem")

			prov, err := New(WithChoriaConfig(&build.Info{}, c), WithLog(l.WithFields(logrus.Fields{})))
			Expect(err).ToNot(HaveOccurred())

			pem, err = os.ReadFile(filepath.Join("..", "testdata", "intermediate", "certs", "ca_chain_rip.mcollective.pem"))
			Expect(err).ToNot(HaveOccurred())

			err = prov.VerifyCertificate(pem, "rip.mcollective")
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should work with email addresses", func() {
			c, err := config.NewDefaultConfig()
			Expect(err).ToNot(HaveOccurred())

			c.Choria.FileSecurityCA = filepath.Join("..", "testdata", "intermediate", "certs", "ca_chain_ca.pem")

			prov, err := New(WithChoriaConfig(&build.Info{}, c), WithLog(l.WithFields(logrus.Fields{})))
			Expect(err).ToNot(HaveOccurred())

			pem, err = os.ReadFile(filepath.Join("..", "testdata", "intermediate", "certs", "email-chain-rip.mcollective.pem"))
			Expect(err).ToNot(HaveOccurred())

			err = prov.VerifyCertificate(pem, "email:test@choria-io.com")
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should not work with wrong addresses", func() {
			c, err := config.NewDefaultConfig()
			Expect(err).ToNot(HaveOccurred())

			c.Choria.FileSecurityCA = filepath.Join("..", "testdata", "intermediate", "certs", "ca_chain_ca.pem")

			prov, err := New(WithChoriaConfig(&build.Info{}, c), WithLog(l.WithFields(logrus.Fields{})))
			Expect(err).ToNot(HaveOccurred())

			pem, err = os.ReadFile(filepath.Join("..", "testdata", "intermediate", "certs", "email-chain-rip.mcollective.pem"))
			Expect(err).ToNot(HaveOccurred())

			err = prov.VerifyCertificate(pem, "email:bad@choria-io.com")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("PublicCertPem", func() {
		It("Should return the correct pem data", func() {
			dat, err := os.ReadFile(cfg.Certificate)
			Expect(err).ToNot(HaveOccurred())
			pb, _ := pem.Decode(dat)
			Expect(err).ToNot(HaveOccurred())

			block, err := prov.PublicCertPem()
			Expect(err).ToNot(HaveOccurred())
			Expect(block.Bytes).To(Equal(pb.Bytes))
		})
	})

	Describe("ShouldAllowCaller", func() {
		It("Should only accept valid certs signed by our ca", func() {
			pd, err := os.ReadFile(filepath.Join("..", "testdata", "foreign.pem"))
			Expect(err).ToNot(HaveOccurred())

			priv, err := prov.ShouldAllowCaller(pd, "foo")
			Expect(err).To(HaveOccurred())
			Expect(priv).To(BeFalse())

			pub := prov.publicCertPath()
			pd, err = os.ReadFile(pub)
			Expect(err).ToNot(HaveOccurred())

			priv, err = prov.ShouldAllowCaller(pd, "rip.mcollective")
			Expect(err).ToNot(HaveOccurred())
			Expect(priv).To(BeFalse())
		})

		It("Should accept privileged certs", func() {
			pd, err := os.ReadFile(filepath.Join("..", "testdata", "good", "certs", "1.privileged.mcollective.pem"))
			Expect(err).ToNot(HaveOccurred())

			priv, err := prov.ShouldAllowCaller(pd, "bob")
			Expect(err).ToNot(HaveOccurred())
			Expect(priv).To(BeTrue())
		})

		It("Should not accept certs with wrong names", func() {
			pub := prov.publicCertPath()

			pd, err := os.ReadFile(pub)
			Expect(err).ToNot(HaveOccurred())

			priv, err := prov.ShouldAllowCaller(pd, "bob")
			Expect(err).To(HaveOccurred())
			Expect(priv).To(BeFalse())
		})

		It("Should only accept certs that's on the allowed list", func() {
			cfg.AllowList = []string{"bob"}
			pub := prov.publicCertPath()

			pd, err := os.ReadFile(pub)
			Expect(err).ToNot(HaveOccurred())

			priv, err := prov.ShouldAllowCaller(pd, "rip.mcollective")
			Expect(priv).To(BeFalse())
			Expect(err).To(MatchError("not on allow list"))
		})

		It("Should accept valid certs", func() {
			pub := prov.publicCertPath()

			pd, err := os.ReadFile(pub)
			Expect(err).ToNot(HaveOccurred())

			priv, err := prov.ShouldAllowCaller(pd, "rip.mcollective")
			Expect(err).ToNot(HaveOccurred())
			Expect(priv).To(BeFalse())
		})
	})

	Describe("privateKeyExists", func() {
		It("Should detect existing keys", func() {
			setSSL(cfg, goodStub, "rip.mcollective", "")

			Expect(prov.privateKeyExists()).To(BeTrue())
		})

		It("Should detect absent keys", func() {
			setSSL(cfg, goodStub, "na.mcollective", "")

			Expect(prov.privateKeyExists()).To(BeFalse())
		})
	})

	Describe("Configurable CipherSuites", func() {
		var cipher string
		var curve string
		var c *config.Config

		BeforeEach(func() {
			_c, err := config.NewDefaultConfig()
			Expect(err).ToNot(HaveOccurred())

			c = _c
		})

		It("Should work with just one cipher", func() {
			for _, cm := range tls.CipherSuites() {
				cipher = cm.Name
				break
			}

			c.Choria.CipherSuites = []string{cipher}

			prov, err := New(WithChoriaConfig(&build.Info{}, c), WithLog(l.WithFields(logrus.Fields{})))
			Expect(err).ToNot(HaveOccurred())

			Expect(prov.conf.TLSConfig.CipherSuites).ToNot(BeNil())
			Expect(len(prov.conf.TLSConfig.CipherSuites)).To(Equal(1))
		})

		It("Should work with one curve", func() {
			for cp := range tlssetup.CurvePreferenceMap {
				curve = cp
				break
			}

			c.Choria.ECCCurves = []string{curve}

			prov, err := New(WithChoriaConfig(&build.Info{}, c), WithLog(l.WithFields(logrus.Fields{})))
			Expect(err).ToNot(HaveOccurred())

			Expect(prov.conf.TLSConfig.CurvePreferences).ToNot(BeNil())
			Expect(len(prov.conf.TLSConfig.CurvePreferences)).To(Equal(1))

		})

		It("Should have a default list cipher and curve list when not overridden", func() {
			prov, err := New(WithChoriaConfig(&build.Info{}, c), WithLog(l.WithFields(logrus.Fields{})))
			Expect(err).ToNot(HaveOccurred())

			Expect(prov.conf.TLSConfig.CipherSuites).To(Equal(tlssetup.DefaultCipherSuites()))

			Expect(prov.conf.TLSConfig.CurvePreferences).To(Equal(tlssetup.DefaultCurvePreferences()))
		})
	})
})
