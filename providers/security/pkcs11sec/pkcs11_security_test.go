// Copyright (c) 2020-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package pkcs11sec

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/choria-io/go-choria/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestFileSecurity(t *testing.T) {
	RegisterFailHandler(Fail)
	if runPkcs11 := os.Getenv("RUN_PKCS11_TESTS"); runPkcs11 == "1" {
		RunSpecs(t, "Security/Pkcs11Security")
	}
}

var _ = Describe("Pkcs11Security", func() {
	var err error
	var prov *Pkcs11Security
	var l *logrus.Logger
	var lib string
	var pin = "1234"
	var c *config.Config
	var testSlot = 374292918

	BeforeEach(func() {
		lib = "/usr/lib/softhsm/libsofthsm2.so"
		if envLib := os.Getenv("SOFTHSM_LIB"); envLib != "" {
			lib = envLib
		}

		wd, _ := os.Getwd()
		err = os.Setenv("SOFTHSM2_CONF", wd+"/softhsm2.conf")
		Expect(err).ToNot(HaveOccurred())

		l = logrus.New()
		l.Out = GinkgoWriter

		c, err = config.NewDefaultConfig()
		Expect(err).ToNot(HaveOccurred())

		c.Choria.FileSecurityCA = filepath.Join("..", "testdata", "good", "certs", "ca.pem")
		c.Choria.PKCS11Slot = testSlot
		c.Choria.PKCS11DriverFile = lib

		prov, err = New(WithChoriaConfig(c), WithLog(l.WithFields(logrus.Fields{})), WithPin(pin))
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("WithChoriaConfig", func() {
		It("Should copy all the relevant settings", func() {
			c, err := config.NewDefaultConfig()
			Expect(err).ToNot(HaveOccurred())

			c.Choria.FileSecurityCA = filepath.Join("..", "testdata", "good", "certs", "ca.pem")
			c.Choria.PKCS11Slot = testSlot
			c.Choria.PKCS11DriverFile = lib
			c.DisableTLSVerify = true

			prov, err := New(WithChoriaConfig(c), WithLog(l.WithFields(logrus.Fields{})), WithPin("1234"))
			Expect(err).ToNot(HaveOccurred())

			Expect(prov.conf.AllowList).To(Equal([]string{"\\.mcollective$", "\\.choria$"}))
			Expect(prov.conf.PrivilegedUsers).To(Equal([]string{"\\.privileged.mcollective$", "\\.privileged.choria$"}))
			Expect(prov.conf.CAFile).To(Equal("../testdata/good/certs/ca.pem"))
			Expect(prov.conf.DisableTLSVerify).To(BeTrue())
		})
	})

	Describe("Validate", func() {
		It("Should return true if provider was successfully initialized", func() {
			errs, ok := prov.Validate()
			Expect(errs).To(HaveLen(0))
			Expect(ok).To(BeTrue())
		})
		It("Should handle missing files", func() {
			c, err := config.NewDefaultConfig()
			Expect(err).ToNot(HaveOccurred())

			c.Choria.FileSecurityCA = "stub/ca.pem"
			c.Choria.PKCS11Slot = testSlot
			c.Choria.PKCS11DriverFile = lib

			prov, err = New(WithChoriaConfig(c), WithLog(l.WithFields(logrus.Fields{})), WithPin(pin))
			Expect(err).ToNot(HaveOccurred())

			errs, ok := prov.Validate()
			Expect(ok).To(BeFalse())
			Expect(errs).To(HaveLen(2))
			Expect(errs[0]).To(Equal(fmt.Sprintf("stat %s: no such file or directory", prov.conf.CAFile)))
		})
	})

	Describe("Identity", func() {
		It("Should return the identity", func() {
			Expect(prov.Identity()).To(Equal("joeuser"))
		})
	})

	Describe("CallerName", func() {
		It("Should return the right caller name", func() {
			Expect(prov.CallerName()).To(Equal("choria=joeuser"))
		})
	})

	Describe("CallerIdentity", func() {
		It("Should return the right caller ident, no matter the input", func() {
			Expect(prov.CallerIdentity("choria=test.choria")).To(Equal("test.choria"))
			Expect(prov.CallerIdentity("foo=test1.choria")).To(Equal("test1.choria"))
		})
	})

	Describe("SignBytes", func() {
		It("Should produce the right signature", func() {
			sig, err := prov.SignBytes([]byte("too many secrets"))
			Expect(err).ToNot(HaveOccurred())
			Expect(base64.StdEncoding.EncodeToString(sig)).To(Equal("PQlGnXt8jQ9N2WbghvKhH4qNTJcmTpbfspkT+9aSabivRbMGNIlMwDGMg8PQEC5AMF9eoxdaXuR/t2rbgUfqQrB3oI2YMD2clUtdVI1MIJ81ww90o0KHZa3C0N/OlshJVCDg1mUiget7rdfE5K3HARKbPZZbQFe/q5yPnjA7FGHEb1K+qnPyLGKD8WKIDTjHza16O6QWAcbyAWk2CP9ziLH5flVGMP0zMkdXQPiFfzexUG6iTIi64zVJ2k6E3k1JOGzRLeQfvUDNEQnmekH4w0iK0+uTZzBsQPr3jbd8xraTInv+v1CzrpBwoIP36Qlr296vxKngaqDSN2K3uSyKWg=="))
		})
	})

	Describe("VerifyByteSignature", func() {
		It("Should validate correctly", func() {
			sig, err := base64.StdEncoding.DecodeString("PQlGnXt8jQ9N2WbghvKhH4qNTJcmTpbfspkT+9aSabivRbMGNIlMwDGMg8PQEC5AMF9eoxdaXuR/t2rbgUfqQrB3oI2YMD2clUtdVI1MIJ81ww90o0KHZa3C0N/OlshJVCDg1mUiget7rdfE5K3HARKbPZZbQFe/q5yPnjA7FGHEb1K+qnPyLGKD8WKIDTjHza16O6QWAcbyAWk2CP9ziLH5flVGMP0zMkdXQPiFfzexUG6iTIi64zVJ2k6E3k1JOGzRLeQfvUDNEQnmekH4w0iK0+uTZzBsQPr3jbd8xraTInv+v1CzrpBwoIP36Qlr296vxKngaqDSN2K3uSyKWg==")
			Expect(err).ToNot(HaveOccurred())

			valid, _ := prov.VerifyByteSignature([]byte("too many secrets"), sig, nil)
			Expect(valid).To(BeTrue())
		})

		It("Should fail for invalid sigs", func() {
			valid, _ := prov.VerifyByteSignature([]byte("too many secrets"), []byte("meh"), nil)
			Expect(valid).To(BeFalse())
		})

		It("Should support cached certificates", func() {
			prov, err = New(WithChoriaConfig(c), WithLog(l.WithFields(logrus.Fields{})), WithPin(pin))
			Expect(err).ToNot(HaveOccurred())

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
			c, err := config.NewDefaultConfig()
			Expect(err).ToNot(HaveOccurred())

			c.Choria.FileSecurityCA = filepath.Join("test_data", "ssl_dir", "certs", "ca.pem")
			c.Choria.PKCS11Slot = testSlot
			c.Choria.PKCS11DriverFile = lib

			prov, err = New(WithChoriaConfig(c), WithLog(l.WithFields(logrus.Fields{})), WithPin(pin))

			Expect(err).ToNot(HaveOccurred())
			tlsC, err := prov.TLSConfig()
			Expect(err).ToNot(HaveOccurred())

			Expect(tlsC.InsecureSkipVerify).To(BeFalse())
			Expect(err).ToNot(HaveOccurred())

			cert := prov.cert

			Expect(tlsC.Certificates).To(HaveLen(1))
			Expect(tlsC.Certificates[0].Certificate).To(Equal(cert.Certificate))
		})

		It("Should support disabling tls verify", func() {
			c, err := config.NewDefaultConfig()
			Expect(err).ToNot(HaveOccurred())

			c.Choria.FileSecurityCA = filepath.Join("test_data", "ssl_dir", "certs", "ca.pem")
			c.DisableTLSVerify = true
			c.Choria.PKCS11Slot = testSlot
			c.Choria.PKCS11DriverFile = lib

			prov, err = New(WithChoriaConfig(c), WithLog(l.WithFields(logrus.Fields{})), WithPin(pin))

			Expect(err).ToNot(HaveOccurred())

			tlsC, err := prov.TLSConfig()
			Expect(err).ToNot(HaveOccurred())

			Expect(tlsC.InsecureSkipVerify).To(BeTrue())

		})
	})

	Describe("VerifyCertificate", func() {
		var pem []byte
		var inter *config.Config

		BeforeEach(func() {
			inter, err = config.NewDefaultConfig()
			Expect(err).ToNot(HaveOccurred())

			inter.Choria.FileSecurityCA = filepath.Join("..", "testdata", "intermediate", "certs", "ca_chain_ca.pem")
			inter.Choria.PKCS11Slot = testSlot
			inter.Choria.PKCS11DriverFile = lib

			pem, err = prov.PublicCertBytes()
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should fail for foreign certs", func() {
			pem, err = os.ReadFile(filepath.Join("..", "testdata", "foreign.pem"))
			Expect(err).ToNot(HaveOccurred())
			err := prov.VerifyCertificate(pem, "rip.mcollective")
			Expect(err).To(MatchError("x509: certificate signed by unknown authority"))

		})

		It("Should fail for invalid names", func() {
			prov, err = New(WithChoriaConfig(c), WithLog(l.WithFields(logrus.Fields{})), WithPin(pin))
			Expect(err).ToNot(HaveOccurred())

			err := prov.VerifyCertificate(pem, "bob")
			Expect(err).To(MatchError("x509: certificate is valid for joeuser, not bob"))
		})

		It("Should accept valid certs", func() {
			c, err := config.NewDefaultConfig()
			Expect(err).ToNot(HaveOccurred())

			c.Choria.FileSecurityCA = filepath.Join("test_data", "ssl_dir", "certs", "ca.pem")
			c.Choria.PKCS11Slot = testSlot
			c.Choria.PKCS11DriverFile = lib

			prov, err = New(WithChoriaConfig(c), WithLog(l.WithFields(logrus.Fields{})), WithPin(pin))
			Expect(err).ToNot(HaveOccurred())

			err = prov.VerifyCertificate(pem, "joeuser")
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should work with client provided intermediate chains", func() {
			c, err := config.NewDefaultConfig()
			Expect(err).ToNot(HaveOccurred())

			c.Choria.FileSecurityCA = filepath.Join("..", "testdata", "intermediate", "certs", "ca.pem")
			c.Choria.SSLDir = filepath.Join("..", "testdata", "intermediate")
			c.Choria.PKCS11Slot = testSlot
			c.Choria.PKCS11DriverFile = lib

			prov, err := New(WithChoriaConfig(c), WithLog(l.WithFields(logrus.Fields{})), WithPin("1234"))
			Expect(err).ToNot(HaveOccurred())

			pem, err = os.ReadFile(filepath.Join("..", "testdata", "intermediate", "certs", "rip.mcollective.pem"))
			Expect(err).ToNot(HaveOccurred())

			err = prov.VerifyCertificate(pem, "rip.mcollective")
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should work with server side ca intermediate chains", func() {
			prov, err := New(WithChoriaConfig(inter), WithLog(l.WithFields(logrus.Fields{})), WithPin("1234"))
			Expect(err).ToNot(HaveOccurred())

			pem, err = os.ReadFile(filepath.Join("..", "testdata", "intermediate", "certs", "ca_chain_rip.mcollective.pem"))
			Expect(err).ToNot(HaveOccurred())

			err = prov.VerifyCertificate(pem, "rip.mcollective")
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should work with email addresses", func() {
			prov, err := New(WithChoriaConfig(inter), WithLog(l.WithFields(logrus.Fields{})), WithPin(pin))
			Expect(err).ToNot(HaveOccurred())

			pem, err = os.ReadFile(filepath.Join("..", "testdata", "intermediate", "certs", "email-chain-rip.mcollective.pem"))
			Expect(err).ToNot(HaveOccurred())

			err = prov.VerifyCertificate(pem, "email:test@choria-io.com")
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should not work with wrong addresses", func() {
			prov, err := New(WithChoriaConfig(inter), WithLog(l.WithFields(logrus.Fields{})), WithPin(pin))
			Expect(err).ToNot(HaveOccurred())

			pem, err = os.ReadFile(filepath.Join("..", "testdata", "intermediate", "certs", "email-chain-rip.mcollective.pem"))
			Expect(err).ToNot(HaveOccurred())

			err = prov.VerifyCertificate(pem, "email:bad@choria-io.com")
			Expect(err).To(HaveOccurred())
		})
	})
})
