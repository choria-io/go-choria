// Copyright (c) 2020-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"fmt"
	"time"

	"github.com/choria-io/go-choria/inter"
	imock "github.com/choria-io/go-choria/inter/imocks"
	"github.com/choria-io/go-choria/message"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/choria-io/go-choria/protocol"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/ddl/agent"
)

var _ = Describe("McoRPC/Client/Options", func() {
	var (
		mockctl *gomock.Controller
		o       *RequestOptions
		fw      *imock.MockFramework
		err     error
	)

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())
		fw, _ = imock.NewFrameworkForTests(mockctl, GinkgoWriter, imock.WithCallerID())

		ddl, _ := agent.FindLocally("package", []string{"testdata"})
		o, err = NewRequestOptions(fw, ddl)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	Describe("ConfigureMessage", func() {
		It("Should configure the message", func() {
			msg, err := message.NewMessage(nil, "test", "mcollective", "request", nil, fw)
			Expect(err).ToNot(HaveOccurred())

			Targets([]string{"host1", "host2"})(o)
			BroadcastRequest()(o)

			err = o.ConfigureMessage(msg)
			Expect(err).ToNot(HaveOccurred())

			Expect(msg.DiscoveredHosts()).To(Equal([]string{"host1", "host2"}))
			Expect(o.Targets).To(Equal([]string{"host1", "host2"}))
			Expect(msg.Type()).To(Equal("request"))
			Expect(o.ReplyTo).To(Equal(msg.ReplyTo()))
			Expect(o.ProcessReplies).To(BeTrue())
			Expect(o.totalStats.discoveredNodes).To(Equal([]string{"host1", "host2"}))
			Expect(o.totalStats.RequestID).To(Equal(msg.RequestID()))
			Expect(o.RequestID).To(Equal(msg.RequestID()))
		})

		It("Should support the message supplying targets", func() {
			msg, err := message.NewMessage(nil, "test", "mcollective", "request", nil, fw)
			Expect(err).ToNot(HaveOccurred())
			msg.SetDiscoveredHosts([]string{"host1", "host2"})

			o.ConfigureMessage(msg)

			Expect(o.Targets).To(Equal([]string{"host1", "host2"}))
		})

		It("Should support custom reply targets", func() {
			msg, err := message.NewMessage(nil, "test", "mcollective", "request", nil, fw)
			Expect(err).ToNot(HaveOccurred())

			Targets([]string{"host1", "host2"})(o)
			ReplyTo("test.target")(o)

			o.ConfigureMessage(msg)

			Expect(msg.ReplyTo()).To(Equal("test.target"))
			Expect(o.ReplyTo).To(Equal(msg.ReplyTo()))
			Expect(o.ProcessReplies).To(BeFalse())
		})

		It("Should support limiting targets", func() {
			targets := make([]string, 100)
			for i := 0; i < 100; i++ {
				targets[i] = fmt.Sprintf("target%d", i)
			}

			msg, err := message.NewMessage(nil, "test", "mcollective", "request", nil, fw)
			Expect(err).ToNot(HaveOccurred())

			Targets(targets)(o)
			LimitMethod("first")(o)
			LimitSize("2")(o)
			o.ConfigureMessage(msg)
			Expect(o.Targets).To(Equal([]string{"target0", "target1"}))
			Expect(o.totalStats.discoveredNodes).To(Equal([]string{"target0", "target1"}))
		})

		It("Should support cached transports", func() {
			targets := make([]string, 100)
			for i := 0; i < 100; i++ {
				targets[i] = fmt.Sprintf("target%d", i)
			}

			msg, err := message.NewMessage(nil, "test", "mcollective", "request", nil, fw)
			Expect(err).ToNot(HaveOccurred())

			msg.CacheTransport()
			Targets(targets)(o)
			InBatches(10, 30)(o)
			DiscoveryTimeout(2 * time.Second)(o)
			Timeout(20 * time.Second)(o)
			msg.SetTTL(10)

			err = o.ConfigureMessage(msg)
			Expect(err).ToNot(HaveOccurred())

			expected := 10 * (10 + 2 + 20)
			Expect(msg.TTL()).To(Equal(expected))
		})

		It("Should limit cached TTL to 5 hours", func() {
			targets := make([]string, 100)
			for i := 0; i < 100; i++ {
				targets[i] = fmt.Sprintf("target%d", i)
			}

			msg, err := message.NewMessage(nil, "test", "mcollective", "request", nil, fw)
			Expect(err).ToNot(HaveOccurred())

			msg.CacheTransport()
			Targets(targets)(o)
			InBatches(10, 30)(o)
			DiscoveryTimeout(2 * time.Second)(o)
			Timeout(20 * time.Second)(o)
			msg.SetTTL(int(6 * time.Hour.Seconds()))

			err = o.ConfigureMessage(msg)
			Expect(err).To(MatchError("cached transport TTL is unreasonably long"))
		})

		It("Should support service requests", func() {
			msg, err := message.NewMessage(nil, "test", "mcollective", "request", nil, fw)
			Expect(err).ToNot(HaveOccurred())

			msg.CacheTransport()
			ServiceRequest()(o)
			err = o.ConfigureMessage(msg)
			Expect(err).ToNot(HaveOccurred())

			Expect(msg.Type()).To(Equal(inter.ServiceRequestMessageType))
			Expect(msg.Filter().Empty()).To(BeTrue())
			Expect(msg.DiscoveredHosts()).To(HaveLen(0))
		})
	})

	Describe("NewRequestOptions", func() {
		It("Should create correct new options", func() {
			Expect(o.ProtocolVersion).To(Equal(protocol.RequestV1))
			Expect(o.RequestType).To(Equal("direct_request"))
			Expect(o.Collective).To(Equal("ginkgo"))
			Expect(o.ProcessReplies).To(BeTrue())
			Expect(o.Timeout).To(Equal(time.Duration(182) * time.Second))
			Expect(o.fw).To(Equal(fw))
			Expect(o.LimitSeed).To(BeNumerically(">", 0))
			Expect(o.LimitMethod).To(Equal("first"))
		})
	})

	Describe("Stats", func() {
		It("Should return the stats", func() {
			Expect(o.Stats()).To(Equal(o.totalStats))
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
			Replies(make(chan inter.ConnectorMessage, 123))(o)
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

	Describe("LimitMethod", func() {
		It("Should set the method", func() {
			LimitMethod("random")(o)
			Expect(o.LimitMethod).To(Equal("random"))
		})
	})

	Describe("LimitSize", func() {
		It("Should set the size", func() {
			LimitSize("10%")(o)
			Expect(o.LimitSize).To(Equal("10%"))
		})
	})

	Describe("DiscoveryStartCB", func() {
		It("Should set the cb", func() {
			called := false
			cb := func() { called = true }
			DiscoveryStartCB(cb)(o)
			o.DiscoveryStartCB()
			Expect(called).To(BeTrue())
		})
	})

	Describe("DiscoveryStartCB", func() {
		It("Should set the cb", func() {
			called := false
			cb := func(_, _ int) error { called = true; return nil }

			DiscoveryEndCB(cb)(o)
			o.DiscoveryEndCB(0, 0)

			Expect(called).To(BeTrue())
		})
	})

	Describe("LimitSeed", func() {
		It("Should set the seed", func() {
			LimitSeed(100)(o)
			Expect(o.LimitSeed).To(Equal((int64(100))))
		})
	})

	Describe("limitTargets", func() {
		var targets []string

		BeforeEach(func() {
			targets = make([]string, 100)
			for i := 0; i < 100; i++ {
				targets[i] = fmt.Sprintf("target%d", i)
			}
		})

		It("Should not limit to 0", func() {
			o.LimitSize = "0"
			_, err := o.limitTargets(targets)
			Expect(err).To(MatchError("no targets left after applying target limits of '0'"))
		})

		It("Should accept only valid methods", func() {
			o.LimitMethod = "broken"
			l, err := o.limitTargets(targets)
			Expect(err).To(MatchError("limit method 'broken' is not valid, only 'random' or 'first' supported"))
			Expect(l).To(HaveLen(100))
		})

		It("Should return the supplied targets unshuffled when limit size is not set", func() {
			o.LimitSize = ""
			o.LimitMethod = "random"
			l, err := o.limitTargets(targets)
			Expect(err).ToNot(HaveOccurred())
			Expect(l).To(HaveLen(100))
			Expect(targets[0]).To(Equal("target0"))
			Expect(targets[20]).To(Equal("target20"))
			Expect(targets[30]).To(Equal("target30"))
			Expect(targets[40]).To(Equal("target40"))
			Expect(targets[50]).To(Equal("target50"))
			Expect(targets[99]).To(Equal("target99"))
		})

		It("Should limit to specific size and optionally shuffle the targets", func() {
			o.LimitSize = "5"
			o.LimitMethod = "first"
			l, err := o.limitTargets(targets)
			Expect(err).ToNot(HaveOccurred())
			Expect(l).To(HaveLen(5))
			Expect(l).To(Equal([]string{"target0", "target1", "target2", "target3", "target4"}))

			o.LimitMethod = "random"
			o.LimitSeed = 1
			l, err = o.limitTargets(targets)
			Expect(err).ToNot(HaveOccurred())
			Expect(l).To(HaveLen(5))
			Expect(l).To(Equal([]string{"target19", "target26", "target0", "target73", "target94"}))
		})

		It("Should limit to specific percentage and optionally shuffle the targets", func() {
			o.LimitSize = "5%"
			o.LimitMethod = "first"
			l, err := o.limitTargets(targets)
			Expect(err).ToNot(HaveOccurred())
			Expect(l).To(HaveLen(5))
			Expect(l).To(Equal(targets[0:5]))

			o.LimitMethod = "random"
			o.LimitSeed = 1
			l, err = o.limitTargets(targets)
			Expect(err).ToNot(HaveOccurred())
			Expect(l).To(HaveLen(5))
			Expect(l).To(Equal([]string{"target19", "target26", "target0", "target73", "target94"}))
		})
	})

	Describe("shuffleLimitedTargets", func() {
		var targets []string

		BeforeEach(func() {
			targets = make([]string, 100)
			for i := 0; i < 100; i++ {
				targets[i] = fmt.Sprintf("target%d", i)
			}
		})

		It("Should support not shuffling non random method targets", func() {
			o.LimitMethod = "first"
			o.shuffleLimitedTargets(targets)
			Expect(targets).To(HaveLen(100))
			Expect(targets[0]).To(Equal("target0"))
			Expect(targets[20]).To(Equal("target20"))
			Expect(targets[30]).To(Equal("target30"))
			Expect(targets[40]).To(Equal("target40"))
			Expect(targets[50]).To(Equal("target50"))
			Expect(targets[99]).To(Equal("target99"))
		})

		It("Should shuffle random method targets", FlakeAttempts(3), func() {
			o.LimitMethod = "random"
			o.LimitSeed = -1
			o.shuffleLimitedTargets(targets)
			Expect(targets).To(HaveLen(100))
			// small chance of failure here if random shuffling leaves these 2 in place
			for i := 0; i < 10; i++ {
				if targets[0] == "target0" || targets[99] == "target99" {
					o.shuffleLimitedTargets(targets)
				} else {
					break
				}
			}
			Expect(targets[0]).ToNot(Equal("target0"))
			Expect(targets[99]).ToNot(Equal("target99"))
			Expect(targets).To(HaveLen(100))
		})

		It("Should support seeds", func() {
			o.LimitMethod = "random"
			o.LimitSeed = 1
			o.shuffleLimitedTargets(targets)
			Expect(targets).To(HaveLen(100))
			Expect(targets[0]).To(Equal("target19"))
			Expect(targets[1]).To(Equal("target26"))
			Expect(targets[2]).To(Equal("target0"))
			Expect(targets[3]).To(Equal("target73"))
			Expect(targets).To(HaveLen(100))
		})
	})
})
