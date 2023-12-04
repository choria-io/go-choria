// Copyright (c) 2020-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package ruby

import (
	"context"
	"io"
	"path/filepath"
	"runtime"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/providers/agent/mcorpc"
	ddl "github.com/choria-io/go-choria/providers/agent/mcorpc/ddl/agent"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

var _ = Describe("McoRPC/Ruby", func() {
	var (
		mockctl  *gomock.Controller
		agentMgr *MockAgentManager
		cfg      *config.Config
		fw       *choria.Framework
		err      error
		logger   *logrus.Entry
		agent    *mcorpc.Agent
	)

	BeforeEach(func() {
		build.TLS = "false"

		l := logrus.New()
		l.Out = io.Discard
		logger = l.WithFields(logrus.Fields{})

		mockctl = gomock.NewController(GinkgoT())
		agentMgr = NewMockAgentManager(mockctl)

		cfg = config.NewConfigForTests()
		cfg.DisableSecurityProviderVerify = true

		fw, err = choria.NewWithConfig(cfg)
		Expect(err).ToNot(HaveOccurred())

		agentMgr.EXPECT().Choria().Return(fw).AnyTimes()
		agentMgr.EXPECT().Logger().Return(logger).AnyTimes()
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	Describe("rubyAction", func() {
		var req *mcorpc.Request
		var rep *mcorpc.Reply
		var ctx context.Context
		var agent *mcorpc.Agent
		var ci inter.ConnectorInfo

		BeforeEach(func() {
			req = &mcorpc.Request{
				Agent:  "one",
				Action: "status",
			}

			rep = &mcorpc.Reply{}
			ctx = context.Background()

			ddl, err := ddl.New("testdata/lib1/mcollective/agent/one.json")
			Expect(err).ToNot(HaveOccurred())

			agent, err = NewRubyAgent(ddl, agentMgr)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should fail when no shim is configured", func() {
			fw.Config.Choria.RubyAgentShim = ""
			rubyAction(ctx, req, rep, agent, ci)
			Expect(rep.Statuscode).To(Equal(mcorpc.Aborted))
			Expect(rep.Statusmsg).To(Equal("Cannot call Ruby action one#status: Ruby compatibility shim was not configured"))
		})

		It("Should fail when the shim cannot be found", func() {
			fw.Config.Choria.RubyAgentShim = "/nonexisting"
			fw.Config.Choria.RubyAgentConfig = "testdata/shim.cfg"
			rubyAction(ctx, req, rep, agent, ci)
			Expect(rep.Statuscode).To(Equal(mcorpc.Aborted))
			Expect(rep.Statusmsg).To(Equal("Cannot call Ruby action one#status: Ruby compatibility shim was not found in /nonexisting"))
		})

		It("Should fail without a shim config file", func() {
			fw.Config.Choria.RubyAgentShim = "testdata/nonzero_shim.sh"
			fw.Config.Choria.RubyAgentConfig = ""

			rubyAction(ctx, req, rep, agent, ci)
			Expect(rep.Statuscode).To(Equal(mcorpc.Aborted))
			Expect(rep.Statusmsg).To(Equal("Cannot call Ruby action one#status: Ruby compatibility shim configuration file not configured"))
		})

		It("Should fail when a shim config file does not exist", func() {
			fw.Config.Choria.RubyAgentShim = "testdata/nonzero_shim.sh"
			fw.Config.Choria.RubyAgentConfig = "/nonexisting"

			rubyAction(ctx, req, rep, agent, ci)
			Expect(rep.Statuscode).To(Equal(mcorpc.Aborted))
			Expect(rep.Statusmsg).To(Equal("Cannot call Ruby action one#status: Ruby compatibility shim configuration file was not found in /nonexisting"))
		})

		It("Should unmarshal the result", func() {
			if runtime.GOOS == "windows" {
				fw.Config.Choria.RubyAgentShim = filepath.Join("testdata", "good_shim_windows.bat")
			} else {
				fw.Config.Choria.RubyAgentShim = filepath.Join("testdata", "good_shim.sh")
			}

			fw.Config.Choria.RubyAgentConfig = filepath.Join("testdata", "shim.cfg")

			try := 0
			for try < 5 {
				rubyAction(ctx, req, rep, agent, ci)
				if rep.Statuscode == mcorpc.OK {
					break
				}
				try++
			}

			Expect(rep.Statusmsg).To(Equal("OK"))
			Expect(rep.Statuscode).To(Equal(mcorpc.OK))

			d := rep.Data.(map[string]any)
			Expect(d["test"].(string)).To(Equal("ok"))
		})
	})

	Describe("activationCheck", func() {
		var (
			d   *ddl.DDL
			err error
		)

		BeforeEach(func() {
			d, err = ddl.New("testdata/lib1/mcollective/agent/one.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should respect true value activate_agent setting", func() {
			cfg, err := config.NewConfig("testdata/one_agent_enabled.cfg")
			Expect(err).ToNot(HaveOccurred())
			fw.Config = cfg
			Expect(activationCheck(d, agentMgr)()).To(BeTrue())
		})

		It("Should respect false value activate_agent setting", func() {
			cfg, err := config.NewConfig("testdata/one_agent_disabled.cfg")
			Expect(err).ToNot(HaveOccurred())
			fw.Config = cfg
			Expect(activationCheck(d, agentMgr)()).To(BeFalse())
		})

		It("Should default to the correct value", func() {
			cfg, err := config.NewConfig("testdata/shim.cfg")
			Expect(err).ToNot(HaveOccurred())
			fw.Config = cfg
			Expect(activationCheck(d, agentMgr)()).To(Equal(activationDefault))
		})
	})

	Describe("NewRubyAgent", func() {
		It("Should create a shim with all the actions mapped", func() {
			d, err := ddl.New("testdata/lib1/mcollective/agent/one.json")
			Expect(err).ToNot(HaveOccurred())

			agent, err = NewRubyAgent(d, agentMgr)
			Expect(err).ToNot(HaveOccurred())

			Expect(agent.ActionNames()).To(Equal(d.ActionNames()))
		})
	})
})
