// Copyright (c) 2017-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package agents

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/inter"
	imock "github.com/choria-io/go-choria/inter/imocks"
	"github.com/choria-io/go-choria/message"
	"github.com/choria-io/go-choria/protocol"
	v1 "github.com/choria-io/go-choria/protocol/v1"
	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
		conn     *imock.MockConnector
		agent    *MockAgent
		requests chan inter.ConnectorMessage
		ctx      context.Context
		cancel   func()
		fw       *imock.MockFramework
		cfg      *config.Config
		handler  func(ctx context.Context, msg *message.Message, request protocol.Request, ci inter.ConnectorInfo, result chan *AgentReply)
	)

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())
		fw, cfg = imock.NewFrameworkForTests(mockctl, GinkgoWriter, imock.WithCallerID(), imock.LogDiscard())
		cfg.Collectives = []string{"cone", "ctwo"}

		requests = make(chan inter.ConnectorMessage)
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

		handler = func(ctx context.Context, msg *message.Message, request protocol.Request, ci inter.ConnectorInfo, result chan *AgentReply) {
			if bytes.Equal(msg.Payload(), []byte("sleep")) {
				time.Sleep(10 * time.Second)
			}

			reply := &AgentReply{
				Body:    []byte(fmt.Sprintf("pong %s", msg.Payload())),
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

		mgr = New(requests, fw, conn, is, fw.Logger("x"))
		conn = imock.NewMockConnector(mockctl)

		agent = NewMockAgent(mockctl)
		agent.EXPECT().Name().Return(metadata.Name).AnyTimes()
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
		It("Should reject agents with small timeouts", func() {
			agent.Metadata().Timeout = 0
			err := mgr.RegisterAgent(ctx, "testing", agent, conn)
			Expect(err).To(MatchError("invalid agent: timeout < 1"))
		})

		It("Should reject agents without a name", func() {
			agent.Metadata().Name = ""
			err := mgr.RegisterAgent(ctx, "testing", agent, conn)
			Expect(err).To(MatchError("invalid agent: invalid metadata"))
		})

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
			Expect(mgr.agents).To(BeEmpty())

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

	Describe("UnregisterAgent", func() {
		It("Should unsubscribe and unregister the agent", func() {
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

			Expect(mgr.agents).To(HaveKey("stub"))
			Expect(mgr.subs).To(HaveKey("stub"))
			Expect(mgr.subs["stub"]).To(HaveLen(2))

			conn.EXPECT().Unsubscribe("cone.stub").Return(nil)
			conn.EXPECT().Unsubscribe("ctwo.stub").Return(fmt.Errorf("fail"))
			Expect(mgr.UnregisterAgent("stub", conn)).To(Succeed())
			Expect(mgr.agents).ToNot(HaveKey("stub"))
			Expect(mgr.subs).ToNot(HaveKey("stub"))
		})
	})

	Describe("ReplaceAgent", func() {
		var oa *MockAgent
		BeforeEach(func() {
			oa = NewMockAgent(mockctl)
			oa.EXPECT().Metadata().Return(&Metadata{
				Author:      "stub@example.net",
				Description: "Stub Agent",
				License:     "Apache-2.0",
				Name:        "stub_agent",
				Timeout:     10,
				URL:         "https://choria.io/",
				Version:     "0.99.0",
			}).AnyTimes()
		})

		It("Should validate agents", func() {
			agent.Metadata().Timeout = 0
			Expect(mgr.ReplaceAgent("testing", agent)).To(MatchError("invalid agent: timeout < 1"))
		})

		It("Should only replace agents with active agents", func() {
			agent.EXPECT().ShouldActivate().Return(false).Times(1)
			Expect(mgr.ReplaceAgent("testing", agent)).To(MatchError("replacement agent is not activating due to activation checks"))
		})

		It("Should reject replacements of unknown agents", func() {
			agent.EXPECT().ShouldActivate().Return(true).Times(1)
			Expect(mgr.ReplaceAgent("testing", agent)).To(MatchError("agent \"testing\" is not currently known"))
		})

		It("Should not allow the service property to be changed", func() {
			oa.Metadata().Service = true
			mgr.agents["testing"] = oa
			agent.EXPECT().ShouldActivate().Return(true).Times(1)
			Expect(mgr.ReplaceAgent("testing", agent)).To(MatchError("replacement agent cannot change service property"))
		})

		It("Should replace the agent", func() {
			mgr.agents["testing"] = oa
			agent.EXPECT().ShouldActivate().Return(true).Times(1)
			Expect(mgr.agents["testing"]).To(Equal(oa))
			Expect(mgr.ReplaceAgent("testing", agent)).To(Succeed())
			Expect(mgr.agents["testing"]).To(Equal(agent))
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
		var msg inter.Message
		var err error
		wg := &sync.WaitGroup{}

		BeforeEach(func() {
			cfg.Collectives = []string{"cone"}
			request, err = v1.NewRequest("stub", "example.net", "choria=rip.mcollective", 60, "123", "cone")
			Expect(err).ToNot(HaveOccurred())
			request.SetMessage([]byte("hello world"))

			msg, err = message.NewMessageFromRequest(request, "choria.reply.to", mgr.fw)
			Expect(err).ToNot(HaveOccurred())
			agent.EXPECT().ShouldActivate().Return(true).AnyTimes()
			conn.EXPECT().AgentBroadcastTarget("cone", "stub").Return("cone.stub").AnyTimes()
			conn.EXPECT().QueueSubscribe(gomock.Any(), "cone.stub", "cone.stub", "", gomock.Any()).Return(nil).AnyTimes()
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

			msg.SetPayload([]byte("sleep"))
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

			msg.SetPayload([]byte("sleep"))
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
