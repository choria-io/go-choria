package agents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/protocol"
	"github.com/golang/mock/gomock"

	"github.com/choria-io/go-choria/choria"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func Test(t *testing.T) {
	os.Setenv("MCOLLECTIVE_CERTNAME", "rip.mcollective")
	RegisterFailHandler(Fail)
	RunSpecs(t, "Server/Agents")
}

var _ = Describe("Server/Agents", func() {
	var (
		mockctl  *gomock.Controller
		mgr      *Manager
		conn     *MockAgentConnector
		agent    *MockAgent
		requests chan *choria.ConnectorMessage
		ctx      context.Context
		cancel   func()
		fw       *choria.Framework
		handler  func(ctx context.Context, msg *choria.Message, request protocol.Request, ci choria.ConnectorInfo, result chan *AgentReply)
		err      error
	)

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())

		cfg := config.NewConfigForTests()
		cfg.DisableTLS = true

		fw, err = choria.NewWithConfig(cfg)
		Expect(err).ToNot(HaveOccurred())
		fw.Config.Collectives = []string{"cone", "ctwo"}
		fw.SetLogWriter(GinkgoWriter)

		requests = make(chan *choria.ConnectorMessage)
		ctx, cancel = context.WithCancel(context.Background())

		metadata := Metadata{
			Author:      "stub@example.net",
			Description: "Stub Agent",
			License:     "Apache-2.0",
			Name:        "stub_agent",
			Timeout:     10,
			URL:         "https://choria.io/",
			Version:     "1.0.0",
		}

		handler = func(ctx context.Context, msg *choria.Message, request protocol.Request, ci choria.ConnectorInfo, result chan *AgentReply) {
			if msg.Payload == "sleep" {
				time.Sleep(10 * time.Second)
			}

			reply := &AgentReply{
				Body:    []byte(fmt.Sprintf("pong %s", msg.Payload)),
				Message: msg,
				Request: request,
			}

			result <- reply
		}

		is := NewMockServerInfoSource(mockctl)
		is.EXPECT().KnownAgents().Return([]string{"stub_agent"}).AnyTimes()
		is.EXPECT().Classes().Return([]string{"one", "two"}).AnyTimes()
		is.EXPECT().Facts().Return(json.RawMessage(`{"stub":true}`)).AnyTimes()
		is.EXPECT().AgentMetadata("stub_agent").Return(metadata, true).AnyTimes()

		mgr = New(requests, fw, conn, is, logrus.WithFields(logrus.Fields{"testing": true}))
		conn = NewMockAgentConnector(mockctl)

		agent = NewMockAgent(mockctl)
		agent.EXPECT().Metadata().Return(&metadata).AnyTimes()
		agent.EXPECT().SetServerInfo(is).Return().AnyTimes()
		agent.EXPECT().HandleMessage(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(handler).AnyTimes()
	})

	AfterEach(func() {
		cancel()
		mockctl.Finish()
	})

	Describe("DenyAgent", func() {
		It("Should add the agent to the deny list", func() {
			Expect(mgr.denylist).To(BeEmpty())
			Expect(mgr.agentDenied("testing")).To(BeFalse())
			mgr.DenyAgent("testing")
			Expect(mgr.denylist).To(Equal([]string{"testing"}))
			Expect(mgr.agentDenied("testing")).To(BeTrue())
		})
	})

	Describe("RegisterAgent", func() {
		It("Should honor the ShouldActivate wish of the agent", func() {
			agent.EXPECT().ShouldActivate().Return(false).Times(1)
			err := mgr.RegisterAgent(ctx, "testing", agent, conn)
			Expect(err).ToNot(HaveOccurred())
			Expect(mgr.KnownAgents()).To(BeEmpty())
		})

		It("Should honor the deny list", func() {
			agent.EXPECT().ShouldActivate().Return(true).AnyTimes()
			mgr.DenyAgent("testing")
			err := mgr.RegisterAgent(ctx, "testing", agent, conn)
			Expect(err).ToNot(HaveOccurred())
			Expect(mgr.KnownAgents()).To(BeEmpty())
		})

		It("should not subscribe the agent twice", func() {
			conn.EXPECT().AgentBroadcastTarget("cone", "stub").Return("cone.stub")
			conn.EXPECT().AgentBroadcastTarget("ctwo", "stub").Return("ctwo.stub")
			conn.EXPECT().QueueSubscribe(gomock.Any(), "cone.stub", "cone.stub", "", gomock.Any()).Return(nil).Times(1)
			conn.EXPECT().QueueSubscribe(gomock.Any(), "ctwo.stub", "ctwo.stub", "", gomock.Any()).Return(nil).Times(1)

			agent.EXPECT().ShouldActivate().Return(true).AnyTimes()
			err := mgr.RegisterAgent(ctx, "stub", agent, conn)
			Expect(err).ToNot(HaveOccurred())

			err = mgr.RegisterAgent(ctx, "stub", agent, conn)
			Expect(err).To(MatchError("agent stub is already registered"))

		})

		It("should subscribe the agent to all collectives", func() {
			conn.EXPECT().AgentBroadcastTarget("cone", "stub").Return("cone.stub")
			conn.EXPECT().AgentBroadcastTarget("ctwo", "stub").Return("ctwo.stub")
			conn.EXPECT().QueueSubscribe(gomock.Any(), "cone.stub", "cone.stub", "", gomock.Any()).Return(nil).Times(1)
			conn.EXPECT().QueueSubscribe(gomock.Any(), "ctwo.stub", "ctwo.stub", "", gomock.Any()).Return(nil).Times(1)

			agent.EXPECT().ShouldActivate().Return(true).AnyTimes()
			err := mgr.RegisterAgent(ctx, "stub", agent, conn)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should support service agents", func() {
			agent.Metadata().Service = true
			conn.EXPECT().ServiceBroadcastTarget("cone", "stub").Return("cone.stub")
			conn.EXPECT().ServiceBroadcastTarget("ctwo", "stub").Return("ctwo.stub")
			conn.EXPECT().QueueSubscribe(gomock.Any(), "cone.stub", "cone.stub", "stub", gomock.Any()).Return(nil).Times(1)
			conn.EXPECT().QueueSubscribe(gomock.Any(), "ctwo.stub", "ctwo.stub", "stub", gomock.Any()).Return(nil).Times(1)
			agent.EXPECT().ShouldActivate().Return(true).AnyTimes()
			err := mgr.RegisterAgent(ctx, "stub", agent, conn)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should only register service agents in service host mode", func() {
			mgr.servicesOnly = true

			err := mgr.RegisterAgent(ctx, "stub", agent, conn)
			Expect(err).ToNot(HaveOccurred())
			Expect(mgr.agents).To(HaveLen(0))

			agent.Metadata().Service = true
			conn.EXPECT().ServiceBroadcastTarget("cone", "stub").Return("cone.stub")
			conn.EXPECT().ServiceBroadcastTarget("ctwo", "stub").Return("ctwo.stub")
			conn.EXPECT().QueueSubscribe(gomock.Any(), "cone.stub", "cone.stub", "stub", gomock.Any()).Return(nil).Times(1)
			conn.EXPECT().QueueSubscribe(gomock.Any(), "ctwo.stub", "ctwo.stub", "stub", gomock.Any()).Return(nil).Times(1)
			agent.EXPECT().ShouldActivate().Return(true).AnyTimes()
			err = mgr.RegisterAgent(ctx, "stub", agent, conn)
			Expect(err).ToNot(HaveOccurred())
			Expect(mgr.agents).To(HaveLen(1))
		})

		It("should handle subscribe failures", func() {
			agent.EXPECT().ShouldActivate().Return(true).AnyTimes()
			conn.EXPECT().AgentBroadcastTarget("cone", "stub").Return("cone.stub")
			conn.EXPECT().AgentBroadcastTarget("ctwo", "stub").Return("ctwo.stub")
			conn.EXPECT().QueueSubscribe(gomock.Any(), "cone.stub", "cone.stub", "", gomock.Any()).Return(nil).AnyTimes()
			conn.EXPECT().QueueSubscribe(gomock.Any(), "ctwo.stub", "ctwo.stub", "", gomock.Any()).Return(errors.New("2nd sub failed")).AnyTimes()
			conn.EXPECT().Unsubscribe("cone.stub").Return(nil)

			err := mgr.RegisterAgent(ctx, "stub", agent, conn)
			Expect(err).To(MatchError("could not register agent stub: subscription failed: 2nd sub failed"))
		})

		It("Should retrieve the right agent", func() {
			conn.EXPECT().AgentBroadcastTarget("cone", "stub").Return("cone.stub")
			conn.EXPECT().AgentBroadcastTarget("ctwo", "stub").Return("ctwo.stub")
			conn.EXPECT().QueueSubscribe(gomock.Any(), "cone.stub", "cone.stub", "", gomock.Any()).Return(nil).AnyTimes()
			conn.EXPECT().QueueSubscribe(gomock.Any(), "ctwo.stub", "ctwo.stub", "", gomock.Any()).Return(nil).AnyTimes()
			agent.EXPECT().ShouldActivate().Return(true).AnyTimes()

			err := mgr.RegisterAgent(ctx, "stub", agent, conn)
			Expect(err).ToNot(HaveOccurred())

			a, ok := mgr.Get("stub")
			Expect(ok).To(BeTrue())
			Expect(a).To(Equal(agent))
		})
	})

	Describe("KnownAgents", func() {
		It("Should report on all the known agents", func() {
			for _, a := range []string{"stub1", "stub2", "stub3"} {
				conn.EXPECT().AgentBroadcastTarget("cone", a).Return("cone." + a)
				conn.EXPECT().AgentBroadcastTarget("ctwo", a).Return("ctwo." + a)
				conn.EXPECT().QueueSubscribe(gomock.Any(), "cone."+a, "cone."+a, "", gomock.Any()).Return(nil).AnyTimes()
				conn.EXPECT().QueueSubscribe(gomock.Any(), "ctwo."+a, "ctwo."+a, "", gomock.Any()).Return(nil).AnyTimes()
			}

			agent.EXPECT().ShouldActivate().Return(true).AnyTimes()
			err := mgr.RegisterAgent(ctx, "stub1", agent, conn)
			Expect(err).ToNot(HaveOccurred())
			err = mgr.RegisterAgent(ctx, "stub2", agent, conn)
			Expect(err).ToNot(HaveOccurred())
			err = mgr.RegisterAgent(ctx, "stub3", agent, conn)
			Expect(err).ToNot(HaveOccurred())

			Expect(mgr.KnownAgents()).To(Equal([]string{"stub1", "stub2", "stub3"}))
		})
	})

	Describe("Dispatch", func() {
		var request protocol.Request
		var msg *choria.Message
		var err error
		wg := &sync.WaitGroup{}

		BeforeEach(func() {
			fw.Config.Collectives = []string{"mcollective"}
			request, err = mgr.fw.NewRequest(protocol.RequestV1, "stub", "example.net", "choria=rip.mcollecitve", 60, "123", "mcollective")
			Expect(err).ToNot(HaveOccurred())
			request.SetMessage("hello world")

			msg, err = choria.NewMessageFromRequest(request, "choria.reply.to", mgr.fw)
			Expect(err).ToNot(HaveOccurred())
			agent.EXPECT().ShouldActivate().Return(true).AnyTimes()
			conn.EXPECT().AgentBroadcastTarget("mcollective", "stub").Return("mcollective.stub").AnyTimes()
			conn.EXPECT().QueueSubscribe(gomock.Any(), "mcollective.stub", "mcollective.stub", "", gomock.Any()).Return(nil).AnyTimes()
		})

		It("Should handle unknown agents", func() {
			replyc := make(chan *AgentReply, 1)

			wg.Add(1)
			mgr.Dispatch(ctx, wg, replyc, msg, request)

			var reply *AgentReply

			select {
			case reply = <-replyc:
			default:
				reply = nil
			}

			Expect(reply).To(BeNil())
		})

		It("Should handle replies correctly", func() {
			wg.Add(1)

			agent.Metadata().Timeout = 1

			err := mgr.RegisterAgent(ctx, "stub", agent, conn)
			Expect(err).ToNot(HaveOccurred())

			replyc := make(chan *AgentReply, 1)
			mgr.Dispatch(ctx, wg, replyc, msg, request)

			reply := <-replyc

			Expect(reply.Body).To(Equal([]byte("pong hello world")))
		})

		It("Should finish when the context is canceled", func() {
			wg.Add(1)

			agent.Metadata().Timeout = 10

			err := mgr.RegisterAgent(ctx, "stub", agent, conn)
			Expect(err).ToNot(HaveOccurred())

			msg.Payload = "sleep"
			replyc := make(chan *AgentReply, 1)
			go func() {
				defer GinkgoRecover()
				mgr.Dispatch(ctx, wg, replyc, msg, request)
			}()

			cancel()

			reply := <-replyc

			Expect(reply.Error.Error()).To(MatchRegexp("exiting on interrupt"))
		})

		It("Should finish on timeout", func() {
			wg.Add(1)

			agent.Metadata().Timeout = 1

			err := mgr.RegisterAgent(ctx, "stub", agent, conn)
			Expect(err).ToNot(HaveOccurred())

			msg.Payload = "sleep"
			replyc := make(chan *AgentReply, 1)
			go func() {
				defer GinkgoRecover()
				mgr.Dispatch(ctx, wg, replyc, msg, request)
			}()

			reply := <-replyc

			Expect(reply.Error.Error()).To(MatchRegexp("exiting on 1s timeout"))
		})
	})
})
