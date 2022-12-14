// Copyright (c) 2020-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package external

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/providers/agent/mcorpc"
	addl "github.com/choria-io/go-choria/providers/agent/mcorpc/ddl/agent"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/ddl/common"
	"github.com/choria-io/go-choria/server/agents"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Providers/Agent/McoRPC/External")
}

var _ = Describe("McoRPC/External", func() {
	var (
		mockctl  *gomock.Controller
		agentMgr *MockAgentManager
		cfg      *config.Config
		prov     *Provider
		si       *MockServerInfoSource
		err      error
		wd       string
	)

	BeforeEach(func() {
		build.TLS = "false"

		mockctl = gomock.NewController(GinkgoT())
		agentMgr = NewMockAgentManager(mockctl)
		si = NewMockServerInfoSource(mockctl)

		cfg = config.NewConfigForTests()
		cfg.DisableSecurityProviderVerify = true

		wd, err = os.Getwd()
		Expect(err).ToNot(HaveOccurred())

		cfg.Choria.RubyLibdir = []string{filepath.Join(wd, "testdata")}

		fw, err := choria.NewWithConfig(cfg)
		Expect(err).ToNot(HaveOccurred())
		fw.SetLogWriter(GinkgoWriter)

		agentMgr.EXPECT().Choria().Return(fw).AnyTimes()
		agentMgr.EXPECT().Logger().Return(fw.Logger("mgr")).AnyTimes()
		si.EXPECT().Facts().Return(json.RawMessage(`{"ginkgo":true}`)).AnyTimes()

		prov = &Provider{
			cfg:    cfg,
			log:    fw.Logger("ginkgo"),
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
					{Name: "act1"},
					{Name: "act2"},
				},
			}
		})

		It("Should load all the actions", func() {
			agent, err := prov.newExternalAgent(ddl, agentMgr)
			Expect(err).ToNot(HaveOccurred())
			Expect(agent.ActionNames()).To(Equal([]string{"act1", "act2"}))
		})
	})

	Describe("agentPath", func() {
		It("Should support the basic agent path to a single file", func() {
			dir := filepath.Join(wd, "testdata/mcollective/agent/ginkgo.json")
			Expect(prov.agentPath("ginkgo", filepath.Join(wd, "testdata/mcollective/agent/ginkgo.json"))).To(Equal(filepath.Join(filepath.Dir(dir), "ginkgo")))
		})

		It("Should support the basic os and arch aware agent paths", func() {
			td, err := os.MkdirTemp("", "")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(td)

			dir := filepath.Join(td, "na")
			Expect(os.MkdirAll(dir, 0744)).To(Succeed())

			path := prov.agentPath("na", dir)
			expected := filepath.Join(dir, fmt.Sprintf("na-%s_%s", runtime.GOOS, runtime.GOARCH))
			Expect(path).To(Equal(expected))
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
			if runtime.GOOS == "windows" {
				Skip("Windows TODO")
			}

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
					{
						Name: "ping",
						Input: map[string]*common.InputItem{
							"hello": {
								Type:       "string",
								Optional:   false,
								Validation: "shellsafe",
								MaxLength:  0,
							},
						},
						Output: map[string]*common.OutputItem{
							"hello": {
								Type:    "string",
								Default: "default",
							},
							"optional": {
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
			agent.SetServerInfo(si)

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
			if runtime.GOOS == "windows" {
				Skip("Windows TODO")
			}

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
			if runtime.GOOS == "windows" {
				Skip("Windows TODO")
			}

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
			Expect(rep.Data.(map[string]any)["hello"].(string)).To(Equal("world"))
			Expect(rep.Data.(map[string]any)["optional"].(string)).To(Equal("optional default"))
		})
	})
})
