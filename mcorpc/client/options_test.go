package client

import (
	"time"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/mcorpc/ddl/agent"

	"github.com/choria-io/go-protocol/protocol"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("McoRPC/Client/Options", func() {
	var (
		o  *RequestOptions
		fw *choria.Framework
	)

	BeforeEach(func() {
		cfg, _ := config.NewConfig("testdata/default.cfg")
		fw, _ = choria.NewWithConfig(cfg)
		ddl, _ := agent.Find("package", []string{"testdata"})
		o = NewRequestOptions(fw, ddl)
	})

	Describe("ConfigureMessage", func() {
		It("Should configure the message", func() {
			msg, err := fw.NewMessage("", "test", "mcollective", "request", nil)
			Expect(err).ToNot(HaveOccurred())

			Targets([]string{"host1", "host2"})(o)
			BroadcastRequest()(o)

			err = o.ConfigureMessage(msg)
			Expect(err).ToNot(HaveOccurred())

			Expect(msg.DiscoveredHosts).To(Equal([]string{"host1", "host2"}))
			Expect(o.Targets).To(Equal([]string{"host1", "host2"}))
			Expect(msg.Type()).To(Equal("request"))
			Expect(o.ReplyTo).To(Equal(msg.ReplyTo()))
			Expect(o.ProcessReplies).To(BeTrue())
			Expect(o.totalStats.discoveredNodes).To(Equal([]string{"host1", "host2"}))
			Expect(o.totalStats.RequestID).To(Equal(msg.RequestID))
			Expect(o.RequestID).To(Equal(msg.RequestID))
		})

		It("Should support the message supplying targets", func() {
			msg, err := fw.NewMessage("", "test", "mcollective", "request", nil)
			Expect(err).ToNot(HaveOccurred())

			msg.DiscoveredHosts = []string{"host1", "host2"}

			o.ConfigureMessage(msg)

			Expect(msg.DiscoveredHosts).To(Equal([]string{"host1", "host2"}))
			Expect(o.Targets).To(Equal([]string{"host1", "host2"}))
		})

		It("Should support custom reply targets", func() {
			msg, err := fw.NewMessage("", "test", "mcollective", "request", nil)
			Expect(err).ToNot(HaveOccurred())

			Targets([]string{"host1", "host2"})(o)
			ReplyTo("test.target")(o)

			o.ConfigureMessage(msg)

			Expect(msg.ReplyTo()).To(Equal("test.target"))
			Expect(o.ReplyTo).To(Equal(msg.ReplyTo()))
			Expect(o.ProcessReplies).To(BeFalse())
		})
	})

	Describe("NewRequestOptions", func() {
		It("Should create correct new options", func() {
			Expect(o.ProtocolVersion).To(Equal(protocol.RequestV1))
			Expect(o.RequestType).To(Equal("direct_request"))
			Expect(o.Collective).To(Equal("mcollective"))
			Expect(o.ProcessReplies).To(BeTrue())
			Expect(o.Progress).To(BeFalse())
			Expect(o.Timeout).To(Equal(time.Duration(182) * time.Second))
			Expect(o.stats).ToNot(BeNil())
			Expect(o.fw).To(Equal(fw))
		})
	})

	Describe("Stats", func() {
		It("Should return the stats", func() {
			Expect(o.Stats()).To(Equal(o.stats))
		})
	})

	Describe("WithProgress", func() {
		It("Should enable progress reporting", func() {
			WithProgress()(o)
			Expect(o.Progress).To(BeTrue())
		})
	})

	Describe("Targets", func() {
		It("Should set the targets", func() {
			Targets([]string{"host1"})(o)
			Expect(o.Targets).To(Equal([]string{"host1"}))
		})
	})

	Describe("Protocol", func() {
		It("Should set the protocol to use", func() {
			Protocol(protocol.RequestV1)(o)
			Expect(o.ProtocolVersion).To(Equal(protocol.RequestV1))
		})
	})

	Describe("DirectRequest", func() {
		It("Should set the type", func() {
			DirectRequest()(o)
			Expect(o.RequestType).To(Equal("direct_request"))
		})
	})

	Describe("BroadcastRequest", func() {
		It("Should set the type", func() {
			BroadcastRequest()(o)
			Expect(o.RequestType).To(Equal("request"))
		})
	})

	Describe("Workers", func() {
		It("Should set the workers", func() {
			Workers(10)(o)
			Expect(o.Workers).To(Equal(10))
		})
	})

	Describe("Collective", func() {
		It("Should set the collective", func() {
			Collective("bob")(o)
			Expect(o.Collective).To(Equal("bob"))
		})
	})

	Describe("ReplyTo", func() {
		It("Should set the target", func() {
			ReplyTo("bob")(o)
			Expect(o.ReplyTo).To(Equal("bob"))
		})
	})

	Describe("InBatches", func() {
		It("Should set the size, batched mode and sleep", func() {
			InBatches(10, 5)(o)
			Expect(o.BatchSize).To(Equal(10))
			Expect(o.BatchSleep).To(Equal(5 * time.Second))
		})
	})

	Describe("Replies", func() {
		It("Should set the channel and disable the handlers", func() {
			Replies(make(chan *choria.ConnectorMessage, 123))(o)
			Expect(o.Replies).To(HaveCap(123))
		})
	})

	Describe("Timeout", func() {
		It("Should set the timeout", func() {
			Timeout(10 * time.Second)(o)
			Expect(o.Timeout).To(Equal(10 * time.Second))
		})
	})

	Describe("ReplyHandler", func() {
		It("Should set the handler", func() {
			seen := false

			ReplyHandler(func(p protocol.Reply, r *RPCReply) { seen = true })(o)

			o.Handler(nil, nil)

			Expect(seen).To(BeTrue())
		})
	})

	Describe("ConnectionName", func() {
		It("Should set the name", func() {
			ConnectionName("ginkgo")(o)
			Expect(o.ConnectionName).To(Equal("ginkgo"))
		})
	})
})
