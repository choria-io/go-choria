package ruby

import (
	"context"
	"io/ioutil"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/mcorpc"
	ddl "github.com/choria-io/go-choria/mcorpc/ddl/agent"
	"github.com/choria-io/go-choria/server"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

var _ = Describe("McoRPC/Ruby", func() {
	var (
		mockctl  *gomock.Controller
		agentMgr *server.MockAgentManager
		fw       *choria.Framework
		err      error
		logger   *logrus.Entry
		agent    *mcorpc.Agent
	)

	BeforeEach(func() {
		build.TLS = "false"

		l := logrus.New()
		l.Out = ioutil.Discard
		logger = l.WithFields(logrus.Fields{})

		mockctl = gomock.NewController(GinkgoT())
		agentMgr = server.NewMockAgentManager(mockctl)

		fw, err = choria.New("/dev/null")
		Expect(err).ToNot(HaveOccurred())

		agentMgr.EXPECT().Choria().Return(fw).AnyTimes()
		agentMgr.EXPECT().Logger().Return(logger).AnyTimes()
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	var _ = Describe("rubyAction", func() {
		var req *mcorpc.Request
		var rep *mcorpc.Reply
		var ctx context.Context
		var agent *mcorpc.Agent
		var ci choria.ConnectorInfo

		BeforeEach(func() {
			req = &mcorpc.Request{
				Agent:  "one",
				Action: "status",
			}

			rep = &mcorpc.Reply{}
			ctx = context.Background()
			choria.NewMockConnectorInfo(mockctl)

			ddl, err := ddl.New("testdata/lib1/mcollective/agent/one.json")
			Expect(err).ToNot(HaveOccurred())

			agent, err = NewRubyAgent(ddl, agentMgr)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should fail when no shim is configured", func() {
			fw.Config.Choria.RubyAgentShim = ""
			rubyAction(ctx, req, rep, agent, ci)
			Expect(rep.Statuscode).To(Equal(mcorpc.Aborted))
			Expect(rep.Statusmsg).To(Equal("Cannot call Ruby action one#status: Ruby compatability shim was not configured"))
		})

		It("Should fail when the shim cannot be found", func() {
			fw.Config.Choria.RubyAgentShim = "/nonexisting"
			fw.Config.Choria.RubyAgentConfig = "testdata/shim.cfg"
			rubyAction(ctx, req, rep, agent, ci)
			Expect(rep.Statuscode).To(Equal(mcorpc.Aborted))
			Expect(rep.Statusmsg).To(Equal("Cannot call Ruby action one#status: Ruby compatability shim was not found in /nonexisting"))
		})

		It("Should fail without a shim config file", func() {
			fw.Config.Choria.RubyAgentShim = "testdata/nonzero_shim.sh"
			fw.Config.Choria.RubyAgentConfig = ""

			rubyAction(ctx, req, rep, agent, ci)
			Expect(rep.Statuscode).To(Equal(mcorpc.Aborted))
			Expect(rep.Statusmsg).To(Equal("Cannot call Ruby action one#status: Ruby compatability shim configuration file not configured"))
		})

		It("Should fail when a shim config file does not exist", func() {
			fw.Config.Choria.RubyAgentShim = "testdata/nonzero_shim.sh"
			fw.Config.Choria.RubyAgentConfig = "/nonexisting"

			rubyAction(ctx, req, rep, agent, ci)
			Expect(rep.Statuscode).To(Equal(mcorpc.Aborted))
			Expect(rep.Statusmsg).To(Equal("Cannot call Ruby action one#status: Ruby compatability shim configuration file was not found in /nonexisting"))
		})

		It("Should unmarshal the result", func() {
			fw.Config.Choria.RubyAgentShim = "testdata/good_shim.sh"
			fw.Config.Choria.RubyAgentConfig = "testdata/shim.cfg"

			rubyAction(ctx, req, rep, agent, ci)

			d := rep.Data.(map[string]interface{})

			Expect(rep.Statusmsg).To(Equal("OK"))
			Expect(rep.Statuscode).To(Equal(mcorpc.OK))
			Expect(d["test"].(string)).To(Equal("ok"))
		})
	})

	var _ = Describe("NewRubyAgent", func() {
		It("Should create a shim with all the actions mapped", func() {
			d, err := ddl.New("testdata/lib1/mcollective/agent/one.json")
			Expect(err).ToNot(HaveOccurred())

			agent, err = NewRubyAgent(d, agentMgr)
			Expect(err).ToNot(HaveOccurred())

			Expect(agent.ActionNames()).To(Equal(d.ActionNames()))
		})
	})
})
