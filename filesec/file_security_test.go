package filesec

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-security"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestFileSecurity(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Security/File")
}

func setSSL(c *Config, parent string, id string) {
	c.Certificate = filepath.Join(parent, "certs", fmt.Sprintf("%s.pem", id))
	c.CA = filepath.Join(parent, "certs", "ca.pem")
	c.Cache = filepath.Join(parent, "choria_security", "public_certs")
	c.Key = filepath.Join(parent, "private_keys", fmt.Sprintf("%s.pem", id))
	c.AllowList = []string{"\\.mcollective$"}
	c.PrivilegedUsers = []string{"\\.privileged.mcollective$"}
	c.DisableTLSVerify = false
	c.Identity = id

	useFakeUID = true
	fakeUID = 500
}

var _ = Describe("FileSSL", func() {
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
		setSSL(cfg, goodStub, "rip.mcollective")

		l = logrus.New()

		l.Out = ioutil.Discard

		prov, err = New(WithConfig(cfg), WithLog(l.WithFields(logrus.Fields{})))
		Expect(err).ToNot(HaveOccurred())
	})

	It("Should impliment the provider interface", func() {
		f := func(p security.Provider) {}
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
			prov, err := New(WithChoriaConfig(c), WithLog(l.WithFields(logrus.Fields{})))
			Expect(err).ToNot(HaveOccurred())

			Expect(prov.conf.Identity).To(Equal("override.choria"))
		})

		It("Should support MCOLLECTIVE_CERTNAME", func() {
			os.Setenv("MCOLLECTIVE_CERTNAME", "bob.mcollective")
			c, err := config.NewDefaultConfig()
			Expect(err).ToNot(HaveOccurred())
			prov, err := New(WithChoriaConfig(c), WithLog(l.WithFields(logrus.Fields{})))
			Expect(err).ToNot(HaveOccurred())

			Expect(prov.conf.Identity).To(Equal("bob.mcollective"))
		})

		It("Should copy all the relevant settings", func() {
			c, err := config.NewDefaultConfig()
			Expect(err).ToNot(HaveOccurred())

			fakeUID = 0
			c.Choria.FileSecurityCA = "stub/ca.pem"
			c.Choria.FileSecurityCache = "stub/cache"
			c.Choria.FileSecurityCertificate = "stub/cert.pem"
			c.Choria.FileSecurityKey = "stub/key.pem"
			c.DisableTLSVerify = true
			c.Identity = "test.identity"

			prov, err := New(WithChoriaConfig(c), WithLog(l.WithFields(logrus.Fields{})))
			Expect(err).ToNot(HaveOccurred())

			Expect(prov.conf.AllowList).To(Equal([]string{"\\.mcollective$", "\\.choria$"}))
			Expect(prov.conf.PrivilegedUsers).To(Equal([]string{"\\.privileged.mcollective$", "\\.privileged.choria$"}))
			Expect(prov.conf.CA).To(Equal("stub/ca.pem"))
			Expect(prov.conf.Cache).To(Equal("stub/cache"))
			Expect(prov.conf.Certificate).To(Equal("stub/cert.pem"))
			Expect(prov.conf.Key).To(Equal("stub/key.pem"))
			Expect(prov.conf.DisableTLSVerify).To(BeTrue())
			Expect(prov.conf.Identity).To(Equal("test.identity"))
		})

		It("Should support override certname", func() {
			c, err := config.NewDefaultConfig()
			Expect(err).ToNot(HaveOccurred())

			c.Choria.FileSecurityCA = "stub/ca.pem"
			c.Choria.FileSecurityCache = "stub/cache"
			c.Choria.FileSecurityCertificate = "stub/cert.pem"
			c.Choria.FileSecurityKey = "stub/key.pem"
			c.DisableTLSVerify = true
			c.Identity = "test.identity"
			c.OverrideCertname = "bob.identity"

			prov, err := New(WithChoriaConfig(c), WithLog(l.WithFields(logrus.Fields{})))
			Expect(err).ToNot(HaveOccurred())

			Expect(prov.conf.Identity).To(Equal("bob.identity"))
		})

		It("Should support root and windows", func() {
			c, err := config.NewDefaultConfig()
			Expect(err).ToNot(HaveOccurred())

			c.Choria.FileSecurityCA = "stub/ca.pem"
			c.Choria.FileSecurityCache = "stub/cache"
			c.Choria.FileSecurityCertificate = "stub/cert.pem"
			c.Choria.FileSecurityKey = "stub/key.pem"
			c.DisableTLSVerify = true
			c.Identity = "test.identity"

			useFakeOS = true
			defer func() { useFakeOS = false }()
			fakeOS = "windows"
			Expect(runtimeOs()).To(Equal("windows"))

			prov, err = New(WithChoriaConfig(c), WithLog(l.WithFields(logrus.Fields{})))
			Expect(err).ToNot(HaveOccurred())

			Expect(prov.conf.Identity).To(Equal("test.identity"))
		})
	})

	Describe("Validate", func() {
		It("Should handle missing files", func() {
			setSSL(cfg, nonexistingStub, "test.mcollective")
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
			setSSL(cfg, goodStub, "rip.mcollective")

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
			cfg.Identity = "2.mcollective"

			sig, err := base64.StdEncoding.DecodeString("Zq1F2bdXOAvB5Ca+iYCZ/BLYz2ZzbQP/V8kwQY0E3cuDrBDArX7UhUnBakzN+Msr7UyF+EkYmzvIi4KHnFBrgi7otM8Q5YMh5IT+IPaoHj3Rj/jorqD4g8ltZINqCUBWDN4wvSG98SxLyawV69gAK4SnP+oy7SU7zxuQiPwIMJ7lVoiQ3t+tiQAHUxeykQPw7WElLb+wPTb1k4DM3yRkijA9OeUk+3SVyl2sTCu5h/Lg0lcI372bkLDESlnhnvw7yuLD2SSncrEQrBdv/N2yEpY2fx1UKGlTrn9GH4MGA1GuzE1F87RH9P8ieeul6vI13BkBAlMk5KaGlmWpgiGri5UjCHHXMxEnXfwUcKFE+E6yVg4SbrJknkuJzNJduypMIep7YOnPHVLNIBZLuOUdJrRgBQ+Yb9mxPnEQHhOHeN0XHUcseRJEISqPkagpNx1xhOb7g3hsNyEvqibT/DZsc/2hyU2I/wG9fl26CnN9c12r1zInyCQYsU/wuIvjDtRZvTpLGJSJdgjSmTPzGmA/fKpAfOWObdsoLeorjF/pNweuc0x0JZMsBrZauldLL53wnnvllsFEmIAxs+RusoJ2UfW7WugZ7lXGISHTef6IHjukHgDBSbeGawVCnAgPbPz1dy42x04koUW3Bmz89fJ4/j+e49ijz7z3W/IercNeke4=")
			Expect(err).ToNot(HaveOccurred())

			valid := prov.VerifyByteSignature([]byte("too many secrets"), sig, "2.mcollective")
			Expect(valid).To(BeTrue())
		})
	})

	Describe("PrivilegedVerifyByteSignature", func() {
		It("Should allow a privileged cert to act as someone else", func() {
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
			pem, err = ioutil.ReadFile(pub)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should fail for foreign certs", func() {
			pem, err = ioutil.ReadFile(filepath.Join("..", "testdata", "foreign.pem"))
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
			c.Choria.FileSecurityCache = filepath.Join("..", "testdata", "intermediate", "certs")

			prov, err := New(WithChoriaConfig(c), WithLog(l.WithFields(logrus.Fields{})))
			Expect(err).ToNot(HaveOccurred())

			pem, err = ioutil.ReadFile(filepath.Join("..", "testdata", "intermediate", "certs", "rip.mcollective.pem"))
			Expect(err).ToNot(HaveOccurred())

			err = prov.VerifyCertificate(pem, "rip.mcollective")
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should work with server side ca intermediate chains", func() {
			c, err := config.NewDefaultConfig()
			Expect(err).ToNot(HaveOccurred())

			c.Choria.FileSecurityCA = filepath.Join("..", "testdata", "intermediate", "certs", "ca_chain_ca.pem")
			c.Choria.FileSecurityCache = filepath.Join("..", "testdata", "intermediate", "certs")

			prov, err := New(WithChoriaConfig(c), WithLog(l.WithFields(logrus.Fields{})))
			Expect(err).ToNot(HaveOccurred())

			pem, err = ioutil.ReadFile(filepath.Join("..", "testdata", "intermediate", "certs", "ca_chain_rip.mcollective.pem"))
			Expect(err).ToNot(HaveOccurred())

			err = prov.VerifyCertificate(pem, "rip.mcollective")
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("PublicCertPem", func() {
		It("Should return the correct pem data", func() {
			dat, err := ioutil.ReadFile(cfg.Certificate)
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
			pd, err := ioutil.ReadFile(filepath.Join("..", "testdata", "foreign.pem"))
			Expect(err).ToNot(HaveOccurred())

			should, name := prov.shouldCacheClientCert(pd, "foo")
			Expect(should).To(BeFalse())
			Expect(name).To(Equal("foo"))

			pub := prov.publicCertPath()
			pd, err = ioutil.ReadFile(pub)
			Expect(err).ToNot(HaveOccurred())

			should, name = prov.shouldCacheClientCert(pd, "rip.mcollective")
			Expect(should).To(BeTrue())
			Expect(name).To(Equal("rip.mcollective"))
		})

		It("Should cache privileged certs", func() {
			pd, err := ioutil.ReadFile(filepath.Join("..", "testdata", "good", "certs", "1.privileged.mcollective.pem"))
			Expect(err).ToNot(HaveOccurred())

			should, name := prov.shouldCacheClientCert(pd, "bob")
			Expect(should).To(BeTrue())
			Expect(name).To(Equal("1.privileged.mcollective"))
		})

		It("Should not cache certs with wrong names", func() {
			pub := prov.publicCertPath()

			pd, err := ioutil.ReadFile(pub)
			Expect(err).ToNot(HaveOccurred())

			should, name := prov.shouldCacheClientCert(pd, "bob")
			Expect(should).To(BeFalse())
			Expect(name).To(Equal("bob"))
		})

		It("Should only cache certs thats on the allowed list", func() {
			cfg.AllowList = []string{"bob"}
			pub := prov.publicCertPath()

			pd, err := ioutil.ReadFile(pub)
			Expect(err).ToNot(HaveOccurred())

			should, name := prov.shouldCacheClientCert(pd, "rip.mcollective")
			Expect(should).To(BeFalse())
			Expect(name).To(Equal("rip.mcollective"))
		})

		It("Should cache valid certs", func() {
			pub := prov.publicCertPath()

			pd, err := ioutil.ReadFile(pub)
			Expect(err).ToNot(HaveOccurred())

			should, name := prov.shouldCacheClientCert(pd, "rip.mcollective")
			Expect(should).To(BeTrue())
			Expect(name).To(Equal("rip.mcollective"))
		})
	})

	Describe("CachePublicData", func() {
		It("Should not write untrusted files to disk", func() {
			cfg.Cache = os.TempDir()
			pd, err := ioutil.ReadFile(filepath.Join("..", "testdata", "foreign.pem"))
			Expect(err).ToNot(HaveOccurred())
			err = prov.CachePublicData(pd, "foreign")
			Expect(err).To(MatchError("certificate 'foreign' did not pass validation"))

			cpath, err := prov.cachePath("foreign")
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove(cpath)

			_, err = os.Stat(cpath)
			Expect(err).To(HaveOccurred())
		})

		It("Should write trusted files to disk", func() {
			cfg.Cache = os.TempDir()
			pub := prov.publicCertPath()

			pd, err := ioutil.ReadFile(pub)
			Expect(err).ToNot(HaveOccurred())

			err = prov.CachePublicData(pd, "rip.mcollective")
			Expect(err).ToNot(HaveOccurred())

			cpath, err := prov.cachePath("rip.mcollective")
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove(cpath)

			_, err = os.Stat(cpath)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should not overwrite existing files", func() {
			cfg.Cache = os.TempDir()
			pub := prov.publicCertPath()

			pd, err := ioutil.ReadFile(pub)
			Expect(err).ToNot(HaveOccurred())

			err = prov.CachePublicData(pd, "rip.mcollective")
			Expect(err).ToNot(HaveOccurred())

			cpath, err := prov.cachePath("rip.mcollective")
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove(cpath)

			// deliberately change the file so that we can figure out if its being changed
			// I'd check time stamps but they are per second so not much use
			err = ioutil.WriteFile(cpath, []byte("too many secrets"), os.FileMode(int(0644)))
			Expect(err).ToNot(HaveOccurred())

			err = prov.CachePublicData(pd, "rip.mcollective")
			Expect(err).ToNot(HaveOccurred())

			stat, err := os.Stat(cpath)
			Expect(err).ToNot(HaveOccurred())
			Expect(stat.Size()).To(Equal(int64(16)))
		})

		It("Should support always overwrite files", func() {
			c, err := config.NewDefaultConfig()
			Expect(err).ToNot(HaveOccurred())

			identity := "rip.mcollective"

			// These certs both have the same hostname.  First we cache the first one, then we attempt to cache the second one.
			// This should result in the caching layer storing the second certificate.
			firstcert := filepath.Join("..", "testdata", "intermediate", "certs", identity+".pem")
			secondcert := filepath.Join("..", "testdata", "intermediate", "certs", "second."+identity+".pem")

			c.Choria.FileSecurityCertificate = firstcert
			c.Choria.FileSecurityCA = filepath.Join("..", "testdata", "intermediate", "certs", "ca.pem")
			c.Choria.FileSecurityCache = filepath.Join("..", "testdata", "intermediate", "certs")
			c.Choria.SecurityAlwaysOverwriteCache = true

			c.Choria.FileSecurityCache, err = ioutil.TempDir("", "cache-always")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(c.Choria.FileSecurityCache)

			prov, err := New(WithChoriaConfig(c), WithLog(l.WithFields(logrus.Fields{})))
			Expect(err).ToNot(HaveOccurred())

			fpd, err := ioutil.ReadFile(firstcert)
			Expect(err).ToNot(HaveOccurred())

			err = prov.CachePublicData(fpd, identity)
			Expect(err).ToNot(HaveOccurred())

			spd, err := ioutil.ReadFile(secondcert)
			Expect(err).ToNot(HaveOccurred())

			err = prov.CachePublicData(spd, identity)
			Expect(err).To(BeNil())

			cpd, err := prov.CachedPublicData(identity)
			Expect(err).ToNot(HaveOccurred())

			res := bytes.Compare(spd, cpd)
			Expect(res).To(BeZero())
		})

		It("Should fail cache validation if allow lists change", func() {
			cfg.Cache = os.TempDir()
			cfg.Cache = os.TempDir()
			pub := prov.publicCertPath()

			pd, err := ioutil.ReadFile(pub)
			Expect(err).ToNot(HaveOccurred())

			err = prov.CachePublicData(pd, "rip.mcollective")
			Expect(err).ToNot(HaveOccurred())

			cfg.AllowList = []string{"^bees$"}

			err = prov.CachePublicData(pd, "rip.mcollective")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("CachedPublicData", func() {
		It("Should read the correct file", func() {
			cfg.Cache = os.TempDir()

			pub := prov.publicCertPath()

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
			cfg.PrivilegedUsers = []string{"\\.privileged.mcollective$", "\\.super.mcollective$"}
			cfg.AllowList = []string{"\\.mcollective$"}

			expected := []string{
				"1.privileged.mcollective",
				"1.super.mcollective",
				"2.privileged.mcollective",
				"2.super.mcollective",
			}

			Expect(prov.privilegedCerts()).To(Equal(expected))
		})
	})

	Describe("privateKeyExists", func() {
		It("Should detect existing keys", func() {
			setSSL(cfg, goodStub, "rip.mcollective")

			Expect(prov.privateKeyExists()).To(BeTrue())
		})

		It("Should detect absent keys", func() {
			setSSL(cfg, goodStub, "na.mcollective")

			Expect(prov.privateKeyExists()).To(BeFalse())
		})
	})

})
