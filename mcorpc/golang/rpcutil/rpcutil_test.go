package rpcutil

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/server/agents"
	"github.com/choria-io/mcorpc-agent-provider/mcorpc"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"testing"
)

func Test(t *testing.T) {
	os.Setenv("MCOLLECTIVE_CERTNAME", "rip.mcollective")
	RegisterFailHandler(Fail)
	RunSpecs(t, "McoRPC/Golang/RPCUtil")
}

var _ = Describe("McoRPC/Golang/RPCUtil", func() {
	var (
		mockctl  *gomock.Controller
		requests chan *choria.ConnectorMessage
		cfg      *config.Config
		fw       *choria.Framework
		am       *agents.Manager
		err      error
		rpcutil  *mcorpc.Agent
		reply    *mcorpc.Reply
		ctx      context.Context
		is       *agents.MockServerInfoSource
	)

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())

		requests = make(chan *choria.ConnectorMessage)
		reply = &mcorpc.Reply{}

		cfg = config.NewConfigForTests()
		cfg.DisableTLS = true
		cfg.ClassesFile = "/foo/bar"

		fw, err = choria.NewWithConfig(cfg)
		Expect(err).ToNot(HaveOccurred())

		am = agents.New(requests, fw, nil, agents.NewMockServerInfoSource(mockctl), logrus.WithFields(logrus.Fields{"test": "1"}))
		rpcutil, err = New(am)
		Expect(err).ToNot(HaveOccurred())
		logrus.SetLevel(logrus.FatalLevel)

		ctx = context.Background()
		cfg.FactSourceFile = "testdata/facts.yaml"

		metadata := agents.Metadata{
			Author:      "stub@example.net",
			Description: "Stub Agent",
			License:     "Apache-2.0",
			Name:        "stub_agent",
			Timeout:     10,
			URL:         "https://choria.io/",
			Version:     "1.0.0",
		}

		is = agents.NewMockServerInfoSource(mockctl)
		is.EXPECT().KnownAgents().Return([]string{"stub_agent"}).AnyTimes()
		is.EXPECT().Classes().Return([]string{"one", "two"}).AnyTimes()
		is.EXPECT().Facts().Return(json.RawMessage(`{"stub":true}`)).AnyTimes()
		is.EXPECT().AgentMetadata("stub_agent").Return(metadata, true).AnyTimes()
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	var _ = Describe("New", func() {
		It("Should create all actions we support", func() {
			Expect(rpcutil.ActionNames()).To(Equal([]string{"agent_inventory", "collective_info", "daemon_stats", "get_config_item", "get_data", "get_fact", "get_facts", "inventory", "ping"}))
		})
	})

	var _ = Describe("inventoryAction", func() {
		It("Should retrieve the correct info", func() {
			build.Version = "1.0.0"
			cfg.Collectives = []string{"mcollective", "other"}
			cfg.MainCollective = "mcollective"

			rpcutil.SetServerInfo(is)

			inventoryAction(ctx, &mcorpc.Request{}, reply, rpcutil, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.OK))

			r := reply.Data.(*InventoryReply)
			Expect(r.Agents).To(Equal([]string{"stub_agent"}))
			Expect(r.Classes).To(Equal([]string{"one", "two"}))
			Expect(r.Collectives).To(Equal([]string{"mcollective", "other"}))
			Expect(r.MainCollective).To(Equal("mcollective"))
			Expect(r.DataPlugins).To(Equal([]string{}))
			Expect(r.Facts).To(Equal(json.RawMessage(`{"stub":true}`)))
			Expect(r.Version).To(Equal("1.0.0"))
		})
	})

	var _ = Describe("agentInventoryAction", func() {
		It("Should get the right inventory", func() {
			rpcutil.SetServerInfo(is)

			agentInventoryAction(ctx, &mcorpc.Request{}, reply, rpcutil, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.OK))

			r := reply.Data.(*AgentInventoryReply).Agents[0]
			Expect(r.Agent).To(Equal("stub_agent"))
			Expect(r.Name).To(Equal("stub_agent"))
			Expect(r.Timeout).To(Equal(10))
		})
	})

	var _ = Describe("collectiveInfoAction", func() {
		It("Should get the right collective info", func() {
			cfg.MainCollective = "test_collective"
			cfg.Collectives = []string{"test_collective", "other_collective"}

			collectiveInfoAction(ctx, &mcorpc.Request{}, reply, rpcutil, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.OK))
			Expect(reply.Data.(CollectiveInfoReply).Collectives).To(Equal(cfg.Collectives))
			Expect(reply.Data.(CollectiveInfoReply).MainCollective).To(Equal(cfg.MainCollective))
		})
	})

	var _ = Describe("getFactsAction", func() {
		It("Should get the right facts", func() {
			getFactsAction(ctx, &mcorpc.Request{Data: json.RawMessage(`{"facts":"string, int, doesnt_exist"}`)}, reply, rpcutil, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.OK))
			Expect(reply.Data.(*GetFactsReply).Values["string"]).To(Equal("hello world"))
			Expect(reply.Data.(*GetFactsReply).Values["int"]).To(Equal(float64(1)))
			Expect(reply.Data.(*GetFactsReply).Values["doesnt_exist"]).To(BeNil())
		})
	})

	var _ = Describe("getFactAction", func() {
		It("Should get the right fact", func() {
			getFactAction(ctx, &mcorpc.Request{Data: json.RawMessage(`{"fact":"string"}`)}, reply, rpcutil, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.OK))
			Expect(reply.Data.(*GetFactReply).Fact).To(Equal("string"))
			Expect(reply.Data.(*GetFactReply).Value).To(Equal("hello world"))

			getFactAction(ctx, &mcorpc.Request{Data: json.RawMessage(`{"fact":"struct"}`)}, reply, rpcutil, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.OK))
			Expect(reply.Data.(*GetFactReply).Fact).To(Equal("struct"))
			expected := make(map[string]interface{})
			expected["foo"] = "bar"
			Expect(reply.Data.(*GetFactReply).Value).To(Equal(expected))
		})

		It("Should handle missing values", func() {
			getFactAction(ctx, &mcorpc.Request{Data: json.RawMessage(`{"fact":"missing"}`)}, reply, rpcutil, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.OK))
			Expect(reply.Data.(*GetFactReply).Fact).To(Equal("missing"))
			Expect(reply.Data.(*GetFactReply).Value).To(BeNil())
		})
	})

	var _ = Describe("pingAction", func() {
		It("Should pong correctly", func() {
			pingAction(ctx, &mcorpc.Request{}, reply, rpcutil, nil)
			Expect(reply.Statuscode).To(Equal(mcorpc.OK))
			Expect(reply.Data.(PingReply).Pong).To(BeNumerically("==", time.Now().Unix(), 1))
		})
	})
})
