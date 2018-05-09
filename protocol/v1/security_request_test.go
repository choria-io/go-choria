package v1

import (
	"encoding/base64"
	"io/ioutil"

	"github.com/choria-io/go-protocol/protocol"
	gomock "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

var _ = Describe("SecureRequest", func() {
	var mockctl *gomock.Controller
	var security *MockSecurityProvider
	var pub []byte

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())
		security = NewMockSecurityProvider(mockctl)

		pub, _ = ioutil.ReadFile("testdata/ssl/certs/rip.mcollective.pem")
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	BeforeSuite(func() {
		logrus.SetLevel(logrus.FatalLevel)
	})

	It("Should create a valid SecureRequest", func() {
		security.EXPECT().PublicCertTXT().Return(pub, nil).AnyTimes()

		r, _ := NewRequest("test", "go.tests", "rip.mcollective", 120, "a2f0ca717c694f2086cfa81b6c494648", "mcollective")
		r.SetMessage(`{"test":1}`)
		rj, err := r.JSON()
		Expect(err).ToNot(HaveOccurred())

		security.EXPECT().SignString(rj).Return([]byte("stub.sig"), nil)

		sr, err := NewSecureRequest(r, security)
		Expect(err).ToNot(HaveOccurred())

		sj, err := sr.JSON()
		Expect(err).ToNot(HaveOccurred())

		Expect(gjson.Get(sj, "protocol").String()).To(Equal(protocol.SecureRequestV1))
		Expect(gjson.Get(sj, "message").String()).To(Equal(rj))
		Expect(gjson.Get(sj, "pubcert").String()).To(Equal(string(pub)))
		Expect(gjson.Get(sj, "signature").String()).To(Equal(base64.StdEncoding.EncodeToString([]byte("stub.sig"))))
	})

	PMeasure("SecureRequest creation time", func(b Benchmarker) {
		r, _ := NewRequest("test", "go.tests", "rip.mcollective", 120, "a2f0ca717c694f2086cfa81b6c494648", "mcollective")
		r.SetMessage(`{"test":1}`)

		runtime := b.Time("runtime", func() {
			NewSecureRequest(r, security)
		})

		Expect(runtime.Seconds()).Should(BeNumerically("<", 0.5))
	}, 10)

})
