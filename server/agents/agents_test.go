package agents

import (
	"errors"
	"testing"

	"github.com/choria-io/go-choria/choria/connectortest"

	"github.com/choria-io/go-choria/choria"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

type stubAgent struct {
	meta      *Metadata
	nextError string
}

func (s *stubAgent) Metadata() *Metadata {
	return s.meta
}

func (s *stubAgent) Name() string {
	return "stub"
}

func (s *stubAgent) Handle(msg *choria.Message) (*[]byte, error) {
	if s.nextError != "" {
		err := errors.New(s.nextError)
		s.nextError = ""
		return &[]byte{}, err
	}

	r := []byte("pong")
	return &r, nil
}

func TestFileContent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Server/Agents")
}

var _ = Describe("Server/Agents", func() {
	var mgr *Manager
	var conn *connectortest.AgentConnector
	var agent *stubAgent

	BeforeEach(func() {
		fw, err := choria.New("/dev/null")
		Expect(err).ToNot(HaveOccurred())
		fw.Config.Collectives = []string{"cone", "ctwo"}

		logrus.SetLevel(logrus.FatalLevel)
		mgr = New(fw, logrus.WithFields(logrus.Fields{"testing": true}))
		conn = &connectortest.AgentConnector{}
		conn.Init()

		agent = &stubAgent{meta: &Metadata{}}
	})

	It("should not subscribe the agent twice", func() {
		err := mgr.RegisterAgent("stub", agent, conn)
		Expect(err).ToNot(HaveOccurred())

		err = mgr.RegisterAgent("stub", agent, conn)
		Expect(err).To(MatchError("Agent stub is already registered"))

	})

	It("should subscribe the agent to all collectives", func() {
		err := mgr.RegisterAgent("stub", agent, conn)
		Expect(err).ToNot(HaveOccurred())

		Expect(conn.ActibeSubs["cone.stub"]).To(Equal("cone.broadcast.agent.stub"))
		Expect(conn.ActibeSubs["ctwo.stub"]).To(Equal("ctwo.broadcast.agent.stub"))
		Expect(conn.ActibeSubs).To(HaveLen(2))
	})

	It("should handle subscribe failures", func() {
		conn.NextErr = append(conn.NextErr, nil)
		conn.NextErr = append(conn.NextErr, errors.New("2nd sub failed"))

		err := mgr.RegisterAgent("stub", agent, conn)
		Expect(err).To(MatchError("Could not register agent stub: Subscription failed: 2nd sub failed"))

		Expect(conn.Subscribes).To(HaveLen(2))
		Expect(conn.Unsubscribes).To(HaveLen(1))
		Expect(conn.ActibeSubs).To(BeEmpty())
	})

	It("Should retrieve the right agent", func() {
		err := mgr.RegisterAgent("stub", agent, conn)
		Expect(err).ToNot(HaveOccurred())

		a, ok := mgr.Get("stub")
		Expect(ok).To(BeTrue())
		Expect(a).To(Equal(agent))
	})
})
