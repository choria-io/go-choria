package v1

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"

	"github.com/choria-io/go-choria/protocol"
	"github.com/tidwall/gjson"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SecureReply", func() {
	It("Should create valid replies", func() {
		request, _ := NewRequest("test", "go.tests", "rip.mcollective", 120, "a2f0ca717c694f2086cfa81b6c494648", "mcollective")
		request.SetMessage(`{"test":1}`)

		reply, err := NewReply(request, "testing")
		Expect(err).ToNot(HaveOccurred())

		rj, err := reply.JSON()
		Expect(err).ToNot(HaveOccurred())

		sha := sha256.Sum256([]byte(rj))

		sreply, _ := NewSecureReply(reply)
		sj, err := sreply.JSON()
		Expect(err).ToNot(HaveOccurred())

		Expect(gjson.Get(sj, "protocol").String()).To(Equal(protocol.SecureReplyV1))
		Expect(gjson.Get(sj, "message").String()).To(Equal(rj))
		Expect(gjson.Get(sj, "hash").String()).To(Equal(base64.StdEncoding.EncodeToString(sha[:])))
		Expect(sreply.Valid()).To(BeTrue())
	})
})

var _ = Describe("SecureRequest", func() {
	It("Should create a valid SecureRequest", func() {
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

	Measure("SecureRequest creation time", func(b Benchmarker) {
		r, _ := NewRequest("test", "go.tests", "rip.mcollective", 120, "a2f0ca717c694f2086cfa81b6c494648", "mcollective")
		r.SetMessage(`{"test":1}`)

		runtime := b.Time("runtime", func() {
			NewSecureRequest(r, "testdata/ssl/certs/rip.mcollective.pem", "testdata/ssl/private_keys/rip.mcollective.pem")
		})

		Expect(runtime.Seconds()).Should(BeNumerically("<", 0.5))
	}, 10)
})
