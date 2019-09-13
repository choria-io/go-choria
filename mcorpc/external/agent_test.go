package external

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/choria"
	agents "github.com/choria-io/go-choria/server/agents"
	"github.com/choria-io/go-config"
	"github.com/choria-io/mcorpc-agent-provider/mcorpc"
	addl "github.com/choria-io/mcorpc-agent-provider/mcorpc/ddl/agent"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	logrus "github.com/sirupsen/logrus"
)

func Test(t *testing.T) {
	os.Setenv("MCOLLECTIVE_CERTNAME", "rip.mcollective")
	RegisterFailHandler(Fail)
	RunSpecs(t, "McoRPC/External")
}

var _ = Describe("McoRPC/External", func() {
	var (
		mockctl  *gomock.Controller
		agentMgr *MockAgentManager
		cfg      *config.Config
		logger   *logrus.Entry
		prov     *Provider
		err      error
	)

	BeforeEach(func() {
		build.TLS = "false"
		logger = logrus.NewEntry(logrus.New())
		logger.Logger.Out = ioutil.Discard

		mockctl = gomock.NewController(GinkgoT())
		agentMgr = NewMockAgentManager(mockctl)

		cfg = config.NewConfigForTests()
		cfg.DisableSecurityProviderVerify = true

		fw, err := choria.NewWithConfig(cfg)
		Expect(err).ToNot(HaveOccurred())

		agentMgr.EXPECT().Choria().Return(fw).AnyTimes()
		agentMgr.EXPECT().Logger().Return(logger).AnyTimes()

		path, err := filepath.Abs("testdata/external")
		Expect(err).ToNot(HaveOccurred())

		prov = &Provider{
			cfg:    cfg,
			log:    logger,
			dir:    path,
			agents: []*addl.DDL{},
		}
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	Describe("newExternalAgent", func() {
		var (
			ddl *addl.DDL
		)

		BeforeEach(func() {
			ddl = &addl.DDL{
				Metadata: &agents.Metadata{
					Name:    "ginkgo",
					Timeout: 1,
				},
				Actions: []*addl.Action{
					&addl.Action{Name: "act1"},
					&addl.Action{Name: "act2"},
				},
			}
		})

		It("Should load all the actions", func() {
			agnt, err := prov.newExternalAgent(ddl, agentMgr)
			Expect(err).ToNot(HaveOccurred())
			Expect(agnt.ActionNames()).To(Equal([]string{"act1", "act2"}))
		})

		It("Should set the correct activation checker", func() {

		})
	})

	Describe("externalActivationCheck", func() {
		It("should handle non 0 exit code checks", func() {
			d := &addl.DDL{Metadata: &agents.Metadata{Name: "activation_checker_fails"}}
			c, err := prov.externalActivationCheck(d)
			Expect(err).ToNot(HaveOccurred())
			Expect(c()).To(BeFalse())
		})

		It("should handle specifically disabled agents", func() {
			d := &addl.DDL{Metadata: &agents.Metadata{Name: "activation_checker_disabled"}}
			c, err := prov.externalActivationCheck(d)
			Expect(err).ToNot(HaveOccurred())
			Expect(c()).To(BeFalse())
		})

		It("should handle specifically enabled agents", func() {
			d := &addl.DDL{Metadata: &agents.Metadata{Name: "activation_checker_enabled"}}
			c, err := prov.externalActivationCheck(d)
			Expect(err).ToNot(HaveOccurred())
			Expect(c()).To(BeTrue())
		})
	})

	Describe("externalAction", func() {
		var (
			ddl   *addl.DDL
			agent *mcorpc.Agent
		)

		BeforeEach(func() {
			ddl = &addl.DDL{
				Metadata: &agents.Metadata{
					Name:    "ginkgo",
					Timeout: 1,
				},
			}
			agent, err = prov.newExternalAgent(ddl, agentMgr)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should handle missing executables", func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			rep := &mcorpc.Reply{}
			req := &mcorpc.Request{
				Agent:  "ginkgo_missing",
				Action: "ping",
				Data:   json.RawMessage(`{"hello":"world"}`),
			}

			prov.externalAction(ctx, req, rep, agent, nil)
			Expect(rep.Statuscode).To(Equal(mcorpc.Aborted))
			Expect(rep.Statusmsg).To(MatchRegexp("Cannot call.+ginkgo_missing#ping.+ginkgo_missing was not found"))
		})

		It("Should execute the correct request binary with the correct input", func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			rep := &mcorpc.Reply{}
			req := &mcorpc.Request{
				Agent:  "ginkgo",
				Action: "ping",
				Data:   json.RawMessage(`{"hello":"world"}`),
			}

			prov.externalAction(ctx, req, rep, agent, nil)
			Expect(rep.Statuscode).To(Equal(mcorpc.OK))
			Expect(rep.Statusmsg).To(Equal("OK"))
			Expect(rep.Data.(map[string]interface{})["hello"].(string)).To(Equal("world"))
		})

		It("Should handle execution failures", func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			rep := &mcorpc.Reply{}
			req := &mcorpc.Request{
				Agent:  "ginkgo_abort",
				Action: "ping",
			}

			prov.externalAction(ctx, req, rep, agent, nil)
			Expect(rep.Statuscode).To(Equal(mcorpc.Aborted))
			Expect(rep.Statusmsg).To(MatchRegexp("Could not call.+ginkgo_abort#ping.+exit status 1"))
		})
	})
})
