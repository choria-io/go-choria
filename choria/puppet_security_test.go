package choria

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"

	"github.com/sirupsen/logrus"

	gomock "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PuppetSSL", func() {
	var mockctl *gomock.Controller
	var settings *MocksettingsProvider
	var cfg *Config
	var err error
	var prov *PuppetSecurity

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())
		settings = NewMocksettingsProvider(mockctl)

		cfg, err = NewDefaultConfig()
		Expect(err).ToNot(HaveOccurred())
		cfg.Choria.SSLDir = filepath.Join("testdata", "good")

		l := logrus.New()
		l.Out = ioutil.Discard

		prov, err = NewPuppetSecurity(settings, cfg, l.WithFields(logrus.Fields{}))
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		mockctl.Finish()
		os.Setenv("MCOLLECTIVE_CERTNAME", "rip.mcollective")
	})

	It("Should impliment the provider interface", func() {
		f := func(p SecurityProvider) {}
		f(prov)
	})

	Describe("Validate", func() {
		It("Should handle missing ssl dirs", func() {
			cfg.Choria.SSLDir = ""
			settings.EXPECT().Getuid().Return(0).Times(1)
			settings.EXPECT().PuppetSetting("ssldir").Return(filepath.Join("testdata", "allmissing"), fmt.Errorf("error invoking puppet")).Times(1)

			errs, ok := prov.Validate()
			Expect(errs[0]).To(Equal("SSL Directory does not exist: error invoking puppet"))
			Expect(ok).To(BeFalse())
		})

		It("Should handle missing files", func() {
			cfg.Choria.SSLDir = filepath.Join("testdata", "allmissing")
			cfg.OverrideCertname = "test.mcollective"

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
			Expect(prov.Identity()).To(Equal("bob.choria"))
		})

		It("Should support MCOLLECTIVE_CERTNAME", func() {
			cfg.OverrideCertname = ""
			os.Setenv("MCOLLECTIVE_CERTNAME", "env.choria")
			Expect(prov.Identity()).To(Equal("env.choria"))
		})

		It("Should support non root users", func() {
			settings.EXPECT().Getuid().Return(500).AnyTimes()
			os.Setenv("USER", "bob")
			os.Unsetenv("MCOLLECTIVE_CERTNAME")
			Expect(prov.Identity()).To(Equal("bob.mcollective"))
		})

		It("Should support root users", func() {
			settings.EXPECT().Getuid().Return(0).AnyTimes()
			os.Unsetenv("MCOLLECTIVE_CERTNAME")
			cfg.Identity = "node.example.net"
			Expect(prov.Identity()).To(Equal("node.example.net"))
		})
	})

	Describe("CallerName", func() {
		It("Should return the right caller name", func() {
			cfg.OverrideCertname = "test.choria"
			Expect(prov.CallerName()).To(Equal("choria=test.choria"))
		})
	})

	Describe("CallerIdentity", func() {
		It("Should return the right caller ident", func() {
			Expect(prov.CallerIdentity("choria=test.choria")).To(Equal("test.choria"))
		})

		It("Should handle invalid caller ident", func() {
			_, err := prov.CallerIdentity("test.choria")
			Expect(err).To(MatchError("could not find a valid caller identity name in test.choria"))
		})
	})

	Describe("SignBytes", func() {
		It("Should produce the right signature", func() {
			sig, err := prov.SignBytes([]byte("too many secrets"))
			Expect(err).ToNot(HaveOccurred())
			Expect(base64.StdEncoding.EncodeToString(sig)).To(Equal("PXj4RDHHt1oS1zF7r6EKiPyQ9oHlY4qyDP4DemZT26Hcr1A84l1p3nOVNMoksACrCdB1mW47FAwatgCB7cfCaOHsIiGOW/LQsmyE8eRpCYrV2gAHNsU6hA/CeIATwCq0Wtzp7Vc4PWR2VgrlSmihuK7sYGBJHEkillUG7F+P9c+epGJvLleM+nP7pTZVkrPqzwQ1tXFHgCNS2di5wTc5tCoJ0HHU3b31tuLGwROny3g3SsOjirrqdLDxciHYe/WzOGKByzTiqj1jjPZuuvkCzL9myr4anMBkwn1qtuqGtQ8FSwXLfgOKEwlLyf83rQ1OYWQFP+hdPJHaOlBm4iuVGjDEjla6MG081W8wpho6SqwhD1x2U9CUofQj2e0kNLQmjNK0xbIJUGSiStMcNFhIx5qoJYub40uJZkbfTE3hVp6cuOk9+yswGxfRO/RA88DBW679v8QoGeB+3RehggL2qGyRjdiPtxJj4Jt/pUAgBofrbausiIi8SUOnRSgYqpt0CLeYIiVgiNHa2EbYRfLgCsGGdVb+owAQ2Xh2VpMCelakgEBLXxBDBQ5CU8a+K992eUqDCWN6k70hDAsxXqjL+Li1J6yFjg8mAIaPLBUYgbttu47wItFZPpqlJ82cM01mELc2LyS1mChZHlo+h1q4GEbUevt0Q/VMpGNaa/WyeSQ="))
		})
	})

	Describe("VerifyByteSignature", func() {
		It("Should validate correctly", func() {
			sig, err := base64.StdEncoding.DecodeString("PXj4RDHHt1oS1zF7r6EKiPyQ9oHlY4qyDP4DemZT26Hcr1A84l1p3nOVNMoksACrCdB1mW47FAwatgCB7cfCaOHsIiGOW/LQsmyE8eRpCYrV2gAHNsU6hA/CeIATwCq0Wtzp7Vc4PWR2VgrlSmihuK7sYGBJHEkillUG7F+P9c+epGJvLleM+nP7pTZVkrPqzwQ1tXFHgCNS2di5wTc5tCoJ0HHU3b31tuLGwROny3g3SsOjirrqdLDxciHYe/WzOGKByzTiqj1jjPZuuvkCzL9myr4anMBkwn1qtuqGtQ8FSwXLfgOKEwlLyf83rQ1OYWQFP+hdPJHaOlBm4iuVGjDEjla6MG081W8wpho6SqwhD1x2U9CUofQj2e0kNLQmjNK0xbIJUGSiStMcNFhIx5qoJYub40uJZkbfTE3hVp6cuOk9+yswGxfRO/RA88DBW679v8QoGeB+3RehggL2qGyRjdiPtxJj4Jt/pUAgBofrbausiIi8SUOnRSgYqpt0CLeYIiVgiNHa2EbYRfLgCsGGdVb+owAQ2Xh2VpMCelakgEBLXxBDBQ5CU8a+K992eUqDCWN6k70hDAsxXqjL+Li1J6yFjg8mAIaPLBUYgbttu47wItFZPpqlJ82cM01mELc2LyS1mChZHlo+h1q4GEbUevt0Q/VMpGNaa/WyeSQ=")
			Expect(err).ToNot(HaveOccurred())

			valid := prov.VerifyByteSignature([]byte("too many secrets"), sig, "")
			Expect(valid).To(BeTrue())
		})

		It("Should fail for invalid sigs", func() {
			valid := prov.VerifyByteSignature([]byte("too many secrets"), []byte("meh"), "")
			Expect(valid).To(BeFalse())
		})

		It("Should support cached certificates", func() {
			cfg.OverrideCertname = "2.mcollective"
			settings.EXPECT().Getuid().Return(500).AnyTimes()

			sig, err := base64.StdEncoding.DecodeString("Zq1F2bdXOAvB5Ca+iYCZ/BLYz2ZzbQP/V8kwQY0E3cuDrBDArX7UhUnBakzN+Msr7UyF+EkYmzvIi4KHnFBrgi7otM8Q5YMh5IT+IPaoHj3Rj/jorqD4g8ltZINqCUBWDN4wvSG98SxLyawV69gAK4SnP+oy7SU7zxuQiPwIMJ7lVoiQ3t+tiQAHUxeykQPw7WElLb+wPTb1k4DM3yRkijA9OeUk+3SVyl2sTCu5h/Lg0lcI372bkLDESlnhnvw7yuLD2SSncrEQrBdv/N2yEpY2fx1UKGlTrn9GH4MGA1GuzE1F87RH9P8ieeul6vI13BkBAlMk5KaGlmWpgiGri5UjCHHXMxEnXfwUcKFE+E6yVg4SbrJknkuJzNJduypMIep7YOnPHVLNIBZLuOUdJrRgBQ+Yb9mxPnEQHhOHeN0XHUcseRJEISqPkagpNx1xhOb7g3hsNyEvqibT/DZsc/2hyU2I/wG9fl26CnN9c12r1zInyCQYsU/wuIvjDtRZvTpLGJSJdgjSmTPzGmA/fKpAfOWObdsoLeorjF/pNweuc0x0JZMsBrZauldLL53wnnvllsFEmIAxs+RusoJ2UfW7WugZ7lXGISHTef6IHjukHgDBSbeGawVCnAgPbPz1dy42x04koUW3Bmz89fJ4/j+e49ijz7z3W/IercNeke4=")
			Expect(err).ToNot(HaveOccurred())

			valid := prov.VerifyByteSignature([]byte("too many secrets"), sig, "2.mcollective")
			Expect(valid).To(BeTrue())
		})
	})

	Describe("PrivilegedVerifyByteSignature", func() {
		It("Should allow a privileged cert to act as someone else", func() {
			settings.EXPECT().Getuid().Return(500).AnyTimes()

			sig, err := base64.StdEncoding.DecodeString("4dSmCRsnZzJjqSLdXxCRufw+wh9BrbqMZfEkB9c0yLkqjuc6r2b3tl5bh28l2lm50nPcIKeMyVHh2pkhvVsnjTYVhEYGTBcJdhAf/4PQCCqllHfiD0i+EZNTC916P4C2TVFNw5kOx/qjz6KYBuBV0K0U5JG1L7rHmlSoJ1La9vs/x1RMLly91NYnPOtCpwSsAwRG6uGMnCQK/vGg+NiwIQpQrchCpVf6rrXSqqUrJzZc/SeNl42AA2EYbkq8ys79sye1w91BF07gX6n/gK/472tlTh9OK49GmLdi15oGiEOPbkCbPYm2hcWAJzdqGprCQAsYjuMfUByswxkthEw72Bp9tmSuc6P6QPLswkAeVi4NivQCm81CFEB0ZKl0WluJp5xEL9/mO9/Z/iUuvMRGQSbfIzi+8PVJeNIWsY8rzsDMdoIdwPD+vqVU7BhHxKXjAHq2nnhQCj35HuV2dN7n0MOy4A6H5kA4a8d5UVTBRMsFZ5s6Bo4/leFOlylgU2DIWq+DXdg05Zr98H9JulDM0epKEjLeowo5z5f2s7/eQymaSzdoW2zUhe9Hp0G0D8CkQUXm/RzjzLBTZ1fNQYIQGA9U6n+ApwBNHW9ClmlbbvcUb+Bw2rRHVgKM6+kUam+TLpLljuZkOY6wkk+h97aHYJyO7tOezyTuPPM5L3CDQ+M=")
			Expect(err).ToNot(HaveOccurred())

			valid := prov.PrivilegedVerifyByteSignature([]byte("too many secrets"), sig, "rip.mcollective")
			Expect(valid).To(BeTrue())
		})

		It("Should allow a cert to act as itself", func() {
			sig, err := base64.StdEncoding.DecodeString("PXj4RDHHt1oS1zF7r6EKiPyQ9oHlY4qyDP4DemZT26Hcr1A84l1p3nOVNMoksACrCdB1mW47FAwatgCB7cfCaOHsIiGOW/LQsmyE8eRpCYrV2gAHNsU6hA/CeIATwCq0Wtzp7Vc4PWR2VgrlSmihuK7sYGBJHEkillUG7F+P9c+epGJvLleM+nP7pTZVkrPqzwQ1tXFHgCNS2di5wTc5tCoJ0HHU3b31tuLGwROny3g3SsOjirrqdLDxciHYe/WzOGKByzTiqj1jjPZuuvkCzL9myr4anMBkwn1qtuqGtQ8FSwXLfgOKEwlLyf83rQ1OYWQFP+hdPJHaOlBm4iuVGjDEjla6MG081W8wpho6SqwhD1x2U9CUofQj2e0kNLQmjNK0xbIJUGSiStMcNFhIx5qoJYub40uJZkbfTE3hVp6cuOk9+yswGxfRO/RA88DBW679v8QoGeB+3RehggL2qGyRjdiPtxJj4Jt/pUAgBofrbausiIi8SUOnRSgYqpt0CLeYIiVgiNHa2EbYRfLgCsGGdVb+owAQ2Xh2VpMCelakgEBLXxBDBQ5CU8a+K992eUqDCWN6k70hDAsxXqjL+Li1J6yFjg8mAIaPLBUYgbttu47wItFZPpqlJ82cM01mELc2LyS1mChZHlo+h1q4GEbUevt0Q/VMpGNaa/WyeSQ=")
			Expect(err).ToNot(HaveOccurred())

			valid := prov.PrivilegedVerifyByteSignature([]byte("too many secrets"), sig, "rip.mcollective")
			Expect(valid).To(BeTrue())
		})

		It("Should not allow a unmatched cert", func() {
			sig, err := base64.StdEncoding.DecodeString("PXj4RDHHt1oS1zF7r6EKiPyQ9oHlY4qyDP4DemZT26Hcr1A84l1p3nOVNMoksACrCdB1mW47FAwatgCB7cfCaOHsIiGOW/LQsmyE8eRpCYrV2gAHNsU6hA/CeIATwCq0Wtzp7Vc4PWR2VgrlSmihuK7sYGBJHEkillUG7F+P9c+epGJvLleM+nP7pTZVkrPqzwQ1tXFHgCNS2di5wTc5tCoJ0HHU3b31tuLGwROny3g3SsOjirrqdLDxciHYe/WzOGKByzTiqj1jjPZuuvkCzL9myr4anMBkwn1qtuqGtQ8FSwXLfgOKEwlLyf83rQ1OYWQFP+hdPJHaOlBm4iuVGjDEjla6MG081W8wpho6SqwhD1x2U9CUofQj2e0kNLQmjNK0xbIJUGSiStMcNFhIx5qoJYub40uJZkbfTE3hVp6cuOk9+yswGxfRO/RA88DBW679v8QoGeB+3RehggL2qGyRjdiPtxJj4Jt/pUAgBofrbausiIi8SUOnRSgYqpt0CLeYIiVgiNHa2EbYRfLgCsGGdVb+owAQ2Xh2VpMCelakgEBLXxBDBQ5CU8a+K992eUqDCWN6k70hDAsxXqjL+Li1J6yFjg8mAIaPLBUYgbttu47wItFZPpqlJ82cM01mELc2LyS1mChZHlo+h1q4GEbUevt0Q/VMpGNaa/WyeSQ=")
			Expect(err).ToNot(HaveOccurred())

			valid := prov.PrivilegedVerifyByteSignature([]byte("too many secrets"), sig, "2.mcollective")
			Expect(valid).To(BeFalse())
		})
	})

	Describe("SignString", func() {
		It("Should produce the right signature", func() {
			sig, err := prov.SignString("too many secrets")
			Expect(err).ToNot(HaveOccurred())
			Expect(base64.StdEncoding.EncodeToString(sig)).To(Equal("PXj4RDHHt1oS1zF7r6EKiPyQ9oHlY4qyDP4DemZT26Hcr1A84l1p3nOVNMoksACrCdB1mW47FAwatgCB7cfCaOHsIiGOW/LQsmyE8eRpCYrV2gAHNsU6hA/CeIATwCq0Wtzp7Vc4PWR2VgrlSmihuK7sYGBJHEkillUG7F+P9c+epGJvLleM+nP7pTZVkrPqzwQ1tXFHgCNS2di5wTc5tCoJ0HHU3b31tuLGwROny3g3SsOjirrqdLDxciHYe/WzOGKByzTiqj1jjPZuuvkCzL9myr4anMBkwn1qtuqGtQ8FSwXLfgOKEwlLyf83rQ1OYWQFP+hdPJHaOlBm4iuVGjDEjla6MG081W8wpho6SqwhD1x2U9CUofQj2e0kNLQmjNK0xbIJUGSiStMcNFhIx5qoJYub40uJZkbfTE3hVp6cuOk9+yswGxfRO/RA88DBW679v8QoGeB+3RehggL2qGyRjdiPtxJj4Jt/pUAgBofrbausiIi8SUOnRSgYqpt0CLeYIiVgiNHa2EbYRfLgCsGGdVb+owAQ2Xh2VpMCelakgEBLXxBDBQ5CU8a+K992eUqDCWN6k70hDAsxXqjL+Li1J6yFjg8mAIaPLBUYgbttu47wItFZPpqlJ82cM01mELc2LyS1mChZHlo+h1q4GEbUevt0Q/VMpGNaa/WyeSQ="))

		})
	})

	Describe("VerifyStringSignature", func() {
		It("Should validate correctly", func() {
			sig, err := base64.StdEncoding.DecodeString("PXj4RDHHt1oS1zF7r6EKiPyQ9oHlY4qyDP4DemZT26Hcr1A84l1p3nOVNMoksACrCdB1mW47FAwatgCB7cfCaOHsIiGOW/LQsmyE8eRpCYrV2gAHNsU6hA/CeIATwCq0Wtzp7Vc4PWR2VgrlSmihuK7sYGBJHEkillUG7F+P9c+epGJvLleM+nP7pTZVkrPqzwQ1tXFHgCNS2di5wTc5tCoJ0HHU3b31tuLGwROny3g3SsOjirrqdLDxciHYe/WzOGKByzTiqj1jjPZuuvkCzL9myr4anMBkwn1qtuqGtQ8FSwXLfgOKEwlLyf83rQ1OYWQFP+hdPJHaOlBm4iuVGjDEjla6MG081W8wpho6SqwhD1x2U9CUofQj2e0kNLQmjNK0xbIJUGSiStMcNFhIx5qoJYub40uJZkbfTE3hVp6cuOk9+yswGxfRO/RA88DBW679v8QoGeB+3RehggL2qGyRjdiPtxJj4Jt/pUAgBofrbausiIi8SUOnRSgYqpt0CLeYIiVgiNHa2EbYRfLgCsGGdVb+owAQ2Xh2VpMCelakgEBLXxBDBQ5CU8a+K992eUqDCWN6k70hDAsxXqjL+Li1J6yFjg8mAIaPLBUYgbttu47wItFZPpqlJ82cM01mELc2LyS1mChZHlo+h1q4GEbUevt0Q/VMpGNaa/WyeSQ=")
			Expect(err).ToNot(HaveOccurred())

			valid := prov.VerifyStringSignature("too many secrets", sig, "")
			Expect(valid).To(BeTrue())
		})

		It("Should fail for invalid sigs", func() {
			valid := prov.VerifyStringSignature("too many secrets", []byte("meh"), "")
			Expect(valid).To(BeFalse())
		})
	})

	Describe("ChecksumBytes", func() {
		It("Should produce the right checksum", func() {
			sum, err := base64.StdEncoding.DecodeString("Yk+jdKdZ3v8E2p6dmbfn+ZN9lBBAHEIcOMp4lzuYKTo=")
			Expect(err).ToNot(HaveOccurred())

			Expect(prov.ChecksumBytes([]byte("too many secrets"))).To(Equal(sum))
		})
	})

	Describe("ChecksumString", func() {
		It("Should produce the right checksum", func() {
			sum, err := base64.StdEncoding.DecodeString("Yk+jdKdZ3v8E2p6dmbfn+ZN9lBBAHEIcOMp4lzuYKTo=")
			Expect(err).ToNot(HaveOccurred())

			Expect(prov.ChecksumString("too many secrets")).To(Equal(sum))
		})
	})

	Describe("TLSConfig", func() {
		It("Should produce a valid TLS Config", func() {
			c, err := prov.TLSConfig()
			Expect(err).ToNot(HaveOccurred())

			Expect(c.InsecureSkipVerify).To(BeFalse())
			Expect(err).ToNot(HaveOccurred())

			pub, err := prov.publicCertPath()
			Expect(err).ToNot(HaveOccurred())

			pri, err := prov.privateKeyPath()
			Expect(err).ToNot(HaveOccurred())

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
			pub, err := prov.publicCertPath()
			Expect(err).ToNot(HaveOccurred())
			pem, err = ioutil.ReadFile(pub)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should fail for foreign certs", func() {
			pem, err = ioutil.ReadFile(filepath.Join("testdata", "foreign.pem"))
			Expect(err).ToNot(HaveOccurred())
			err, verified := prov.VerifyCertificate(pem, "rip.mcollective")
			Expect(verified).To(BeFalse())
			Expect(err).To(MatchError("x509: certificate signed by unknown authority"))

		})

		It("Should fail for invalid names", func() {
			err, verified := prov.VerifyCertificate(pem, "bob")
			Expect(err).To(MatchError("x509: certificate is valid for rip.mcollective, not bob"))
			Expect(verified).To(BeFalse())
		})

		It("Should accept valid certs", func() {
			_, verified := prov.VerifyCertificate(pem, "rip.mcollective")
			Expect(verified).To(BeTrue())
		})
	})

	Describe("PublicCertPem", func() {
		It("Should return the correct pem data", func() {
			dat, err := ioutil.ReadFile(filepath.Join(cfg.Choria.SSLDir, "certs", "rip.mcollective.pem"))
			Expect(err).ToNot(HaveOccurred())
			pb, _ := pem.Decode(dat)
			Expect(err).ToNot(HaveOccurred())

			block, err := prov.PublicCertPem()
			Expect(err).ToNot(HaveOccurred())
			Expect(block.Bytes).To(Equal(pb.Bytes))
		})
	})

	Describe("shouldCacheClientCert", func() {
		It("Should only accept valid certs signed by our ca", func() {
			pd, err := ioutil.ReadFile(filepath.Join("testdata", "foreign.pem"))
			Expect(err).ToNot(HaveOccurred())
			Expect(prov.shouldCacheClientCert(pd, "foo")).To(BeFalse())

			pub, err := prov.publicCertPath()
			Expect(err).ToNot(HaveOccurred())
			pd, err = ioutil.ReadFile(pub)
			Expect(err).ToNot(HaveOccurred())
			Expect(prov.shouldCacheClientCert(pd, "rip.mcollective")).To(BeTrue())
		})

		It("Should cache privileged certs", func() {
			cfg.OverrideCertname = "1.privileged.mcollective"

			pub, err := prov.publicCertPath()
			Expect(err).ToNot(HaveOccurred())

			pd, err := ioutil.ReadFile(pub)
			Expect(err).ToNot(HaveOccurred())
			Expect(prov.shouldCacheClientCert(pd, "1.privileged.mcollective")).To(BeTrue())
		})

		It("Should not cache certs with wrong names", func() {
			pub, err := prov.publicCertPath()
			Expect(err).ToNot(HaveOccurred())

			pd, err := ioutil.ReadFile(pub)
			Expect(err).ToNot(HaveOccurred())
			Expect(prov.shouldCacheClientCert(pd, "bob")).To(BeFalse())
		})

		It("Should only cache certs thats on the allowed list", func() {
			cfg.Choria.CertnameWhitelist = []string{"bob"}
			pub, err := prov.publicCertPath()
			Expect(err).ToNot(HaveOccurred())

			pd, err := ioutil.ReadFile(pub)
			Expect(err).ToNot(HaveOccurred())
			Expect(prov.shouldCacheClientCert(pd, "rip.mcollective")).To(BeFalse())
		})

		It("Should cache valid certs", func() {
			pub, err := prov.publicCertPath()
			Expect(err).ToNot(HaveOccurred())

			pd, err := ioutil.ReadFile(pub)
			Expect(err).ToNot(HaveOccurred())
			Expect(prov.shouldCacheClientCert(pd, "rip.mcollective")).To(BeTrue())
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

	Describe("CachePublicData", func() {
		It("Should not write untrusted files to disk", func() {
			prov.cache = os.TempDir()
			pd, err := ioutil.ReadFile(filepath.Join("testdata", "foreign.pem"))
			Expect(err).ToNot(HaveOccurred())
			err = prov.CachePublicData(pd, "foreign")
			Expect(err).To(MatchError("certificate 'foreign' did not pass validation"))

			cpath, err := prov.cachePath("foreign")
			Expect(err).ToNot(HaveOccurred())

			_, err = os.Stat(cpath)
			Expect(err).To(HaveOccurred())
		})

		It("Should handle missing directories", func() {
			prov.cache = filepath.Join("testdata", "nonexisting")

			path, err := prov.publicCertPath()
			Expect(err).ToNot(HaveOccurred())

			pd, err := ioutil.ReadFile(path)
			Expect(err).ToNot(HaveOccurred())

			err = prov.CachePublicData(pd, "rip.mcollective")

			if runtime.GOOS == "windows" {
				Expect(err).To(MatchError(fmt.Sprintf("could not cache client public certificate: open %s: The system cannot find the path specified.", filepath.Join("testdata", "nonexisting", "rip.mcollective.pem"))))
			} else {
				Expect(err).To(MatchError(fmt.Sprintf("could not cache client public certificate: open %s: no such file or directory", filepath.Join("testdata", "nonexisting", "rip.mcollective.pem"))))
			}
		})

		It("Should write trusted files to disk", func() {
			prov.cache = os.TempDir()
			pub, err := prov.publicCertPath()
			Expect(err).ToNot(HaveOccurred())

			pd, err := ioutil.ReadFile(pub)
			Expect(err).ToNot(HaveOccurred())

			err = prov.CachePublicData(pd, "rip.mcollective")
			Expect(err).ToNot(HaveOccurred())

			cpath, err := prov.cachePath("rip.mcollective")
			Expect(err).ToNot(HaveOccurred())

			_, err = os.Stat(cpath)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("CachedPublicData", func() {
		It("Should read the correct file", func() {
			prov.cache = os.TempDir()
			settings.EXPECT().Getuid().Return(500).AnyTimes()

			pub, err := prov.publicCertPath()
			Expect(err).ToNot(HaveOccurred())

			pd, err := ioutil.ReadFile(pub)
			Expect(err).ToNot(HaveOccurred())

			err = prov.CachePublicData(pd, "rip.mcollective")
			Expect(err).ToNot(HaveOccurred())

			dat, err := prov.CachedPublicData("rip.mcollective")
			Expect(err).ToNot(HaveOccurred())
			Expect(dat).To(Equal(pd))
		})
	})

	Describe("privilegedCerts", func() {
		It("Should find the right certs", func() {
			cfg.Choria.PrivilegedUsers = []string{"\\.privileged.mcollective$", "\\.super.mcollective$"}
			cfg.Choria.CertnameWhitelist = []string{"\\.mcollective$"}

			expected := []string{
				"1.privileged.mcollective",
				"1.super.mcollective",
				"2.privileged.mcollective",
				"2.super.mcollective",
			}

			Expect(prov.privilegedCerts()).To(Equal(expected))
		})
	})

	Describe("writeCSR", func() {
		It("should not write over existing CSRs", func() {
			prov.conf.OverrideCertname = "na.mcollective"

			kpath, err := prov.privateKeyPath()
			Expect(err).ToNot(HaveOccurred())
			csrpath, err := prov.csrPath()
			Expect(err).ToNot(HaveOccurred())

			defer os.Remove(kpath)
			defer os.Remove(csrpath)

			key, err := prov.writePrivateKey()
			Expect(err).ToNot(HaveOccurred())

			prov.conf.OverrideCertname = "rip.mcollective"
			err = prov.writeCSR(key, "rip.mcollective", "choria.io")

			Expect(err).To(MatchError("a certificate request already exist for rip.mcollective"))
		})

		It("Should create a valid CSR", func() {
			prov.conf.OverrideCertname = "na.mcollective"

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
			Expect(prov.csrExists()).To(BeTrue())
		})

		It("Should detect absent keys", func() {
			prov.conf.OverrideCertname = "na.mcollective"
			Expect(prov.csrExists()).To(BeFalse())
		})
	})

	Describe("privateKeyExists", func() {
		It("Should detect existing keys", func() {
			prov.conf.OverrideCertname = "rip.mcollective"
			Expect(prov.privateKeyExists()).To(BeTrue())
		})

		It("Should detect absent keys", func() {
			prov.conf.OverrideCertname = "na.mcollective"
			Expect(prov.privateKeyExists()).To(BeFalse())
		})
	})

	Describe("puppetCA", func() {
		It("Should use supplied config", func() {
			cfg.rawOpts["plugin.choria.puppetca_host"] = "set"
			cfg.rawOpts["plugin.choria.puppetca_port"] = "set"

			s := prov.puppetCA()
			Expect(s.Host).To(Equal("puppet"))
			Expect(s.Port).To(Equal(8140))
			Expect(s.Scheme).To(Equal("https"))
		})

		It("Should return defaults when SRV fails", func() {
			settings.EXPECT().QuerySrvRecords([]string{"_x-puppet-ca._tcp", "_x-puppet._tcp"}).Return([]Server{}, errors.New("simulated error"))

			s := prov.puppetCA()
			Expect(s.Host).To(Equal("puppet"))
			Expect(s.Port).To(Equal(8140))
			Expect(s.Scheme).To(Equal("https"))
		})

		It("Should use SRV records", func() {
			ans := []Server{
				Server{"p1", 8080, "http"},
				Server{"p2", 8080, "http"},
			}

			settings.EXPECT().QuerySrvRecords([]string{"_x-puppet-ca._tcp", "_x-puppet._tcp"}).Return(ans, errors.New("simulated error"))

			s := prov.puppetCA()
			Expect(s.Host).To(Equal("p1"))
			Expect(s.Port).To(Equal(8080))
			Expect(s.Scheme).To(Equal("http"))

		})
	})
})
