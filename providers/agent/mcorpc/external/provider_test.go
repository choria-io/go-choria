// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package external

import (
	"bytes"
	"context"
	"github.com/choria-io/go-choria/server/agents"
	"github.com/onsi/gomega/gbytes"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"path/filepath"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/config"
	imock "github.com/choria-io/go-choria/inter/imocks"
	addl "github.com/choria-io/go-choria/providers/agent/mcorpc/ddl/agent"
	"github.com/choria-io/go-choria/server"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("McoRPC/External", func() {
	var (
		mockctl *gomock.Controller
		fw      *imock.MockFramework
		mgr     *MockAgentManager
		conn    *imock.MockConnector
		cfg     *config.Config
		prov    *Provider
		ctx     context.Context
		cancel  context.CancelFunc
	)

	BeforeEach(func() {
		build.TLS = "false"

		mockctl = gomock.NewController(GinkgoT())

		buf := gbytes.NewBuffer()
		fw, cfg = imock.NewFrameworkForTests(mockctl, buf)
		fw.EXPECT().Configuration().Return(cfg).AnyTimes()

		conn = imock.NewMockConnector(mockctl)

		mgr = NewMockAgentManager(mockctl)
		mgr.EXPECT().Choria().Return(fw).AnyTimes()
		mgr.EXPECT().Logger().Return(fw.Logger("ginkgo")).AnyTimes()

		lib, err := filepath.Abs("testdata")
		Expect(err).ToNot(HaveOccurred())

		cfg.Choria.RubyLibdir = []string{lib}

		prov = &Provider{
			cfg:    cfg,
			log:    fw.Logger("x"),
			agents: []*addl.DDL{},
			paths:  make(map[string]string),
		}
		prov.log.Logger.SetLevel(logrus.DebugLevel)

		ctx, cancel = context.WithCancel(context.Background())

		DeferCleanup(func() {
			cancel()
			mockctl.Finish()
		})
	})

	Describe("reconcileAgents", func() {
		var td string
		var agentDir string
		var err error

		BeforeEach(func() {
			td, err = os.MkdirTemp("", "")
			Expect(err).ToNot(HaveOccurred())

			agentDir = filepath.Join(td, "mcollective", "agent")

			Expect(os.MkdirAll(agentDir, 0700)).To(Succeed())

			DeferCleanup(func() { os.RemoveAll(td) })
		})

		copyAgentFile := func(f string) {
			src, err := os.Open(filepath.Join("testdata", "mcollective", "agent", f))
			Expect(err).ToNot(HaveOccurred())
			defer src.Close()

			dst, err := os.Create(filepath.Join(agentDir, f))
			Expect(err).ToNot(HaveOccurred())
			defer dst.Close()

			_, err = io.Copy(dst, src)
			Expect(err).ToNot(HaveOccurred())
		}

		It("Should register new agents", func() {
			mgr.EXPECT().RegisterAgent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(3)
			Expect(prov.agents).To(HaveLen(0))
			Expect(prov.reconcileAgents(ctx, mgr, conn)).To(Succeed())
			Expect(prov.agents).To(HaveLen(3))
		})

		It("Should upgrade changed agents", func() {
			fileChangeGrace = 0

			copyAgentFile("one.json")
			prov.cfg.Choria.RubyLibdir = []string{td}
			mgr.EXPECT().RegisterAgent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

			// load what we have
			Expect(prov.agents).To(HaveLen(0))
			Expect(prov.reconcileAgents(ctx, mgr, conn)).To(Succeed())
			Expect(prov.agents).To(HaveLen(1))
			Expect(prov.paths).To(HaveLen(1))

			ddlb, err := os.ReadFile(filepath.Join(agentDir, "one.json"))
			Expect(err).ToNot(HaveOccurred())
			ddlb = bytes.Replace(ddlb, []byte(`"version": "5.0.0"`), []byte(`"version": "6.0.0"`), 1)
			Expect(os.WriteFile(filepath.Join(agentDir, "one.json"), ddlb, 0700)).To(Succeed())

			mgr.EXPECT().ReplaceAgent("one", gomock.Any()).DoAndReturn(func(name string, agent agents.Agent) error {
				Expect(agent.Metadata().Version).To(Equal("6.0.0"))

				return nil
			}).Times(1)

			Expect(prov.reconcileAgents(ctx, mgr, conn)).To(Succeed())
			Expect(prov.agents).To(HaveLen(1))
			Expect(prov.paths).To(HaveLen(1))
			Expect(prov.agents[0].Metadata.Version).To(Equal("6.0.0"))
		})

		It("Should remove orphaned agents", func() {
			copyAgentFile("one.json")
			copyAgentFile("go_agent.json")
			prov.cfg.Choria.RubyLibdir = []string{td}
			mgr.EXPECT().RegisterAgent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(2)

			Expect(prov.reconcileAgents(ctx, mgr, conn)).To(Succeed())
			Expect(prov.agents).To(HaveLen(2))

			Expect(os.Remove(filepath.Join(agentDir, "one.json"))).To(Succeed())

			mgr.EXPECT().UnregisterAgent("one", conn).Times(1)
			Expect(prov.reconcileAgents(ctx, mgr, conn)).To(Succeed())
			Expect(prov.agents).To(HaveLen(1))
			Expect(prov.agents[0].Metadata.Name).To(Equal("echo"))
		})

		It("Should work in sequence", func() {
			fileChangeGrace = 0
			prov.cfg.Choria.RubyLibdir = []string{td}

			mgr.EXPECT().RegisterAgent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(3)
			mgr.EXPECT().UnregisterAgent(gomock.Any(), conn).Times(2)
			mgr.EXPECT().ReplaceAgent(gomock.Any(), gomock.Any()).Times(1)

			Expect(prov.reconcileAgents(ctx, mgr, conn)).To(Succeed())
			Expect(prov.agents).To(HaveLen(0))

			copyAgentFile("one.json")
			Expect(prov.reconcileAgents(ctx, mgr, conn)).To(Succeed())
			Expect(prov.agents).To(HaveLen(1))
			Expect(prov.agents[0].Metadata.Name).To(Equal("one"))

			copyAgentFile("go_agent.json")
			Expect(prov.reconcileAgents(ctx, mgr, conn)).To(Succeed())
			Expect(prov.agents).To(HaveLen(2))
			Expect(prov.agents[0].Metadata.Name).To(Equal("one"))
			Expect(prov.agents[1].Metadata.Name).To(Equal("echo"))

			Expect(os.Remove(filepath.Join(agentDir, "one.json"))).To(Succeed())
			Expect(prov.reconcileAgents(ctx, mgr, conn)).To(Succeed())
			Expect(prov.agents).To(HaveLen(1))
			Expect(prov.agents[0].Metadata.Name).To(Equal("echo"))

			ddlb, err := os.ReadFile(filepath.Join(agentDir, "go_agent.json"))
			Expect(err).ToNot(HaveOccurred())
			ddlb = bytes.Replace(ddlb, []byte(`"version": "1.0.0"`), []byte(`"version": "6.0.0"`), 1)
			Expect(os.WriteFile(filepath.Join(agentDir, "go_agent.json"), ddlb, 0700)).To(Succeed())
			Expect(prov.reconcileAgents(ctx, mgr, conn)).To(Succeed())
			Expect(prov.agents).To(HaveLen(1))
			Expect(prov.agents[0].Metadata.Name).To(Equal("echo"))
			Expect(prov.agents[0].Metadata.Version).To(Equal("6.0.0"))

			copyAgentFile("one.json")
			Expect(prov.reconcileAgents(ctx, mgr, conn)).To(Succeed())
			Expect(prov.agents).To(HaveLen(2))
			Expect(prov.agents[0].Metadata.Name).To(Equal("echo"))
			Expect(prov.agents[1].Metadata.Name).To(Equal("one"))

			Expect(os.Remove(filepath.Join(agentDir, "go_agent.json"))).To(Succeed())
			Expect(prov.reconcileAgents(ctx, mgr, conn)).To(Succeed())
			Expect(prov.agents).To(HaveLen(1))
			Expect(prov.agents[0].Metadata.Name).To(Equal("one"))
		})
	})

	Describe("Agents", func() {
		It("Should return all the agent ddls", func() {
			mgr.EXPECT().RegisterAgent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			Expect(prov.reconcileAgents(ctx, mgr, conn)).To(Succeed())

			agents := prov.Agents()
			Expect(prov.agents).To(HaveLen(3))
			Expect(prov.agents[0].Metadata.Name).To(Equal("echo"))
			Expect(prov.agents[1].Metadata.Name).To(Equal("one"))
			Expect(prov.agents[2].Metadata.Name).To(Equal("three"))
			Expect(agents).To(Equal(prov.agents))

			Expect(prov.paths).To(HaveLen(3))
		})
	})

	Describe("shouldLoadAgent", func() {
		It("Should correctly allow or deny agents", func() {
			Expect(shouldLoadAgent("choria_util")).To(BeFalse())
			Expect(shouldLoadAgent("ginkgo")).To(BeTrue())
		})
	})

	Describe("Plugin", func() {
		It("Should be a valid AgentProvider", func() {
			p := server.AgentProvider(prov)
			Expect(p).ToNot(BeNil())
		})
	})
})
