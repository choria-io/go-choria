// Copyright (c) 2020-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package puppetsec

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/srvcache"
	"github.com/sirupsen/logrus"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

func TestPuppetSecurity(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Providers/Security/Puppet")
}

var _ = Describe("PuppetSSL", func() {
	var mockctl *gomock.Controller
	var resolver *MockResolver
	var cfg *Config
	var err error
	var prov *PuppetSecurity
	var l *logrus.Logger

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())
		resolver = NewMockResolver(mockctl)
		os.Setenv("MCOLLECTIVE_CERTNAME", "rip.mcollective")

		cfg = &Config{
			SSLDir:       filepath.Join("..", "testdata", "good"),
			Identity:     "rip.mcollective",
			PuppetCAHost: "puppet",
			PuppetCAPort: 8140,
			DisableSRV:   true,
			useFakeUID:   true,
			fakeUID:      500,
		}

		l = logrus.New()
		l.SetOutput(GinkgoWriter)

		prov, err = New(WithConfig(cfg), WithResolver(resolver), WithLog(l.WithFields(logrus.Fields{})))
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	It("Should implement the provider interface", func() {
		f := func(p inter.SecurityProvider) {}
		f(prov)
		Expect(prov.Provider()).To(Equal("puppet"))
	})

	Describe("WithChoriaConfig", func() {
		It("Should disable SRV when the CA is configured", func() {
			c, err := config.NewConfig(filepath.Join("..", "testdata", "puppetca.cfg"))
			Expect(err).ToNot(HaveOccurred())

			prov, err = New(WithChoriaConfig(&build.Info{}, c), WithResolver(resolver), WithLog(l.WithFields(logrus.Fields{})))
			Expect(err).ToNot(HaveOccurred())

			Expect(prov.conf.DisableSRV).To(BeTrue())
		})

		It("Should support OverrideCertname", func() {
			c := config.NewConfigForTests()

			c.OverrideCertname = "override.choria"
			prov, err = New(WithChoriaConfig(&build.Info{}, c), WithResolver(resolver), WithLog(l.WithFields(logrus.Fields{})))
			Expect(err).ToNot(HaveOccurred())

			Expect(prov.conf.Identity).To(Equal("override.choria"))
		})

		// TODO: windows
		if runtime.GOOS != "windows" {
			It("Should fail when it cannot determine user identity", func() {
				c := config.NewConfigForTests()
				c.OverrideCertname = ""
				v := os.Getenv("USER")
				defer os.Setenv("USER", v)
				os.Unsetenv("USER")
				os.Unsetenv("MCOLLECTIVE_CERTNAME")
				_, err = New(WithChoriaConfig(&build.Info{}, c), WithResolver(resolver), WithLog(l.WithFields(logrus.Fields{})))
				Expect(err).To(MatchError("could not determine client identity, ensure USER environment variable is set"))
			})

			It("Should use the user SSL directory when not configured", func() {
				c, err := config.NewDefaultConfig()
				Expect(err).ToNot(HaveOccurred())

				prov, err = New(WithChoriaConfig(&build.Info{}, c), WithResolver(resolver), WithLog(l.WithFields(logrus.Fields{})))
				Expect(err).ToNot(HaveOccurred())

				d, err := userSSlDir()
				Expect(err).ToNot(HaveOccurred())

				Expect(prov.conf.SSLDir).To(Equal(d))
			})
		}

		It("Should copy all the relevant settings", func() {
			c, err := config.NewDefaultConfig()
			Expect(err).ToNot(HaveOccurred())

			c.DisableTLSVerify = true
			c.Choria.SSLDir = "/stub"
			c.Choria.PuppetCAHost = "stubhost"
			c.Choria.PuppetCAPort = 8080

			prov, err = New(WithChoriaConfig(&build.Info{}, c), WithResolver(resolver), WithLog(l.WithFields(logrus.Fields{})))
			Expect(err).ToNot(HaveOccurred())

			Expect(prov.conf.AllowList).To(Equal([]string{"\\.mcollective$", "\\.choria$"}))
			Expect(prov.conf.PrivilegedUsers).To(Equal([]string{"\\.privileged.mcollective$", "\\.privileged.choria$"}))
			Expect(prov.conf.DisableTLSVerify).To(BeTrue())
			Expect(prov.conf.SSLDir).To(Equal("/stub"))
			Expect(prov.conf.PuppetCAHost).To(Equal("stubhost"))
			Expect(prov.conf.PuppetCAPort).To(Equal(8080))
		})
	})

	Describe("Validate", func() {
		It("Should handle missing files", func() {
			cfg.SSLDir = filepath.Join("testdata", "allmissing")
			cfg.Identity = "test.mcollective"
			prov, err = New(WithConfig(cfg), WithResolver(resolver), WithLog(l.WithFields(logrus.Fields{})))

			Expect(err).ToNot(HaveOccurred())

			errs, ok := prov.Validate()

			Expect(ok).To(BeFalse())
			Expect(errs).To(HaveLen(3))
			Expect(errs[0]).To(Equal(fmt.Sprintf("public certificate %s does not exist", filepath.Join(cfg.SSLDir, "certs", "test.mcollective.pem"))))
			Expect(errs[1]).To(Equal(fmt.Sprintf("private key %s does not exist", filepath.Join(cfg.SSLDir, "private_keys", "test.mcollective.pem"))))
			Expect(errs[2]).To(Equal(fmt.Sprintf("CA %s does not exist", filepath.Join(cfg.SSLDir, "certs", "ca.pem"))))
		})

		It("Should accept valid directories", func() {
			cfg.Identity = "rip.mcollective"
			errs, ok := prov.Validate()
			Expect(errs).To(BeEmpty())
			Expect(ok).To(BeTrue())
		})
	})

	Describe("Identity", func() {
		It("Should support OverrideCertname", func() {
			cfg.Identity = "bob.choria"
			prov.reinit()

			Expect(prov.Identity()).To(Equal("bob.choria"))
		})
	})

	Describe("writeCSR", func() {
		It("should not write over existing CSRs", func() {
			cfg.Identity = "na.mcollective"
			prov.reinit()

			kpath := prov.privateKeyPath()
			csrpath := prov.csrPath()

			defer os.Remove(kpath)
			defer os.Remove(csrpath)

			key, err := prov.writePrivateKey()
			Expect(err).ToNot(HaveOccurred())

			prov.conf.Identity = "rip.mcollective"
			prov.reinit()
			_, err = prov.writeCSR(key, "rip.mcollective", "choria.io")

			Expect(err).To(MatchError("a certificate request already exist for rip.mcollective"))
		})

		It("Should create a valid CSR", func() {
			prov.conf.Identity = "na.mcollective"
			prov.reinit()

			kpath := prov.privateKeyPath()
			csrpath := prov.csrPath()

			defer os.Remove(kpath)
			defer os.Remove(csrpath)

			key, err := prov.writePrivateKey()
			Expect(err).ToNot(HaveOccurred())

			_, err = prov.writeCSR(key, "na.mcollective", "choria.io")
			Expect(err).ToNot(HaveOccurred())

			csrpem, err := os.ReadFile(csrpath)
			Expect(err).ToNot(HaveOccurred())

			pb, _ := pem.Decode(csrpem)

			req, err := x509.ParseCertificateRequest(pb.Bytes)
			Expect(err).ToNot(HaveOccurred())
			Expect(req.Subject.CommonName).To(Equal("na.mcollective"))
			Expect(req.Subject.OrganizationalUnit).To(Equal([]string{"choria.io"}))
		})
	})

	Describe("writePrivateKey", func() {
		It("Should not write over existing private keys", func() {
			cfg.Identity = "rip.mcollective"
			key, err := prov.writePrivateKey()
			Expect(err).To(MatchError("a private key already exist for rip.mcollective"))
			Expect(key).To(BeNil())
		})

		It("Should create new keys", func() {
			cfg.Identity = "na.mcollective"
			prov.reinit()

			path := prov.privateKeyPath()
			defer os.Remove(path)

			key, err := prov.writePrivateKey()
			Expect(err).ToNot(HaveOccurred())
			Expect(key).ToNot(BeNil())
			Expect(path).To(BeAnExistingFile())
		})
	})

	Describe("csrExists", func() {
		It("Should detect existing keys", func() {
			cfg.Identity = "rip.mcollective"
			prov.reinit()

			Expect(prov.csrExists()).To(BeTrue())
		})

		It("Should detect absent keys", func() {
			cfg.Identity = "na.mcollective"
			prov.reinit()

			Expect(prov.csrExists()).To(BeFalse())
		})
	})

	Describe("puppetCA", func() {
		It("Should use supplied config when SRV is disabled", func() {
			cfg.DisableSRV = true
			s := prov.puppetCA()
			Expect(s.Host()).To(Equal("puppet"))
			Expect(s.Port()).To(Equal(uint16(8140)))
			Expect(s.Scheme()).To(Equal("https"))
		})

		It("Should use supplied config when no srv resolver is given", func() {
			prov, err = New(WithConfig(cfg), WithLog(l.WithFields(logrus.Fields{})))
			Expect(err).ToNot(HaveOccurred())

			resolver.EXPECT().QuerySrvRecords(gomock.Any()).Times(0)

			s := prov.puppetCA()
			Expect(s.Host()).To(Equal("puppet"))
			Expect(s.Port()).To(Equal(uint16(8140)))
			Expect(s.Scheme()).To(Equal("https"))
		})

		It("Should return defaults when SRV fails", func() {
			resolver.EXPECT().QuerySrvRecords([]string{"_x-puppet-ca._tcp", "_x-puppet._tcp"}).Return(srvcache.NewServers(), errors.New("simulated error"))

			cfg.DisableSRV = false
			s := prov.puppetCA()
			Expect(s.Host()).To(Equal("puppet"))
			Expect(s.Port()).To(Equal(uint16(8140)))
			Expect(s.Scheme()).To(Equal("https"))
		})

		It("Should use SRV records", func() {
			ans := srvcache.NewServers(
				srvcache.NewServer("p1", 8080, "http"),
				srvcache.NewServer("p2", 8081, "https"),
			)

			resolver.EXPECT().QuerySrvRecords([]string{"_x-puppet-ca._tcp", "_x-puppet._tcp"}).Return(ans, nil)
			cfg.DisableSRV = false

			s := prov.puppetCA()
			Expect(s.Host()).To(Equal("p1"))
			Expect(s.Port()).To(Equal(uint16(8080)))
			Expect(s.Scheme()).To(Equal("http"))

		})
	})
})
