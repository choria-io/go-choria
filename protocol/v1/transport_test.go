package v1

import (
	"github.com/choria-io/go-protocol/protocol"
	gomock "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"
)

var _ = Describe("TransportMessage", func() {
	var mockctl *gomock.Controller
	var security *MockSecurityProvider

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())
		security = NewMockSecurityProvider(mockctl)
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	It("Should support reply data", func() {
		security.EXPECT().ChecksumString(gomock.Any()).Return([]byte("stub checksum")).AnyTimes()

		request, _ := NewRequest("test", "go.tests", "rip.mcollective", 120, "a2f0ca717c694f2086cfa81b6c494648", "mcollective")
		request.SetMessage(`{"message":1}`)
		reply, _ := NewReply(request, "testing")
		sreply, _ := NewSecureReply(reply, security)
		treply, _ := NewTransportMessage("rip.mcollective")
		treply.SetReplyData(sreply)

		sj, err := sreply.JSON()
		Expect(err).ToNot(HaveOccurred())

		j, err := treply.JSON()
		Expect(err).ToNot(HaveOccurred())

		Expect(gjson.Get(j, "protocol").String()).To(Equal(protocol.TransportV1))
		Expect(gjson.Get(j, "headers.mc_sender").String()).To(Equal("rip.mcollective"))

		d, err := treply.Message()
		Expect(err).ToNot(HaveOccurred())

		Expect(d).To(Equal(sj))
	})

	It("Should support request data", func() {
		security.EXPECT().PublicCertTXT().Return([]byte("stub cert"), nil).AnyTimes()
		security.EXPECT().SignString(gomock.Any()).Return([]byte("stub sig"), nil).AnyTimes()

		request, _ := NewRequest("test", "go.tests", "rip.mcollective", 120, "a2f0ca717c694f2086cfa81b6c494648", "mcollective")
		request.SetMessage(`{"message":1}`)
		srequest, _ := NewSecureRequest(request, security)
		trequest, _ := NewTransportMessage("rip.mcollective")
		trequest.SetRequestData(srequest)

		sj, _ := srequest.JSON()
		j, _ := trequest.JSON()

		Expect(gjson.Get(j, "protocol").String()).To(Equal(protocol.TransportV1))
		Expect(gjson.Get(j, "headers.mc_sender").String()).To(Equal("rip.mcollective"))

		d, err := trequest.Message()
		Expect(err).ToNot(HaveOccurred())

		Expect(d).To(Equal(sj))
	})

	It("Should support creation from JSON data", func() {
		protocol.ClientStrictValidation = true
		defer func() { protocol.ClientStrictValidation = false }()

		security.EXPECT().PublicCertTXT().Return([]byte("stub cert"), nil).AnyTimes()
		security.EXPECT().SignString(gomock.Any()).Return([]byte("stub sig"), nil).AnyTimes()

		request, _ := NewRequest("test", "go.tests", "rip.mcollective", 120, "a2f0ca717c694f2086cfa81b6c494648", "mcollective")
		srequest, _ := NewSecureRequest(request, security)
		trequest, _ := NewTransportMessage("rip.mcollective")
		trequest.SetRequestData(srequest)

		j, _ := trequest.JSON()

		_, err := NewTransportFromJSON(j)
		Expect(err).ToNot(HaveOccurred())

		_, err = NewTransportFromJSON(`{"protocol": 1}`)
		Expect(err).To(MatchError("Supplied JSON document is not a valid Transport message: (root): data is required, (root): headers is required, protocol: Invalid type. Expected: string, given: integer"))
	})
})
