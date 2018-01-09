package v1

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/choria-io/go-protocol/protocol"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

var _ = Describe("SecureRequest", func() {
	BeforeSuite(func() {
		logrus.SetLevel(logrus.FatalLevel)
	})

	It("Should create a valid SecureRequest for", func() {
		r, _ := NewRequest("test", "go.tests", "rip.mcollective", 120, "a2f0ca717c694f2086cfa81b6c494648", "mcollective")
		r.SetMessage(`{"test":1}`)
		rj, err := r.JSON()
		Expect(err).ToNot(HaveOccurred())

		sr, err := NewSecureRequest(r, "testdata/ssl/certs/rip.mcollective.pem", "testdata/ssl/private_keys/rip.mcollective.pem")
		Expect(err).ToNot(HaveOccurred())

		sj, err := sr.JSON()
		Expect(err).ToNot(HaveOccurred())

		pubf, _ := readFile("testdata/ssl/certs/rip.mcollective.pem")
		privf, _ := readFile("testdata/ssl/private_keys/rip.mcollective.pem")

		// what signString() is doing lets just verify it
		pem, _ := pem.Decode(privf)
		pk, err := x509.ParsePKCS1PrivateKey(pem.Bytes)
		Expect(err).ToNot(HaveOccurred())
		rng := rand.Reader
		hashed := sha256.Sum256([]byte(rj))
		signature, _ := rsa.SignPKCS1v15(rng, pk, crypto.SHA256, hashed[:])

		Expect(gjson.Get(sj, "protocol").String()).To(Equal(protocol.SecureRequestV1))
		Expect(gjson.Get(sj, "message").String()).To(Equal(rj))
		Expect(gjson.Get(sj, "pubcert").String()).To(Equal(string(pubf)))
		Expect(gjson.Get(sj, "signature").String()).To(Equal(base64.StdEncoding.EncodeToString(signature)))
	})

	PMeasure("SecureRequest creation time", func(b Benchmarker) {
		r, _ := NewRequest("test", "go.tests", "rip.mcollective", 120, "a2f0ca717c694f2086cfa81b6c494648", "mcollective")
		r.SetMessage(`{"test":1}`)

		runtime := b.Time("runtime", func() {
			NewSecureRequest(r, "testdata/ssl/certs/rip.mcollective.pem", "testdata/ssl/private_keys/rip.mcollective.pem")
		})

		Expect(runtime.Seconds()).Should(BeNumerically("<", 0.5))
	}, 10)

	var _ = Describe("privilegedCerts", func() {
		It("Should find all priv certs", func() {
			sr := secureRequest{
				cachePath:       "testdata/choria_security/public_certs",
				privilegedRegex: []string{"\\.privileged.mcollective$", "\\.super.mcollective$"},
				whilelistRegex:  []string{"\\.mcollective$"},
			}

			expected := []string{
				"testdata/choria_security/public_certs/1.privileged.mcollective.pem",
				"testdata/choria_security/public_certs/1.super.mcollective.pem",
				"testdata/choria_security/public_certs/2.privileged.mcollective.pem",
				"testdata/choria_security/public_certs/2.super.mcollective.pem",
			}

			Expect(sr.privilegedCerts()).To(Equal(expected))
		})
	})

	var _ = Describe("requestCallerCertname", func() {
		It("Should parse names correctly", func() {
			sr := secureRequest{}
			name, err := sr.requestCallerCertname("choria=1.privileged.mcollective")
			Expect(err).To(Not(HaveOccurred()))
			Expect(name).To(Equal("1.privileged.mcollective"))

			name, err = sr.requestCallerCertname("fail")
			Expect(err).To(HaveOccurred())
		})
	})

	var _ = Describe("verifyCert", func() {
		It("Should verify against the given ca", func() {
			sr := secureRequest{
				caPath: "testdata/choria_security/public_certs/ca.pem",
			}

			cert, err := ioutil.ReadFile("testdata/choria_security/public_certs/1.mcollective.pem")
			Expect(err).To(Not(HaveOccurred()))

			Expect(sr.verifyCert(cert, "")).To(BeTrue())
			Expect(sr.verifyCert(cert, "1.mcollective")).To(BeTrue())
			Expect(sr.verifyCert(cert, "x.y.z")).To(BeFalse())
		})

		It("Should fail for the wrong CA", func() {
			sr := secureRequest{
				caPath: "testdata/ssl/certs/ca.pem",
			}

			cert, err := ioutil.ReadFile("testdata/choria_security/public_certs/1.mcollective.pem")
			Expect(err).To(Not(HaveOccurred()))

			Expect(sr.verifyCert(cert, "")).To(BeFalse())
		})
	})

	var _ = Describe("shouldCacheClientCert", func() {
		var sr secureRequest

		BeforeEach(func() {
			sr = secureRequest{
				caPath:          "testdata/choria_security/public_certs/ca.pem",
				privilegedRegex: []string{"\\.privileged.mcollective$", "\\.super.mcollective$"},
				whilelistRegex:  []string{"\\.mcollective$"},
			}
		})

		It("Should not cache unverifiable certs", func() {
			cert, err := ioutil.ReadFile("testdata/ssl/certs/rip.mcollective.pem")
			Expect(err).ToNot(HaveOccurred())

			sr.PublicCertificate = string(cert)

			Expect(sr.shouldCacheClientCert("")).To(BeFalse())
		})

		It("Should cache privileged certs", func() {
			cert, err := ioutil.ReadFile("testdata/choria_security/public_certs/1.privileged.mcollective.pem")
			Expect(err).ToNot(HaveOccurred())

			sr.PublicCertificate = string(cert)

			Expect(sr.shouldCacheClientCert("1.privileged.mcollective")).To(BeTrue())
		})

		It("Should cache certs matching the whitelist", func() {
			cert, err := ioutil.ReadFile("testdata/choria_security/public_certs/1.mcollective.pem")
			Expect(err).ToNot(HaveOccurred())

			sr.PublicCertificate = string(cert)

			Expect(sr.shouldCacheClientCert("1.mcollective")).To(BeTrue())
		})

		It("Should not cache certs that does not match the whitelist", func() {
			cert, err := ioutil.ReadFile("testdata/choria_security/public_certs/other.pem")
			Expect(err).ToNot(HaveOccurred())

			sr.PublicCertificate = string(cert)

			Expect(sr.shouldCacheClientCert("other")).To(BeFalse())
			Expect(sr.shouldCacheClientCert("1.mcollective")).To(BeFalse())
			Expect(sr.shouldCacheClientCert("1.privilged.mcollective")).To(BeFalse())
		})
	})

	var _ = Describe("cacheClientCert", func() {
		var (
			dir, rj string
			err     error
		)
		BeforeEach(func() {
			dir, err = ioutil.TempDir("", "example")
			Expect(err).ToNot(HaveOccurred())

			r, _ := NewRequest("test", "go.tests", "choria=1.mcollective", 120, "a2f0ca717c694f2086cfa81b6c494648", "mcollective")
			r.SetMessage(`{"test":1}`)
			rj, err = r.JSON()
			Expect(err).ToNot(HaveOccurred())

		})

		AfterEach(func() {
			os.RemoveAll(dir)
		})

		It("Should cache the certificate in the right location and name", func() {
			cert, err := ioutil.ReadFile("testdata/choria_security/public_certs/1.mcollective.pem")
			Expect(err).ToNot(HaveOccurred())

			sr := secureRequest{
				Protocol:          protocol.SecureRequestV1,
				MessageBody:       rj,
				cachePath:         dir,
				caPath:            "testdata/choria_security/public_certs/ca.pem",
				privilegedRegex:   []string{"\\.privileged.mcollective$", "\\.super.mcollective$"},
				whilelistRegex:    []string{"\\.mcollective$"},
				PublicCertificate: string(cert),
			}

			file, err := sr.cacheClientCert()
			Expect(err).ToNot(HaveOccurred())

			Expect(file).To(Equal(filepath.Join(dir, "1.mcollective.pem")))

			cached, err := ioutil.ReadFile(file)
			Expect(err).ToNot(HaveOccurred())

			Expect(cached).To(Equal(cert))
		})

		It("Should not cache invalid certificates", func() {
			cert, err := ioutil.ReadFile("testdata/ssl/certs/rip.mcollective.pem")
			Expect(err).ToNot(HaveOccurred())

			sr := secureRequest{
				Protocol:          protocol.SecureRequestV1,
				MessageBody:       rj,
				cachePath:         dir,
				caPath:            "testdata/choria_security/public_certs/ca.pem",
				privilegedRegex:   []string{"\\.privileged.mcollective$", "\\.super.mcollective$"},
				whilelistRegex:    []string{"\\.mcollective$"},
				PublicCertificate: string(cert),
			}

			file, err := sr.cacheClientCert()
			Expect(err).To(HaveOccurred())
			Expect(file).To(Equal(""))
		})
	})
})
