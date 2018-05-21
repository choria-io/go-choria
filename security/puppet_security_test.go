package security

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/choria-io/go-choria/config"
	srvcache "github.com/choria-io/go-choria/srvcache"
	"github.com/sirupsen/logrus"

	gomock "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PuppetSSL", func() {
	var mockctl *gomock.Controller
	var settings *MocksettingsProvider
	var cfg *config.Config
	var err error
	var prov *PuppetSecurity
	var l *logrus.Logger

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())
		settings = NewMocksettingsProvider(mockctl)

		cfg, err = config.NewDefaultConfig()
		Expect(err).ToNot(HaveOccurred())
		cfg.Choria.SSLDir = filepath.Join("testdata", "good")
		cfg.OverrideCertname = "rip.mcollective"

		l = logrus.New()
		l.Out = ioutil.Discard

		prov, err = NewPuppetSecurity(settings, cfg, l.WithFields(logrus.Fields{}))
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		mockctl.Finish()
		os.Setenv("MCOLLECTIVE_CERTNAME", "rip.mcollective")
	})

	It("Should impliment the provider interface", func() {
		f := func(p Provider) {}
		f(prov)
	})

	Describe("Validate", func() {
		It("Should handle missing files", func() {
			cfg.Choria.SSLDir = filepath.Join("testdata", "allmissing")
			cfg.OverrideCertname = "test.mcollective"
			prov, err = NewPuppetSecurity(settings, cfg, l.WithFields(logrus.Fields{}))

			Expect(err).ToNot(HaveOccurred())

			settings.EXPECT().Getuid().Return(500).AnyTimes()
			errs, ok := prov.Validate()

			Expect(ok).To(BeFalse())
			Expect(errs).To(HaveLen(3))
			Expect(errs[0]).To(Equal(fmt.Sprintf("public certificate %s does not exist", filepath.Join(cfg.Choria.SSLDir, "certs", "test.mcollective.pem"))))
			Expect(errs[1]).To(Equal(fmt.Sprintf("private key %s does not exist", filepath.Join(cfg.Choria.SSLDir, "private_keys", "test.mcollective.pem"))))
			Expect(errs[2]).To(Equal(fmt.Sprintf("CA %s does not exist", filepath.Join(cfg.Choria.SSLDir, "certs", "ca.pem"))))
		})

		It("Should accept valid directories", func() {
			cfg.OverrideCertname = "rip.mcollective"

			settings.EXPECT().Getuid().Return(500).AnyTimes()
			errs, ok := prov.Validate()
			Expect(errs).To(HaveLen(0))
			Expect(ok).To(BeTrue())
		})
	})

	Describe("Identity", func() {
		It("Should support OverrideCertname", func() {
			cfg.OverrideCertname = "bob.choria"
			prov.reinit()

			Expect(prov.Identity()).To(Equal("bob.choria"))
		})

		It("Should support MCOLLECTIVE_CERTNAME", func() {
			cfg.OverrideCertname = ""
			os.Setenv("MCOLLECTIVE_CERTNAME", "env.choria")
			Expect(prov.Identity()).To(Equal("env.choria"))
		})

		It("Should support non root users", func() {
			settings.EXPECT().Getuid().Return(500).AnyTimes()
			cfg.OverrideCertname = ""
			os.Setenv("USER", "bob")
			os.Unsetenv("MCOLLECTIVE_CERTNAME")
			Expect(prov.Identity()).To(Equal("bob.mcollective"))
		})

		It("Should support root users", func() {
			settings.EXPECT().Getuid().Return(0).AnyTimes()
			os.Unsetenv("MCOLLECTIVE_CERTNAME")
			cfg.Identity = "node.example.net"
			cfg.OverrideCertname = ""
			Expect(prov.Identity()).To(Equal("node.example.net"))
		})
	})

	Describe("cachePath", func() {
		It("Should get the right cache path", func() {
			path, err := prov.cachePath("rip.mcollective")
			Expect(err).ToNot(HaveOccurred())
			Expect(path).To(Equal(filepath.FromSlash(filepath.Join(cfg.Choria.SSLDir, "choria_security", "public_certs", "rip.mcollective.pem"))))
		})
	})

	Describe("certCacheDir", func() {
		It("Should determine the right directory", func() {
			path, err := prov.certCacheDir()
			Expect(err).ToNot(HaveOccurred())

			Expect(path).To(Equal(filepath.FromSlash(filepath.Join(cfg.Choria.SSLDir, "choria_security", "public_certs"))))
		})
	})

	Describe("writeCSR", func() {
		It("should not write over existing CSRs", func() {
			prov.conf.OverrideCertname = "na.mcollective"
			prov.reinit()

			kpath, err := prov.privateKeyPath()
			Expect(err).ToNot(HaveOccurred())
			csrpath, err := prov.csrPath()
			Expect(err).ToNot(HaveOccurred())

			defer os.Remove(kpath)
			defer os.Remove(csrpath)

			key, err := prov.writePrivateKey()
			Expect(err).ToNot(HaveOccurred())

			prov.conf.OverrideCertname = "rip.mcollective"
			prov.reinit()
			err = prov.writeCSR(key, "rip.mcollective", "choria.io")

			Expect(err).To(MatchError("a certificate request already exist for rip.mcollective"))
		})

		It("Should create a valid CSR", func() {
			prov.conf.OverrideCertname = "na.mcollective"
			prov.reinit()

			kpath, err := prov.privateKeyPath()
			Expect(err).ToNot(HaveOccurred())
			csrpath, err := prov.csrPath()
			Expect(err).ToNot(HaveOccurred())

			defer os.Remove(kpath)
			defer os.Remove(csrpath)

			key, err := prov.writePrivateKey()
			Expect(err).ToNot(HaveOccurred())

			err = prov.writeCSR(key, "na.mcollective", "choria.io")
			Expect(err).ToNot(HaveOccurred())

			csrpem, err := ioutil.ReadFile(csrpath)
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
			prov.conf.OverrideCertname = "rip.mcollective"
			key, err := prov.writePrivateKey()
			Expect(err).To(MatchError("a private key already exist for rip.mcollective"))
			Expect(key).To(BeNil())
		})

		It("Should create new keys", func() {
			prov.conf.OverrideCertname = "na.mcollective"
			prov.reinit()

			path, err := prov.privateKeyPath()
			defer os.Remove(path)

			key, err := prov.writePrivateKey()
			Expect(err).ToNot(HaveOccurred())
			Expect(key).ToNot(BeNil())
			Expect(path).To(BeAnExistingFile())
		})
	})

	Describe("csrExists", func() {
		It("Should detect existing keys", func() {
			prov.conf.OverrideCertname = "rip.mcollective"
			prov.reinit()

			Expect(prov.csrExists()).To(BeTrue())
		})

		It("Should detect absent keys", func() {
			prov.conf.OverrideCertname = "na.mcollective"
			prov.reinit()

			Expect(prov.csrExists()).To(BeFalse())
		})
	})

	Describe("puppetCA", func() {
		It("Should use supplied config", func() {
			prov.conf, err = config.NewConfig("testdata/puppetca.cfg")
			Expect(err).To(Not(HaveOccurred()))
			s := prov.puppetCA()
			Expect(s.Host).To(Equal("puppet"))
			Expect(s.Port).To(Equal(8140))
			Expect(s.Scheme).To(Equal("https"))
		})

		It("Should return defaults when SRV fails", func() {
			settings.EXPECT().QuerySrvRecords([]string{"_x-puppet-ca._tcp", "_x-puppet._tcp"}).Return([]srvcache.Server{}, errors.New("simulated error"))

			s := prov.puppetCA()
			Expect(s.Host).To(Equal("puppet"))
			Expect(s.Port).To(Equal(8140))
			Expect(s.Scheme).To(Equal("https"))
		})

		It("Should use SRV records", func() {
			ans := []srvcache.Server{
				srvcache.Server{"p1", 8080, "http"},
				srvcache.Server{"p2", 8080, "http"},
			}

			settings.EXPECT().QuerySrvRecords([]string{"_x-puppet-ca._tcp", "_x-puppet._tcp"}).Return(ans, errors.New("simulated error"))

			s := prov.puppetCA()
			Expect(s.Host).To(Equal("p1"))
			Expect(s.Port).To(Equal(8080))
			Expect(s.Scheme).To(Equal("http"))

		})
	})
})
