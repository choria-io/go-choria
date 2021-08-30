package message

import (
	"testing"
	"time"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/inter"
	imock "github.com/choria-io/go-choria/inter/imocks"
	v1 "github.com/choria-io/go-choria/protocol/v1"
	"github.com/choria-io/go-choria/providers/security/filesec"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/protocol"
)

func TestChoria(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Message")
}

var _ = Describe("Choria/Message", func() {
	var (
		mockctl *gomock.Controller
		request *MockRequest
		fw      *imock.MockFramework
		cfg     *config.Config
		now     time.Time
	)

	BeforeEach(func() {
		now = time.Now()

		mockctl = gomock.NewController(GinkgoT())

		request = NewMockRequest(mockctl)
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

		fw, cfg = imock.NewFrameworkForTests(mockctl, GinkgoWriter)
		cfg.Collectives = []string{"test_collective"}
		cfg.Choria.SSLDir = "/nonexisting"
		cfg.Identity = "test.identity"

		sec, err := filesec.New(filesec.WithChoriaConfig(&build.Info{}, cfg), filesec.WithLog(fw.Logger("")))
		Expect(err).ToNot(HaveOccurred())

		fw.EXPECT().CallerID().Return("choria=rip.mcollective").AnyTimes()
		fw.EXPECT().HasCollective(gomock.Eq("test_collective")).Return(true).AnyTimes()
		fw.EXPECT().HasCollective(gomock.Any()).Return(false).AnyTimes()
		fw.EXPECT().NewRequest(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(version string, agent string, senderid string, callerid string, ttl int, requestid string, collective string) (request protocol.Request, err error) {
			return v1.NewRequest(agent, senderid, callerid, ttl, requestid, collective)
		}).AnyTimes()
		fw.EXPECT().NewReplyTransportForMessage(gomock.Any(), gomock.Any()).DoAndReturn(func(msg inter.Message, request protocol.Request) (protocol.TransportMessage, error) {
			reply, err := v1.NewReply(request, cfg.Identity)
			Expect(err).ToNot(HaveOccurred())
			reply.SetMessage(msg.Payload())

			sreply, err := v1.NewSecureReply(reply, sec)
			Expect(err).ToNot(HaveOccurred())

			transport, err := v1.NewTransportMessage(cfg.Identity)
			Expect(err).ToNot(HaveOccurred())

			err = transport.SetReplyData(sreply)
			Expect(err).ToNot(HaveOccurred())

			return transport, nil
		}).AnyTimes()
		fw.EXPECT().NewRequestFromTransportJSON(gomock.Any(), gomock.Any()).DoAndReturn(func(payload []byte, skipvalidate bool) (msg protocol.Request, err error) {
			t, err := v1.NewTransportFromJSON(string(payload))
			Expect(err).ToNot(HaveOccurred())
			sreq, err := v1.NewSecureRequestFromTransport(t, sec, true)
			Expect(err).ToNot(HaveOccurred())
			return v1.NewRequestFromSecureRequest(sreq)
		}).AnyTimes()
		fw.EXPECT().NewRequestTransportForMessage(gomock.Any(), gomock.Any()).DoAndReturn(func(msg inter.Message, version string) (protocol.TransportMessage, error) {
			req, err := v1.NewRequest(msg.Agent(), msg.SenderID(), msg.CallerID(), msg.TTL(), msg.RequestID(), msg.Collective())
			Expect(err).ToNot(HaveOccurred())
			req.SetMessage(msg.Payload())

			sreq, err := v1.NewSecureRequest(req, sec)
			Expect(err).ToNot(HaveOccurred())

			sm, err := v1.NewTransportMessage(cfg.Identity)
			Expect(err).ToNot(HaveOccurred())
			err = sm.SetRequestData(sreq)
			Expect(err).ToNot(HaveOccurred())

			return sm, nil
		}).AnyTimes()
		protocol.Secure = "false"
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	Describe("NewMessageFromRequest", func() {
		It("Should create a new message with the correct properties", func() {
			m, err := NewMessageFromRequest(request, "reply.to", fw)
			Expect(err).ToNot(HaveOccurred())

			Expect(m.Payload()).To(Equal("hello world"))
			Expect(m.ReplyTo()).To(Equal("reply.to"))
			Expect(m.RequestID()).To(Equal("stub.request.id"))
			Expect(m.TimeStamp()).To(Equal(now))
			Expect(m.TTL()).To(Equal(60))
			Expect(m.Filter()).To(Equal(protocol.NewFilter()))
			Expect(m.SenderID()).To(Equal("test.identity"))
			Expect(m.Base64Payload()).To(Equal("aGVsbG8gd29ybGQ="))
			Expect(m.IsCachedTransport()).To(BeFalse())

			Expect(m.Request).ToNot(BeNil())

		})

		It("Should cache transports when configured to do so", func() {
			cfg.CacheBatchedTransports = true
			m, err := NewMessageFromRequest(request, "reply.to", fw)
			Expect(err).ToNot(HaveOccurred())
			Expect(m.IsCachedTransport()).To(BeTrue())
		})
	})

	Describe("NewMessage", func() {
		It("Should handle replies", func() {
			r, err := NewMessageFromRequest(request, "reply.to", fw)
			Expect(err).ToNot(HaveOccurred())

			m, err := NewMessage("hello world", "ginkgo", "test_collective", inter.ReplyMessageType, r, fw)
			Expect(err).ToNot(HaveOccurred())

			Expect(m.Request()).To(Equal(r))
			Expect(m.Agent()).To(Equal("test"))
			Expect(m.ReplyTo()).To(Equal("reply.to"))
			Expect(m.Type()).To(Equal(inter.ReplyMessageType))
			Expect(m.Collective()).To(Equal("test_collective"))
			Expect(m.IsCachedTransport()).To(BeFalse())
		})

		It("Should handle requests", func() {
			m, err := NewMessage("hello world", "ginkgo", "test_collective", inter.RequestMessageType, nil, fw)
			Expect(err).ToNot(HaveOccurred())

			Expect(m.Request()).To(BeNil())
			Expect(m.Agent()).To(Equal("ginkgo"))
			Expect(m.ReplyTo()).To(Equal(""))
			Expect(m.Type()).To(Equal(inter.RequestMessageType))
			Expect(m.Collective()).To(Equal("test_collective"))
		})

		It("Should validate", func() {
			_, err := NewMessage("hello world", "", "test_collective", inter.RequestMessageType, nil, fw)
			Expect(err).To(MatchError("agent has not been set"))

			_, err = NewMessage("hello world", "ginkgo", "mcollective", inter.RequestMessageType, nil, fw)
			Expect(err).To(MatchError("cannot set collective to 'mcollective', it is not on the list of known collectives"))
		})

		It("Should cache transports when configured to do so", func() {
			cfg.CacheBatchedTransports = true
			m, err := NewMessage("hello world", "ginkgo", "test_collective", inter.RequestMessageType, nil, fw)
			Expect(err).ToNot(HaveOccurred())
			Expect(m.IsCachedTransport()).To(BeTrue())
		})
	})

	Describe("Cached transports", func() {
		It("Should support setting and unsetting caching", func() {
			m, err := NewMessage("hello world", "ginkgo", "test_collective", inter.RequestMessageType, nil, fw)
			Expect(err).ToNot(HaveOccurred())
			Expect(m.IsCachedTransport()).To(BeFalse())
			Expect(m.IsCachedTransport()).To(BeFalse())
			m.CacheTransport()
			Expect(m.IsCachedTransport()).To(BeTrue())
			m.(*Message).UniqueTransport()
			Expect(m.IsCachedTransport()).To(BeFalse())
		})
	})
	Describe("Transport", func() {
		It("Should support requests", func() {
			m, err := NewMessage("hello world", "ginkgo", "test_collective", inter.RequestMessageType, nil, fw)
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

			m, err := NewMessage("hello world", "ginkgo", "test_collective", inter.RequestMessageType, nil, fw)
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
			m, err := NewMessage("hello world", "ginkgo", "test_collective", inter.RequestMessageType, nil, fw)
			Expect(err).ToNot(HaveOccurred())

			m.SetDiscoveredHosts([]string{"node1", "node2"})
			err = m.SetType(inter.DirectRequestMessageType)
			Expect(err).ToNot(HaveOccurred())

			m.SetProtocolVersion(protocol.RequestV1)
			m.SetReplyTo("reply.to")

			_, err = m.Transport()
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should support service_requests", func() {
			m, err := NewMessage("hello world", "ginkgo", "test_collective", inter.ServiceRequestMessageType, nil, fw)
			Expect(err).ToNot(HaveOccurred())

			m.SetDiscoveredHosts([]string{"node1", "node2"})
			err = m.SetType(inter.DirectRequestMessageType)
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
			m, err := NewMessage("hello world", "ginkgo", "test_collective", inter.RequestMessageType, nil, fw)
			Expect(err).ToNot(HaveOccurred())

			_, err = m.(*Message).UncachedRequestTransport()
			Expect(err).To(MatchError("cannot create a Request Transport without a version, please set it using SetProtocolVersion()"))
		})

		It("Should require a reply-to", func() {
			m, err := NewMessage("hello world", "ginkgo", "test_collective", inter.RequestMessageType, nil, fw)
			Expect(err).ToNot(HaveOccurred())

			m.SetProtocolVersion(protocol.RequestV1)

			_, err = m.(*Message).UncachedRequestTransport()
			Expect(err).To(MatchError("cannot create a Transport, no reply-to was set, please use SetReplyTo()"))

		})

		It("Should prevent empty filters when configured to do so", func() {
			cfg.Choria.RequireClientFilter = true
			m, err := NewMessage("hello world", "ginkgo", "test_collective", inter.RequestMessageType, nil, fw)
			Expect(err).ToNot(HaveOccurred())
			m.SetProtocolVersion(protocol.RequestV1)
			m.SetReplyTo("reply.to")

			_, err = m.(*Message).UncachedRequestTransport()
			Expect(err).To(MatchError("cannot create a Request Transport, requests without filters have been disabled"))

			cfg.Choria.RequireClientFilter = false
			_, err = m.(*Message).UncachedRequestTransport()
			Expect(err).ToNot(HaveOccurred())

			cfg.Choria.RequireClientFilter = true
			m.Filter().AddClassFilter("foo")
			_, err = m.(*Message).UncachedRequestTransport()
			Expect(err).ToNot(HaveOccurred())

			// discovery has m.Agent==discovery but the filter agent will be what the next request will target so special case tests
			cfg.Choria.RequireClientFilter = true
			m, err = NewMessage("hello world", "discovery", "test_collective", inter.RequestMessageType, nil, fw)
			Expect(err).ToNot(HaveOccurred())
			m.SetProtocolVersion(protocol.RequestV1)
			m.SetReplyTo("reply.to")
			m.Filter().AddAgentFilter("rpcutil")
			_, err = m.(*Message).UncachedRequestTransport()
			Expect(err).To(MatchError("cannot create a Request Transport, requests without filters have been disabled"))

		})

		It("Should set up the transport", func() {
			m, err := NewMessage("hello world", "ginkgo", "test_collective", inter.RequestMessageType, nil, fw)
			Expect(err).ToNot(HaveOccurred())

			m.SetProtocolVersion(protocol.RequestV1)
			m.SetReplyTo("reply.to")

			t, err := m.(*Message).UncachedRequestTransport()
			Expect(err).ToNot(HaveOccurred())
			Expect(t.ReplyTo()).To(Equal("reply.to"))
			Expect(t.SenderID()).To(Equal("test.identity"))

			j, err := t.JSON()
			Expect(err).ToNot(HaveOccurred())
			r, err := fw.NewRequestFromTransportJSON([]byte(j), true)
			Expect(err).ToNot(HaveOccurred())

			Expect(r.Agent()).To(Equal("ginkgo"))
			Expect(r.RequestID()).To(Equal(m.RequestID()))
		})
	})

	Describe("replyTransport", func() {
		It("Should require a request", func() {
			m, err := NewMessage("hello world", "ginkgo", "test_collective", inter.ReplyMessageType, nil, fw)
			Expect(err).ToNot(HaveOccurred())

			_, err = m.(*Message).UncachedReplyTransport()
			Expect(err).To(MatchError("cannot create a Transport, no request were stored in the message"))
		})

		It("Should set up the transport", func() {
			rid, err := fw.NewRequestID()
			Expect(err).ToNot(HaveOccurred())

			req, err := v1.NewRequest("test_agent", "sender.example.net", "test=sender", 60, rid, "test_collective")
			Expect(err).ToNot(HaveOccurred())
			req.SetMessage("hello world")

			m, err := NewMessageFromRequest(req, "reply.to", fw)
			Expect(err).ToNot(HaveOccurred())

			t, err := m.(*Message).UncachedReplyTransport()
			Expect(err).ToNot(HaveOccurred())
			Expect(t).ToNot(BeNil())
		})
	})

	Describe("SetProtocolVersion", func() {
		It("Should set the version", func() {
			m, err := NewMessage("hello world", "ginkgo", "test_collective", inter.ReplyMessageType, nil, fw)
			Expect(err).ToNot(HaveOccurred())

			Expect(m.ProtocolVersion()).To(Equal(""))
			m.SetProtocolVersion(protocol.ReplyV1)
			Expect(m.ProtocolVersion()).To(Equal(protocol.ReplyV1))
		})
	})

	Describe("Validate", func() {
		It("Should validate the message", func() {
			m, err := NewMessage("hello world", "ginkgo", "test_collective", inter.ReplyMessageType, nil, fw)
			Expect(err).ToNot(HaveOccurred())

			m.(*Message).collective = "foo"
			ok, err := m.Validate()
			Expect(ok).To(BeFalse())
			Expect(err).To(MatchError("'foo' is not on the list of known collectives"))

			m.(*Message).collective = ""
			ok, err = m.Validate()
			Expect(ok).To(BeFalse())
			Expect(err).To(MatchError("collective has not been set"))

			m.(*Message).agent = ""
			ok, err = m.Validate()
			Expect(ok).To(BeFalse())
			Expect(err).To(MatchError("agent has not been set"))
		})
	})

	Describe("SetBase64Payload", func() {
		It("Should store the correct payload", func() {
			m, err := NewMessage("hello world", "ginkgo", "test_collective", inter.ReplyMessageType, nil, fw)
			Expect(err).ToNot(HaveOccurred())

			err = m.SetBase64Payload("aGVsbG8gd29ybGQ=")
			Expect(err).ToNot(HaveOccurred())

			Expect(m.Payload()).To(Equal("hello world"))
		})

		It("Should handle invalid base64", func() {
			m, err := NewMessage("hello world", "ginkgo", "test_collective", inter.ReplyMessageType, nil, fw)
			Expect(err).ToNot(HaveOccurred())

			err = m.SetBase64Payload("foo")
			Expect(err).To(MatchError("could not decode supplied payload using base64: illegal base64 data at input byte 0"))
		})
	})

	Describe("SetExpectedMsgID", func() {
		It("Should only set it for reply messages", func() {
			m, err := NewMessage("hello world", "ginkgo", "test_collective", inter.RequestMessageType, nil, fw)
			Expect(err).ToNot(HaveOccurred())

			err = m.SetExpectedMsgID("x")
			Expect(err).To(MatchError("can only store expected message ID for reply messages"))
		})

		It("Should store the expectation", func() {
			m, err := NewMessage("hello world", "ginkgo", "test_collective", inter.ReplyMessageType, nil, fw)
			Expect(err).ToNot(HaveOccurred())

			err = m.SetExpectedMsgID("x")
			Expect(err).ToNot(HaveOccurred())
			Expect(m.ExpectedMessageID()).To(Equal("x"))
		})
	})

	Describe("SetReplyTo", func() {
		It("Should set it only for requests", func() {
			m, err := NewMessage("hello world", "ginkgo", "test_collective", inter.ReplyMessageType, nil, fw)
			Expect(err).ToNot(HaveOccurred())

			err = m.SetReplyTo("reply.to")
			Expect(err).To(MatchError("custom reply to targets can only be set for requests"))
		})

		It("Should set it correctly", func() {
			m, err := NewMessage("hello world", "ginkgo", "test_collective", inter.RequestMessageType, nil, fw)
			Expect(err).ToNot(HaveOccurred())

			err = m.SetReplyTo("reply.to")
			Expect(err).ToNot(HaveOccurred())
			Expect(m.ReplyTo()).To(Equal("reply.to"))
		})
	})

	Describe("SetType", func() {
		It("Should only allow valid types", func() {
			m, err := NewMessage("hello world", "ginkgo", "test_collective", inter.RequestMessageType, nil, fw)
			Expect(err).ToNot(HaveOccurred())

			err = m.SetType("bob")
			Expect(err).To(MatchError("bob is not a valid message type"))

			for _, t := range []string{"message", "request", "reply"} {
				err = m.SetType(t)
				Expect(err).ToNot(HaveOccurred())
				Expect(m.Type()).To(Equal(t))
			}

			err = m.SetType(inter.DirectRequestMessageType)
			Expect(err).To(MatchError("direct_request message type can only be set if DiscoveredHosts have been set"))

			m.SetDiscoveredHosts([]string{"node1"})
			err = m.SetType(inter.DirectRequestMessageType)
			Expect(err).ToNot(HaveOccurred())
			Expect(m.Type()).To(Equal(inter.DirectRequestMessageType))
		})
	})

	Describe("SetCollective", func() {
		It("Should only accept valid collectives", func() {
			m, err := NewMessage("hello world", "ginkgo", "test_collective", inter.RequestMessageType, nil, fw)
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

			m.(*Message).timeStamp = now.Add(30 * time.Second)
			Expect(m.ValidateTTL()).To(BeTrue())

			m.(*Message).timeStamp = now.Add(-30 * time.Second)
			Expect(m.ValidateTTL()).To(BeTrue())
		})

		It("Should not allow messages out of bounds", func() {
			m, err := NewMessageFromRequest(request, "reply.to", fw)
			Expect(err).ToNot(HaveOccurred())

			m.(*Message).timeStamp = now.Add(90 * time.Second)
			Expect(m.ValidateTTL()).To(BeFalse())
			m.(*Message).timeStamp = now.Add(-90 * time.Second)
			Expect(m.ValidateTTL()).To(BeFalse())
		})
	})
})
