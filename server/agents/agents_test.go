package agents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/choria-io/go-choria/choria/connectortest"
	"github.com/choria-io/go-protocol/protocol"

	"github.com/choria-io/go-choria/choria"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

type stubAgent struct {
	meta      *Metadata
	nextError string
	si        ServerInfoSource
}

func (s *stubAgent) Metadata() *Metadata {
	return s.meta
}

func (s *stubAgent) Name() string {
	return "stub"
}

func (s *stubAgent) HandleMessage(ctx context.Context, msg *choria.Message, request protocol.Request, ci choria.ConnectorInfo, result chan *AgentReply) {
	if msg.Payload == "sleep" {
		time.Sleep(10 * time.Second)
	}

	reply := &AgentReply{
		Body:    []byte(fmt.Sprintf("pong %s", msg.Payload)),
		Message: msg,
		Request: request,
	}

	if s.nextError != "" {
		reply.Error = fmt.Errorf(s.nextError)
	}

	result <- reply
}

func (s *stubAgent) SetServerInfo(si ServerInfoSource) {
	s.si = si
}

type stubsi struct{}

func (si *stubsi) KnownAgents() []string {
	return []string{"stub_agent"}
}

func (si *stubsi) AgentMetadata(a string) (Metadata, bool) {
	return Metadata{}, true
}

func (si *stubsi) ConfigFile() string {
	return "/stub/config.cfg"
}

func (si *stubsi) Classes() []string {
	return []string{"one", "two"}
}

func (si *stubsi) Facts() json.RawMessage {
	return json.RawMessage(`{"stub":true}`)
}

func (si *stubsi) StartTime() time.Time {
	return time.Now()
}

func (si *stubsi) Stats() ServerStats {
	return ServerStats{}
}

func TestFileContent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Server/Agents")
}

var _ = Describe("Server/Agents", func() {
	var mgr *Manager
	var conn *connectortest.AgentConnector
	var agent *stubAgent
	var requests chan *choria.ConnectorMessage
	var ctx context.Context
	var cancel func()
	var fw *choria.Framework

	BeforeEach(func() {
		cfg, err := choria.NewConfig("/dev/null")
		Expect(err).ToNot(HaveOccurred())

		cfg.DisableTLS = true

		fw, err = choria.NewWithConfig(cfg)
		Expect(err).ToNot(HaveOccurred())
		fw.Config.Collectives = []string{"cone", "ctwo"}

		requests = make(chan *choria.ConnectorMessage)
		ctx, cancel = context.WithCancel(context.Background())

		logrus.SetLevel(logrus.FatalLevel)
		mgr = New(requests, fw, conn, &stubsi{}, logrus.WithFields(logrus.Fields{"testing": true}))
		conn = &connectortest.AgentConnector{}
		conn.Init()

		agent = &stubAgent{meta: &Metadata{}}
	})

	AfterEach(func() {
		cancel()
	})

	var _ = Describe("RegisterAgent", func() {
		It("should not subscribe the agent twice", func() {
			err := mgr.RegisterAgent(ctx, "stub", agent, conn)
			Expect(err).ToNot(HaveOccurred())

			err = mgr.RegisterAgent(ctx, "stub", agent, conn)
			Expect(err).To(MatchError("Agent stub is already registered"))

		})

		It("should subscribe the agent to all collectives", func() {
			err := mgr.RegisterAgent(ctx, "stub", agent, conn)
			Expect(err).ToNot(HaveOccurred())

			Expect(conn.ActiveSubs["cone.stub"]).To(Equal("cone.broadcast.agent.stub"))
			Expect(conn.ActiveSubs["ctwo.stub"]).To(Equal("ctwo.broadcast.agent.stub"))
			Expect(conn.ActiveSubs).To(HaveLen(2))
		})

		It("should handle subscribe failures", func() {
			conn.NextErr = append(conn.NextErr, nil)
			conn.NextErr = append(conn.NextErr, errors.New("2nd sub failed"))

			err := mgr.RegisterAgent(ctx, "stub", agent, conn)
			Expect(err).To(MatchError("Could not register agent stub: Subscription failed: 2nd sub failed"))

			Expect(conn.Subscribes).To(HaveLen(2))
			Expect(conn.Unsubscribes).To(HaveLen(1))
			Expect(conn.ActiveSubs).To(BeEmpty())
		})

		It("Should retrieve the right agent", func() {
			err := mgr.RegisterAgent(ctx, "stub", agent, conn)
			Expect(err).ToNot(HaveOccurred())

			a, ok := mgr.Get("stub")
			Expect(ok).To(BeTrue())
			Expect(a).To(Equal(agent))
		})
	})

	var _ = Describe("KnownAgents", func() {
		It("Should report on all the known agnets", func() {
			err := mgr.RegisterAgent(ctx, "stub1", agent, conn)
			Expect(err).ToNot(HaveOccurred())
			err = mgr.RegisterAgent(ctx, "stub2", agent, conn)
			Expect(err).ToNot(HaveOccurred())
			err = mgr.RegisterAgent(ctx, "stub3", agent, conn)
			Expect(err).ToNot(HaveOccurred())

			Expect(mgr.KnownAgents()).To(Equal([]string{"stub1", "stub2", "stub3"}))
		})
	})

	var _ = Describe("Dispatch", func() {
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

		It("Should finish when the context is cancelled", func() {
			wg.Add(1)

			agent.Metadata().Timeout = 10

			err := mgr.RegisterAgent(ctx, "stub", agent, conn)
			Expect(err).ToNot(HaveOccurred())

			msg.Payload = "sleep"
			replyc := make(chan *AgentReply, 1)
			go mgr.Dispatch(ctx, wg, replyc, msg, request)
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
			go mgr.Dispatch(ctx, wg, replyc, msg, request)

			reply := <-replyc

			Expect(reply.Error.Error()).To(MatchRegexp("exiting on 1s timeout"))
		})

	})
})
