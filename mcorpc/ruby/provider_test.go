package ruby

import (
	"context"
	"io/ioutil"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/mcorpc-agent-provider/mcorpc"

	"github.com/choria-io/go-choria/choria"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

var _ = Describe("McoRPC/Ruby", func() {
	var (
		mockctl   *gomock.Controller
		agentMgr  *MockAgentManager
		connector *choria.MockInstanceConnector
		cfg       *config.Config
		fw        *choria.Framework
		err       error
		logger    *logrus.Entry
		agent     *mcorpc.Agent
	)

	BeforeEach(func() {
		build.TLS = "false"

		l := logrus.New()
		l.Out = ioutil.Discard
		logger = l.WithFields(logrus.Fields{})

		mockctl = gomock.NewController(GinkgoT())
		agentMgr = NewMockAgentManager(mockctl)
		connector = choria.NewMockInstanceConnector(mockctl)

		cfg = config.NewConfigForTests()
		cfg.DisableSecurityProviderVerify = true
		Expect(err).ToNot(HaveOccurred())
		fw, err = choria.NewWithConfig(cfg)
		Expect(err).ToNot(HaveOccurred())

		agentMgr.EXPECT().Choria().Return(fw).AnyTimes()
		agentMgr.EXPECT().Logger().Return(logger).AnyTimes()
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	var _ = Describe("RegisterAgents", func() {
		var p Provider
		var ctx context.Context

		BeforeEach(func() {
			p = Provider{
				fw:  fw,
				cfg: fw.Config,
				log: logger,
			}

			p.loadAgents([]string{"testdata/lib1", "testdata/lib2"})
			Expect(p.Agents()).To(HaveLen(2))

			ctx = context.Background()
		})

		It("Should register all agents", func() {
			agentMgr.EXPECT().RegisterAgent(ctx, "one", gomock.AssignableToTypeOf(agent), connector).Times(1)
			agentMgr.EXPECT().RegisterAgent(ctx, "two", gomock.AssignableToTypeOf(agent), connector).Times(1)

			err := p.RegisterAgents(ctx, agentMgr, connector, logrus.WithFields(logrus.Fields{}))
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
