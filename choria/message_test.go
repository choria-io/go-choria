package choria

import (
	"time"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-protocol/protocol"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Choria/Message", func() {
	var (
		mockctl *gomock.Controller
		request *MockRequest
		fw      *Framework
		now     time.Time
	)

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())
		request = NewMockRequest(mockctl)
		now = time.Now()

		request.EXPECT().Message().Return("hello world").AnyTimes()
		request.EXPECT().Agent().Return("test").AnyTimes()
		request.EXPECT().Collective().Return("test_collective").AnyTimes()
		request.EXPECT().RequestID().Return("stub.request.id").AnyTimes()
		request.EXPECT().TTL().Return(60).AnyTimes()
		request.EXPECT().Time().Return(now).AnyTimes()
		request.EXPECT().Filter().Return(protocol.NewFilter(), true).AnyTimes()

		cfg, err := config.NewDefaultConfig()
		Expect(err).ToNot(HaveOccurred())
		cfg.Identity = "test.identity"
		protocol.Secure = "false"
		cfg.Collectives = []string{"test_collective"}

		fw, err = NewWithConfig(cfg)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	Describe("NewMessageFromRequest", func() {
		It("Should create a new message with the correct properties", func() {
			m, err := NewMessageFromRequest(request, "reply.to", fw)
			Expect(err).ToNot(HaveOccurred())

			Expect(m.Payload).To(Equal("hello world"))
			Expect(m.replyTo).To(Equal("reply.to"))
			Expect(m.RequestID).To(Equal("stub.request.id"))
			Expect(m.TimeStamp).To(Equal(now))
			Expect(m.TTL).To(Equal(60))
			Expect(m.Filter).To(Equal(protocol.NewFilter()))
			Expect(m.SenderID).To(Equal("test.identity"))
			Expect(m.Base64Payload()).To(Equal("aGVsbG8gd29ybGQ="))

			Expect(m.Request).ToNot(BeNil())
		})
	})

	Describe("ValidateTTL", func() {
		It("Should allow messages within bounds", func() {
			m, err := NewMessageFromRequest(request, "reply.to", fw)
			Expect(err).ToNot(HaveOccurred())

			m.TimeStamp = now.Add(30 * time.Second)
			Expect(m.ValidateTTL()).To(BeTrue())
			m.TimeStamp = now.Add(-30 * time.Second)
			Expect(m.ValidateTTL()).To(BeTrue())
		})

		It("Should not allow messages out of bounds", func() {
			m, err := NewMessageFromRequest(request, "reply.to", fw)
			Expect(err).ToNot(HaveOccurred())

			m.TimeStamp = now.Add(90 * time.Second)
			Expect(m.ValidateTTL()).To(BeFalse())
			m.TimeStamp = now.Add(-90 * time.Second)
			Expect(m.ValidateTTL()).To(BeFalse())
		})
	})
})
