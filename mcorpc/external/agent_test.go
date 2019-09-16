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
		wd       string
	)

	BeforeEach(func() {
		build.TLS = "false"
		logger = logrus.NewEntry(logrus.New())
		logger.Logger.Out = ioutil.Discard

		mockctl = gomock.NewController(GinkgoT())
		agentMgr = NewMockAgentManager(mockctl)

		cfg = config.NewConfigForTests()
		cfg.DisableSecurityProviderVerify = true

		wd, err = os.Getwd()
		Expect(err).ToNot(HaveOccurred())

		cfg.LibDir = []string{filepath.Join(wd, "testdata")}

		fw, err := choria.NewWithConfig(cfg)
		Expect(err).ToNot(HaveOccurred())

		agentMgr.EXPECT().Choria().Return(fw).AnyTimes()
		agentMgr.EXPECT().Logger().Return(logger).AnyTimes()

		prov = &Provider{
			cfg:    cfg,
			log:    logger,
			agents: []*addl.DDL{},
			paths:  make(map[string]string),
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
				SourceLocation: filepath.Join(wd, "testdata/mcollective/agent/ginkgo.json"),
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
			d := &addl.DDL{
				SourceLocation: filepath.Join(wd, "testdata/mcollective/agent/activation_checker_enabled.json"),
				Metadata:       &agents.Metadata{Name: "activation_checker_fails"},
			}
			c, err := prov.externalActivationCheck(d)
			Expect(err).ToNot(HaveOccurred())
			Expect(c()).To(BeFalse())
		})

		It("should handle specifically disabled agents", func() {
			d := &addl.DDL{
				SourceLocation: filepath.Join(wd, "testdata/mcollective/agent/activation_checker_enabled.json"),
				Metadata:       &agents.Metadata{Name: "activation_checker_disabled"},
			}
			c, err := prov.externalActivationCheck(d)
			Expect(err).ToNot(HaveOccurred())
			Expect(c()).To(BeFalse())
		})

		It("should handle specifically enabled agents", func() {
			d := &addl.DDL{
				SourceLocation: filepath.Join(wd, "testdata/mcollective/agent/activation_checker_enabled.json"),
				Metadata:       &agents.Metadata{Name: "activation_checker_enabled"},
			}
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
				SourceLocation: filepath.Join(wd, "testdata/mcollective/agent/activation_checker_enabled.json"),
				Metadata: &agents.Metadata{
					Name:    "ginkgo",
					Timeout: 1,
				},
				Actions: []*addl.Action{
					&addl.Action{
						Name: "ping",
						Input: map[string]*addl.ActionInputItem{
							"hello": &addl.ActionInputItem{
								Type:       "string",
								Optional:   false,
								Validation: "shellsafe",
								MaxLength:  0,
							},
						},
						Output: map[string]*addl.ActionOutputItem{
							"hello": &addl.ActionOutputItem{
								Type:    "string",
								Default: "default",
							},
							"optional": &addl.ActionOutputItem{
								Type:    "string",
								Default: "optional default",
							},
						},
					},
				},
			}
			prov.agents = append(prov.agents, ddl)
			prov.paths["ginkgo"] = ddl.SourceLocation

			agent, err = prov.newExternalAgent(ddl, agentMgr)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should handle a missing executable", func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			prov.paths["ginkgo_missing"] = ddl.SourceLocation
			ddl.Metadata.Name = "ginkgo_missing"
			rep := &mcorpc.Reply{}
			req := &mcorpc.Request{
				Agent:  "ginkgo_missing",
				Action: "ping",
				Data:   json.RawMessage(`{"hello":"world"}`),
			}

			prov.externalAction(ctx, req, rep, agent, nil)
			Expect(rep.Statusmsg).To(MatchRegexp("Cannot call.+ginkgo_missing#ping.+ginkgo_missing was not found"))
			Expect(rep.Statuscode).To(Equal(mcorpc.Aborted))
		})

		It("Should handle execution failures", func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			prov.paths["ginkgo_abort"] = ddl.SourceLocation
			ddl.Metadata.Name = "ginkgo_abort"
			rep := &mcorpc.Reply{}
			req := &mcorpc.Request{
				Agent:  "ginkgo_abort",
				Action: "ping",
				Data:   json.RawMessage(`{"hello":"world"}`),
			}

			prov.externalAction(ctx, req, rep, agent, nil)
			Expect(rep.Statusmsg).To(MatchRegexp("Could not call.+ginkgo_abort#ping.+exit status 1"))
			Expect(rep.Statuscode).To(Equal(mcorpc.Aborted))
		})

		It("Should validate the input before executing the agent", func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			prov.paths["ginkgo_abort"] = ddl.SourceLocation
			ddl.Metadata.Name = "ginkgo_abort"
			rep := &mcorpc.Reply{}
			req := &mcorpc.Request{
				Agent:  "ginkgo_abort",
				Action: "ping",
				Data:   json.RawMessage(`{"hello":1}`),
			}

			prov.externalAction(ctx, req, rep, agent, nil)
			Expect(rep.Statusmsg).To(MatchRegexp("Validation failed: validation failed for input 'hello': is not a string"))
			Expect(rep.Statuscode).To(Equal(mcorpc.Aborted))
		})

		It("Should execute the correct request binary with the correct input and set defaults on the reply", func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			rep := &mcorpc.Reply{}
			req := &mcorpc.Request{
				Agent:  "ginkgo",
				Action: "ping",
				Data:   json.RawMessage(`{"hello":"world"}`),
			}

			prov.externalAction(ctx, req, rep, agent, nil)
			Expect(rep.Statusmsg).To(Equal("OK"))
			Expect(rep.Statuscode).To(Equal(mcorpc.OK))
			Expect(rep.Data.(map[string]interface{})["hello"].(string)).To(Equal("world"))
			Expect(rep.Data.(map[string]interface{})["optional"].(string)).To(Equal("optional default"))
		})
	})
})
