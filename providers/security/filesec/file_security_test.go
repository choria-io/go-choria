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

func setTLS(c *Config, parent string, id string, privateExtension string) {
	if privateExtension == "" {
		privateExtension = "pem"
	}
	c.Certificate = filepath.Join(parent, "certs", fmt.Sprintf("%s.pem", id))
	c.CA = filepath.Join(parent, "certs", "ca.pem")
	c.Key = filepath.Join(parent, "private_keys", fmt.Sprintf("%s.%s", id, privateExtension))
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
		setTLS(cfg, goodStub, "rip.mcollective", "")

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
			setTLS(cfg, nonexistingStub, "test.mcollective", "")
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
			setTLS(cfg, goodStub, "rip.mcollective", "")

			errs, ok := prov.Validate()
			Expect(errs).To(BeEmpty())
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
			Expect(base64.StdEncoding.EncodeToString(sig)).To(Equal("kh5PlHXcht+FeyPdlNdpYjsW4AtOp9lRo6z3NWMcjxZq15mknzXOjkbYT1J4pp627tnlzbSC0dohP7YffGfNv5zJotx8QaIrVm2akSpWjf+M2xBf5V72f3Prn/f4dzZTP6EClM8L6SWxjQHiDamMGyT+6ZCja7Ld9TmgZV5Mx9t66pDu0OgZdi6k45/SRpLdNISnhGWpRQ5KIXgaf9gNqABNPtTstPS9i9PYNYQP6sucZPAzRa9zeyZXlKkxuBLqk4cdMUD9LgtGTy7BaAV/ZG1fzGyybw1swDAMp6x06428R+TCOaystOEbaSTqR1D1/qTDu0xUpA/izN0ZSW1g5f8K2xxv5NHoFbUyWCPGRozbrBc83uJMxhOgkeS6A2ABw1uP2vzm1zdsrj+jTj8BMHHKFn+KEkitXeImEWvWg8JvMeD8arqt0GsDBqgqGjXrlHog4y0cZvv+Yuhya2CJl77BNl08urIl0qonbCiNElB8mMvWcMoyTWo7ksWD27Ao/+oOjN+/Kek132g1PV3AK8gAnJ2RPZy/bT5qZMre0vg4PdVgL6UI3afqLOQs8AvL3KG+RFg1Lw3lG/Obmitoa2+0VJrwEN+WO6D/huGn6B7v3yzuu5UrUwkZhd2/yUbnET7OdpfalqcbTbdq/teeo7TUFNp/OrNLhVb8o63mpQQ="))
		})

		It("Should work with PKCS8 files", func() {
			setTLS(cfg, goodStub, "rip.mcollective", "p8")
			sig, err := prov.SignBytes([]byte("too many secrets"))
			Expect(err).ToNot(HaveOccurred())
			Expect(base64.StdEncoding.EncodeToString(sig)).To(Equal("kh5PlHXcht+FeyPdlNdpYjsW4AtOp9lRo6z3NWMcjxZq15mknzXOjkbYT1J4pp627tnlzbSC0dohP7YffGfNv5zJotx8QaIrVm2akSpWjf+M2xBf5V72f3Prn/f4dzZTP6EClM8L6SWxjQHiDamMGyT+6ZCja7Ld9TmgZV5Mx9t66pDu0OgZdi6k45/SRpLdNISnhGWpRQ5KIXgaf9gNqABNPtTstPS9i9PYNYQP6sucZPAzRa9zeyZXlKkxuBLqk4cdMUD9LgtGTy7BaAV/ZG1fzGyybw1swDAMp6x06428R+TCOaystOEbaSTqR1D1/qTDu0xUpA/izN0ZSW1g5f8K2xxv5NHoFbUyWCPGRozbrBc83uJMxhOgkeS6A2ABw1uP2vzm1zdsrj+jTj8BMHHKFn+KEkitXeImEWvWg8JvMeD8arqt0GsDBqgqGjXrlHog4y0cZvv+Yuhya2CJl77BNl08urIl0qonbCiNElB8mMvWcMoyTWo7ksWD27Ao/+oOjN+/Kek132g1PV3AK8gAnJ2RPZy/bT5qZMre0vg4PdVgL6UI3afqLOQs8AvL3KG+RFg1Lw3lG/Obmitoa2+0VJrwEN+WO6D/huGn6B7v3yzuu5UrUwkZhd2/yUbnET7OdpfalqcbTbdq/teeo7TUFNp/OrNLhVb8o63mpQQ="))
		})
	})

	Describe("VerifySignatureBytes", func() {
		It("Should validate correctly", func() {
			sig, err := base64.StdEncoding.DecodeString("kh5PlHXcht+FeyPdlNdpYjsW4AtOp9lRo6z3NWMcjxZq15mknzXOjkbYT1J4pp627tnlzbSC0dohP7YffGfNv5zJotx8QaIrVm2akSpWjf+M2xBf5V72f3Prn/f4dzZTP6EClM8L6SWxjQHiDamMGyT+6ZCja7Ld9TmgZV5Mx9t66pDu0OgZdi6k45/SRpLdNISnhGWpRQ5KIXgaf9gNqABNPtTstPS9i9PYNYQP6sucZPAzRa9zeyZXlKkxuBLqk4cdMUD9LgtGTy7BaAV/ZG1fzGyybw1swDAMp6x06428R+TCOaystOEbaSTqR1D1/qTDu0xUpA/izN0ZSW1g5f8K2xxv5NHoFbUyWCPGRozbrBc83uJMxhOgkeS6A2ABw1uP2vzm1zdsrj+jTj8BMHHKFn+KEkitXeImEWvWg8JvMeD8arqt0GsDBqgqGjXrlHog4y0cZvv+Yuhya2CJl77BNl08urIl0qonbCiNElB8mMvWcMoyTWo7ksWD27Ao/+oOjN+/Kek132g1PV3AK8gAnJ2RPZy/bT5qZMre0vg4PdVgL6UI3afqLOQs8AvL3KG+RFg1Lw3lG/Obmitoa2+0VJrwEN+WO6D/huGn6B7v3yzuu5UrUwkZhd2/yUbnET7OdpfalqcbTbdq/teeo7TUFNp/OrNLhVb8o63mpQQ=")
			Expect(err).ToNot(HaveOccurred())

			valid, _ := prov.VerifySignatureBytes([]byte("too many secrets"), sig, nil)
			Expect(valid).To(BeTrue())
		})

		It("Should fail for invalid sigs", func() {
			valid, _ := prov.VerifySignatureBytes([]byte("too many secrets"), []byte("meh"), nil)
			Expect(valid).To(BeFalse())
		})

		It("Should support cached certificates", func() {
			cfg.Identity = "2.mcollective"
			sig, err := base64.StdEncoding.DecodeString("a2FyZoRm4wIojH+s6qSo1ghOkeKOhayiMV44I03bRtlbYnBAZJQMzZ3GvA93w5ZDaEWndZIxxLOtzfVfLjJbF+uII2KJandWBd7nR7yByxOlpOdw0sIiBKWtiiugOnXQpbPNRyQxxyLFbmYo4bO/auraqZ8+AYl0nll8I5A0mZ2y6HIo8MdXu1+l1UTX8/Ji6G7f404Mw9CsXjo4EAfjtu/9i+cYMhqlv9lxobsuFzfA+lx/X1dtYmbW/pZ/ClnuydUdA5UV07Mf2iXjZ5c8xutLxnP+xhbQf7ql+yt9DaSX+RMwB+5ntGatRgYS/h8ihQZ970tCrCY456Uosa+xEmvfnZqoL++ja0pIgMJ4h7spQdCrjN2aXnL5IdiFROki1CPhJMaqipCb9kM8+ZtFehFh5Jx6WzekLCqkgKgZshYmJNQ4esAxWGNGxxSkiyYp5jea9qE5fLeidZrLfixNGfyXYfs75fUK9KZo3FkoPq4xFovWNr9KOXGKCT68dfg8S2SmV10CGGQ2wU1atYcpMz9Bua+3oDGpIt7OiDwOFBFHn8d8Nnm5qC2MQdn5Ys9PGCpAMh8b+P886mJlexVl18qXbcnPmM+acYFHMtZgH609w50l1zkd9MpdBqWcfH3r52VvAFh5SKAaWuVedBWea04tlx6E+UMjJvrK03aUJ9w=")
			Expect(err).ToNot(HaveOccurred())

			cert, err := os.ReadFile("../testdata/good/certs/2.mcollective.pem")
			Expect(err).ToNot(HaveOccurred())

			valid, _ := prov.VerifySignatureBytes([]byte("too many secrets"), sig, cert)
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

	Describe("publicCertPem", func() {
		It("Should return the correct pem data", func() {
			dat, err := os.ReadFile(cfg.Certificate)
			Expect(err).ToNot(HaveOccurred())
			pb, _ := pem.Decode(dat)
			Expect(err).ToNot(HaveOccurred())

			block, err := prov.publicCertPem()
			Expect(err).ToNot(HaveOccurred())
			Expect(block.Bytes).To(Equal(pb.Bytes))
		})
	})

	Describe("ShouldAllowCaller", func() {
		It("Should only accept valid certs signed by our ca", func() {
			pd, err := os.ReadFile(filepath.Join("..", "testdata", "foreign.pem"))
			Expect(err).ToNot(HaveOccurred())

			priv, err := prov.ShouldAllowCaller("foo", pd)
			Expect(err).To(HaveOccurred())
			Expect(priv).To(BeFalse())

			pub := prov.publicCertPath()
			pd, err = os.ReadFile(pub)
			Expect(err).ToNot(HaveOccurred())

			priv, err = prov.ShouldAllowCaller("rip.mcollective", pd)
			Expect(err).ToNot(HaveOccurred())
			Expect(priv).To(BeFalse())
		})

		It("Should accept privileged certs", func() {
			pd, err := os.ReadFile(filepath.Join("..", "testdata", "good", "certs", "1.privileged.mcollective.pem"))
			Expect(err).ToNot(HaveOccurred())

			priv, err := prov.ShouldAllowCaller("bob", pd)
			Expect(err).ToNot(HaveOccurred())
			Expect(priv).To(BeTrue())
		})

		It("Should not accept certs with wrong names", func() {
			pub := prov.publicCertPath()

			pd, err := os.ReadFile(pub)
			Expect(err).ToNot(HaveOccurred())

			priv, err := prov.ShouldAllowCaller("bob", pd)
			Expect(err).To(HaveOccurred())
			Expect(priv).To(BeFalse())
		})

		It("Should only accept certs that's on the allowed list", func() {
			cfg.AllowList = []string{"bob"}
			pub := prov.publicCertPath()

			pd, err := os.ReadFile(pub)
			Expect(err).ToNot(HaveOccurred())

			priv, err := prov.ShouldAllowCaller("rip.mcollective", pd)
			Expect(priv).To(BeFalse())
			Expect(err).To(MatchError("not on allow list"))
		})

		It("Should accept valid certs", func() {
			pub := prov.publicCertPath()

			pd, err := os.ReadFile(pub)
			Expect(err).ToNot(HaveOccurred())

			priv, err := prov.ShouldAllowCaller("rip.mcollective", pd)
			Expect(err).ToNot(HaveOccurred())
			Expect(priv).To(BeFalse())
		})
	})

	Describe("privateKeyExists", func() {
		It("Should detect existing keys", func() {
			setTLS(cfg, goodStub, "rip.mcollective", "")

			Expect(prov.privateKeyExists()).To(BeTrue())
		})

		It("Should detect absent keys", func() {
			setTLS(cfg, goodStub, "na.mcollective", "")

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
			Expect(prov.conf.TLSConfig.CipherSuites).To(HaveLen(1))
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
			Expect(prov.conf.TLSConfig.CurvePreferences).To(HaveLen(1))

		})

		It("Should have a default list cipher and curve list when not overridden", func() {
			prov, err := New(WithChoriaConfig(&build.Info{}, c), WithLog(l.WithFields(logrus.Fields{})))
			Expect(err).ToNot(HaveOccurred())

			Expect(prov.conf.TLSConfig.CipherSuites).To(Equal(tlssetup.DefaultCipherSuites()))

			Expect(prov.conf.TLSConfig.CurvePreferences).To(Equal(tlssetup.DefaultCurvePreferences()))
		})
	})
})
