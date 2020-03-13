package choria

import (
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/protocol"
)

var _ = Describe("Choria/Message", func() {
	var (
		mockctl *gomock.Controller
		request *MockRequest
		fw      *Framework
		now     time.Time
		err     error
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
		request.EXPECT().Version().Return(protocol.RequestV1).AnyTimes()
		request.EXPECT().IsFederated().Return(false).AnyTimes()
		request.EXPECT().JSON().Return("{\"mock_request\": true}", nil).AnyTimes()

		cfg := config.NewConfigForTests()
		cfg.Choria.SSLDir = "/nonexisting"
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
			Expect(m.shouldCacheTransport).To(BeFalse())

			Expect(m.Request).ToNot(BeNil())

		})

		It("Should cache transports when configured to do so", func() {
			fw.Config.CacheBatchedTransports = true
			m, err := NewMessageFromRequest(request, "reply.to", fw)
			Expect(err).ToNot(HaveOccurred())
			Expect(m.shouldCacheTransport).To(BeTrue())
		})
	})

	Describe("NewMessage", func() {
		It("Should handle replies", func() {
			r, err := NewMessageFromRequest(request, "reply.to", fw)
			Expect(err).ToNot(HaveOccurred())

			m, err := NewMessage("hello world", "ginkgo", "test_collective", "reply", r, fw)
			Expect(err).ToNot(HaveOccurred())

			Expect(m.Request).To(Equal(r))
			Expect(m.Agent).To(Equal("test"))
			Expect(m.replyTo).To(Equal("reply.to"))
			Expect(m.Type()).To(Equal("reply"))
			Expect(m.Collective()).To(Equal("test_collective"))
			Expect(m.shouldCacheTransport).To(BeFalse())
		})

		It("Should handle requests", func() {
			m, err := NewMessage("hello world", "ginkgo", "test_collective", "request", nil, fw)
			Expect(err).ToNot(HaveOccurred())

			Expect(m.Request).To(BeNil())
			Expect(m.Agent).To(Equal("ginkgo"))
			Expect(m.replyTo).To(Equal(""))
			Expect(m.Type()).To(Equal("request"))
			Expect(m.Collective()).To(Equal("test_collective"))
		})

		It("Should validate", func() {
			_, err := NewMessage("hello world", "", "test_collective", "request", nil, fw)
			Expect(err).To(MatchError("agent has not been set"))

			_, err = NewMessage("hello world", "ginkgo", "mcollective", "request", nil, fw)
			Expect(err).To(MatchError("cannot set collective to 'mcollective', it is not on the list of known collectives"))
		})

		It("Should cache transports when configured to do so", func() {
			fw.Config.CacheBatchedTransports = true
			m, err := NewMessage("hello world", "ginkgo", "test_collective", "request", nil, fw)
			Expect(err).ToNot(HaveOccurred())
			Expect(m.shouldCacheTransport).To(BeTrue())
		})
	})

	Describe("Cached transports", func() {
		It("Should support setting and unsetting caching", func() {
			m, err := NewMessage("hello world", "ginkgo", "test_collective", "request", nil, fw)
			Expect(err).ToNot(HaveOccurred())
			Expect(m.shouldCacheTransport).To(BeFalse())
			Expect(m.IsCachedTransport()).To(BeFalse())
			m.CacheTransport()
			Expect(m.IsCachedTransport()).To(BeTrue())
			m.UniqueTransport()
			Expect(m.IsCachedTransport()).To(BeFalse())
		})
	})
	Describe("Transport", func() {
		It("Should support requests", func() {
			m, err := NewMessage("hello world", "ginkgo", "test_collective", "request", nil, fw)
			Expect(err).ToNot(HaveOccurred())

			m.SetProtocolVersion(protocol.RequestV1)
			m.SetReplyTo("reply.to")

			t1, err := m.Transport()
			Expect(err).ToNot(HaveOccurred())
			t1m, err := t1.Message()
			Expect(err).ToNot(HaveOccurred())

			// force the body to change, and so the payload must change
			time.Sleep(time.Second)

			t2, err := m.Transport()
			Expect(err).ToNot(HaveOccurred())
			t2m, err := t2.Message()
			Expect(err).ToNot(HaveOccurred())

			Expect(t1m).ToNot(Equal(t2m))
		})

		It("Should support cached transports", func() {
			fw.Configuration().CacheBatchedTransports = true

			m, err := NewMessage("hello world", "ginkgo", "test_collective", "request", nil, fw)
			Expect(err).ToNot(HaveOccurred())

			m.SetProtocolVersion(protocol.RequestV1)
			m.SetReplyTo("reply.to")

			t1, err := m.Transport()
			Expect(err).ToNot(HaveOccurred())
			t1m, err := t1.Message()
			Expect(err).ToNot(HaveOccurred())

			// force the body to change, and so the payload must change, but due to cache the result should be identical
			time.Sleep(time.Second)

			t2, err := m.Transport()
			Expect(err).ToNot(HaveOccurred())
			t2m, err := t2.Message()
			Expect(err).ToNot(HaveOccurred())

			Expect(t1m).To(Equal(t2m))
		})

		It("Should support direct_requests", func() {
			m, err := NewMessage("hello world", "ginkgo", "test_collective", "request", nil, fw)
			Expect(err).ToNot(HaveOccurred())

			m.DiscoveredHosts = []string{"node1", "node2"}
			err = m.SetType("direct_request")
			Expect(err).ToNot(HaveOccurred())

			m.SetProtocolVersion(protocol.RequestV1)
			m.SetReplyTo("reply.to")

			_, err = m.Transport()
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should support reply", func() {
			rid, err := fw.NewRequestID()
			Expect(err).ToNot(HaveOccurred())

			req, err := fw.NewRequest(protocol.RequestV1, "test_agent", "sender.example.net", "test=sender", 60, rid, "test_collective")
			Expect(err).ToNot(HaveOccurred())
			req.SetMessage("hello world")

			m, err := NewMessageFromRequest(req, "reply.to", fw)
			Expect(err).ToNot(HaveOccurred())

			t, err := m.Transport()
			Expect(err).ToNot(HaveOccurred())
			Expect(t).ToNot(BeNil())
		})
	})

	Describe("requestTransport", func() {
		It("Should require a version", func() {
			m, err := NewMessage("hello world", "ginkgo", "test_collective", "request", nil, fw)
			Expect(err).ToNot(HaveOccurred())

			_, err = m.requestTransport()
			Expect(err).To(MatchError("cannot create a Request Transport without a version, please set it using SetProtocolVersion()"))
		})

		It("Should require a reply-to", func() {
			m, err := NewMessage("hello world", "ginkgo", "test_collective", "request", nil, fw)
			Expect(err).ToNot(HaveOccurred())

			m.SetProtocolVersion(protocol.RequestV1)

			_, err = m.requestTransport()
			Expect(err).To(MatchError("cannot create a Transport, no reply-to was set, please use SetReplyTo()"))

		})

		It("Should prevent empty filters when configured to do so", func() {
			fw.Config.Choria.RequireClientFilter = true
			m, err := NewMessage("hello world", "ginkgo", "test_collective", "request", nil, fw)
			Expect(err).ToNot(HaveOccurred())
			m.SetProtocolVersion(protocol.RequestV1)
			m.SetReplyTo("reply.to")

			_, err = m.requestTransport()
			Expect(err).To(MatchError("cannot create a Request Transport, requests without filters have been disabled"))

			fw.Config.Choria.RequireClientFilter = false
			_, err = m.requestTransport()
			Expect(err).ToNot(HaveOccurred())

			fw.Config.Choria.RequireClientFilter = true
			m.Filter.AddClassFilter("foo")
			_, err = m.requestTransport()
			Expect(err).ToNot(HaveOccurred())

			// discovery has m.Agent==discovery but the filter agent will be what the next request will target so special case tests
			fw.Config.Choria.RequireClientFilter = true
			m, err = NewMessage("hello world", "discovery", "test_collective", "request", nil, fw)
			Expect(err).ToNot(HaveOccurred())
			m.SetProtocolVersion(protocol.RequestV1)
			m.SetReplyTo("reply.to")
			m.Filter.AddAgentFilter("rpcutil")
			_, err = m.requestTransport()
			Expect(err).To(MatchError("cannot create a Request Transport, requests without filters have been disabled"))

		})

		It("Should set up the transport", func() {
			m, err := NewMessage("hello world", "ginkgo", "test_collective", "request", nil, fw)
			Expect(err).ToNot(HaveOccurred())

			m.SetProtocolVersion(protocol.RequestV1)
			m.SetReplyTo("reply.to")

			t, err := m.requestTransport()
			Expect(err).ToNot(HaveOccurred())
			Expect(t.ReplyTo()).To(Equal("reply.to"))
			Expect(t.SenderID()).To(Equal("test.identity"))

			j, err := t.JSON()
			Expect(err).ToNot(HaveOccurred())
			r, err := fw.NewRequestFromTransportJSON([]byte(j), true)
			Expect(err).ToNot(HaveOccurred())

			Expect(r.Agent()).To(Equal("ginkgo"))
			Expect(r.RequestID()).To(Equal(m.RequestID))
		})
	})

	Describe("replyTransport", func() {
		It("Should require a request", func() {
			m, err := NewMessage("hello world", "ginkgo", "test_collective", "reply", nil, fw)
			Expect(err).ToNot(HaveOccurred())

			_, err = m.replyTransport()
			Expect(err).To(MatchError("cannot create a Transport, no request were stored in the message"))
		})

		It("Should set up the transport", func() {
			rid, err := fw.NewRequestID()
			Expect(err).ToNot(HaveOccurred())

			req, err := fw.NewRequest(protocol.RequestV1, "test_agent", "sender.example.net", "test=sender", 60, rid, "test_collective")
			Expect(err).ToNot(HaveOccurred())
			req.SetMessage("hello world")

			m, err := NewMessageFromRequest(req, "reply.to", fw)
			Expect(err).ToNot(HaveOccurred())

			t, err := m.replyTransport()
			Expect(err).ToNot(HaveOccurred())
			Expect(t).ToNot(BeNil())
		})
	})

	Describe("SetProtocolVersion", func() {
		It("Should set the version", func() {
			m, err := NewMessage("hello world", "ginkgo", "test_collective", "reply", nil, fw)
			Expect(err).ToNot(HaveOccurred())

			Expect(m.protoVersion).To(Equal(""))
			m.SetProtocolVersion(protocol.ReplyV1)
			Expect(m.protoVersion).To(Equal(protocol.ReplyV1))
		})
	})

	Describe("Validate", func() {
		It("Should validate the message", func() {
			m, err := NewMessage("hello world", "ginkgo", "test_collective", "reply", nil, fw)
			Expect(err).ToNot(HaveOccurred())

			m.collective = "foo"
			ok, err := m.Validate()
			Expect(ok).To(BeFalse())
			Expect(err).To(MatchError("'foo' is not on the list of known collectives"))

			m.collective = ""
			ok, err = m.Validate()
			Expect(ok).To(BeFalse())
			Expect(err).To(MatchError("collective has not been set"))

			m.Agent = ""
			ok, err = m.Validate()
			Expect(ok).To(BeFalse())
			Expect(err).To(MatchError("agent has not been set"))
		})
	})

	Describe("SetBase64Payload", func() {
		It("Should store the correct payload", func() {
			m, err := NewMessage("hello world", "ginkgo", "test_collective", "reply", nil, fw)
			Expect(err).ToNot(HaveOccurred())

			err = m.SetBase64Payload("aGVsbG8gd29ybGQ=")
			Expect(err).ToNot(HaveOccurred())

			Expect(m.Payload).To(Equal("hello world"))
		})

		It("Should handle invalid base64", func() {
			m, err := NewMessage("hello world", "ginkgo", "test_collective", "reply", nil, fw)
			Expect(err).ToNot(HaveOccurred())

			err = m.SetBase64Payload("foo")
			Expect(err).To(MatchError("could not decode supplied payload using base64: illegal base64 data at input byte 0"))
		})
	})

	Describe("SetExpectedMsgID", func() {
		It("Should only set it for reply messages", func() {
			m, err := NewMessage("hello world", "ginkgo", "test_collective", "request", nil, fw)
			Expect(err).ToNot(HaveOccurred())

			err = m.SetExpectedMsgID("x")
			Expect(err).To(MatchError("can only store expected message ID for reply messages"))
		})

		It("Should store the expectation", func() {
			m, err := NewMessage("hello world", "ginkgo", "test_collective", "reply", nil, fw)
			Expect(err).ToNot(HaveOccurred())

			err = m.SetExpectedMsgID("x")
			Expect(err).ToNot(HaveOccurred())
			Expect(m.ExpectedMessageID()).To(Equal("x"))
		})
	})

	Describe("SetReplyTo", func() {
		It("Should set it only for requests", func() {
			m, err := NewMessage("hello world", "ginkgo", "test_collective", "reply", nil, fw)
			Expect(err).ToNot(HaveOccurred())

			err = m.SetReplyTo("reply.to")
			Expect(err).To(MatchError("custom reply to targets can only be set for requests"))
		})

		It("Should set it correctly", func() {
			m, err := NewMessage("hello world", "ginkgo", "test_collective", "request", nil, fw)
			Expect(err).ToNot(HaveOccurred())

			err = m.SetReplyTo("reply.to")
			Expect(err).ToNot(HaveOccurred())
			Expect(m.ReplyTo()).To(Equal("reply.to"))
		})
	})

	Describe("SetType", func() {
		It("Should only allow valid types", func() {
			m, err := NewMessage("hello world", "ginkgo", "test_collective", "request", nil, fw)
			Expect(err).ToNot(HaveOccurred())

			err = m.SetType("bob")
			Expect(err).To(MatchError("bob is not a valid message type"))

			for _, t := range []string{"message", "request", "reply"} {
				err = m.SetType(t)
				Expect(err).ToNot(HaveOccurred())
				Expect(m.Type()).To(Equal(t))
			}

			err = m.SetType("direct_request")
			Expect(err).To(MatchError("direct_request message type can only be set if DiscoveredHosts have been set"))

			m.DiscoveredHosts = []string{"node1"}
			err = m.SetType("direct_request")
			Expect(err).ToNot(HaveOccurred())
			Expect(m.Type()).To(Equal("direct_request"))
		})
	})

	Describe("SetCollective", func() {
		It("Should only accept valid collectives", func() {
			m, err := NewMessage("hello world", "ginkgo", "test_collective", "request", nil, fw)
			Expect(err).ToNot(HaveOccurred())

			err = m.SetCollective("bob")
			Expect(err).To(MatchError("cannot set collective to 'bob', it is not on the list of known collectives"))

			err = m.SetCollective("test_collective")
			Expect(err).ToNot(HaveOccurred())
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
