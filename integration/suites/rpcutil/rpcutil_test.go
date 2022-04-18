// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package rpcutil

import (
	"context"
	"encoding/json"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/client/rpcutilclient"
	"github.com/choria-io/go-choria/config"
	. "github.com/choria-io/go-choria/integration/matchers"
	"github.com/choria-io/go-choria/integration/testbroker"
	"github.com/choria-io/go-choria/integration/testutil"
	"github.com/choria-io/go-choria/providers/agent/mcorpc"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/golang/rpcutil"
	"github.com/choria-io/go-choria/server"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/sirupsen/logrus"
)

func TestRPCUtilAgent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "rpcutil agent")
}

var _ = Describe("rpcutil agent", func() {
	var (
		ctx           context.Context
		cancel        context.CancelFunc
		wg            sync.WaitGroup
		srv           *server.Instance
		rpcutilAgent  *mcorpc.Agent
		rpcutilClient *rpcutilclient.RpcutilClient
		brokerLogger  *logrus.Logger
		brokerLogbuff *gbytes.Buffer
		serverLogbuff *gbytes.Buffer
		err           error
	)

	startServerInstance := func(cfgFile string, logbuff *gbytes.Buffer) (*server.Instance, error) {
		logger := logrus.New()
		logger.SetOutput(logbuff)

		srv, err = testutil.StartServerInstance(ctx, &wg, cfgFile, logger)
		Expect(err).ToNot(HaveOccurred())

		Eventually(logbuff).Should(gbytes.Say("Connected to nats://localhost:4222"))

		return srv, nil
	}

	createRpcUtilClient := func() (*gbytes.Buffer, *rpcutilclient.RpcutilClient, error) {
		logBuff, logger := testutil.GbytesLogger(logrus.DebugLevel)

		cfg, err := config.NewConfig("testdata/client.conf")
		if err != nil {
			return nil, nil, err
		}

		cfg.CustomLogger = logger
		cfg.OverrideCertname = "rip.mcollective"

		fw, err := choria.NewWithConfig(cfg)
		if err != nil {
			return nil, nil, err
		}

		client, err := rpcutilclient.New(fw)
		if err != nil {
			return nil, nil, err
		}

		return logBuff, client, nil
	}

	BeforeEach(func() {
		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		DeferCleanup(func() {
			cancel()
			Eventually(brokerLogbuff, 5).Should(gbytes.Say("Choria Network Broker shut down"))
		})

		brokerLogbuff, brokerLogger = testutil.GbytesLogger(logrus.DebugLevel)
		serverLogbuff = gbytes.NewBuffer()

		_, err := testbroker.StartNetworkBrokerWithConfigFile(ctx, &wg, "testdata/broker.conf", brokerLogger)
		Expect(err).ToNot(HaveOccurred())
		Eventually(brokerLogbuff, 1).Should(gbytes.Say("Server is ready"))

		srv, err = startServerInstance("testdata/server.conf", serverLogbuff)
		Expect(err).ToNot(HaveOccurred())

		rpcutilAgent, err = rpcutil.New(srv.AgentManager())
		Expect(err).ToNot(HaveOccurred())

		err = srv.AgentManager().RegisterAgent(ctx, "rpcutil", rpcutilAgent, srv.Connector())
		Expect(err).ToNot(HaveOccurred())

		Eventually(serverLogbuff).Should(gbytes.Say("Registering new agent rpcutil of type rpcutil"))

		_, rpcutilClient, err = createRpcUtilClient()
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("Agent", func() {
		It("Should create all actions we support", func() {
			Expect(rpcutilAgent.ActionNames()).To(Equal([]string{"agent_inventory", "collective_info", "daemon_stats", "get_config_item", "get_data", "get_fact", "get_facts", "inventory", "ping"}))
		})
	})

	Describe("agent_inventory action", func() {
		It("Should get the right inventory", func() {
			res, err := rpcutilClient.OptionTargets([]string{"localhost"}).AgentInventory().Do(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveOnlySuccessfulResponses())
			r := rpcutil.AgentInventoryReply{}
			Expect(res.AllOutputs()[0].ParseAgentInventoryOutput(&r)).ToNot(HaveOccurred())

			Expect(r.Agents).To(HaveLen(1))
			Expect(r.Agents[0].Agent).To(Equal("rpcutil"))
			Expect(r.Agents[0].Name).To(Equal("rpcutil"))
			Expect(r.Agents[0].Timeout).To(Equal(2))
		})
	})

	Describe("collective_info action", func() {
		It("Should fetch correct collective info", func() {
			res, err := rpcutilClient.OptionTargets([]string{"localhost"}).CollectiveInfo().Do(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveOnlySuccessfulResponses())

			r := res.AllOutputs()[0]
			Expect(r.Collectives()).To(Equal([]interface{}{"mcollective", "other"}))
			Expect(r.MainCollective()).To(Equal("mcollective"))
		})
	})

	Describe("daemon_stats action", func() {
		It("Should fetch correct instance stats", func() {
			res, err := rpcutilClient.OptionTargets([]string{"localhost"}).DaemonStats().Do(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveOnlySuccessfulResponses())

			r := res.AllOutputs()[0]

			Expect(r.Agents()).To(Equal([]interface{}{"rpcutil"}))
			Expect(r.Version()).To(Equal(build.Version))
			path, _ := filepath.Abs("testdata/server.conf")
			Expect(r.Configfile()).To(Equal(path))
		})
	})

	Describe("get_config_item action", func() {
		It("Should fetch correct item", func() {
			res, err := rpcutilClient.OptionTargets([]string{"localhost"}).GetConfigItem("classesfile").Do(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveOnlySuccessfulResponses())

			r := res.AllOutputs()[0]
			Expect(r.Item()).To(Equal("classesfile"))
			Expect(r.Value()).To(Equal("testdata/classes.txt"))
		})
	})

	Describe("get_data action", func() {
		It("Should get the right data", func() {
			res, err := rpcutilClient.OptionTargets([]string{"localhost"}).GetData("choria").Do(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveOnlySuccessfulResponses())

			r := res.AllOutputs()[0].HashMap()

			Expect(r).To(HaveKeyWithValue("classes", []interface{}{"one", "three", "two"}))
			Expect(r).To(HaveKeyWithValue("classes_count", float64(3)))
			Expect(r).To(HaveKeyWithValue("connected_broker", "nats://localhost:4222"))
		})
	})

	Describe("get_fact", func() {
		It("Should get the right value", func() {
			res, err := rpcutilClient.OptionTargets([]string{"localhost"}).GetFact("struct.foo").Do(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveOnlySuccessfulResponses())

			r := res.AllOutputs()[0]
			Expect(r.Fact()).To(Equal("struct.foo"))
			Expect(r.Value()).To(Equal("bar"))
		})
	})

	Describe("get_facts", func() {
		It("Should get the right value", func() {
			res, err := rpcutilClient.OptionTargets([]string{"localhost"}).GetFacts("struct.foo,bool").Do(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveOnlySuccessfulResponses())

			r := res.AllOutputs()[0].Values()
			Expect(r).To(HaveKeyWithValue("struct.foo", interface{}("bar")))
			Expect(r).To(HaveKeyWithValue("bool", interface{}(false)))
		})
	})

	Describe("inventory action", func() {
		It("Should retrieve the correct info", func() {
			res, err := rpcutilClient.OptionTargets([]string{"localhost"}).Inventory().Do(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveOnlySuccessfulResponses())

			r := res.AllOutputs()[0]
			Expect(r.Agents()).To(Equal([]interface{}{"rpcutil"}))
			Expect(r.Classes()).To(Equal([]interface{}{"one", "three", "two"}))
			Expect(r.Collectives()).To(Equal([]interface{}{"mcollective", "other"}))
			Expect(r.MainCollective()).To(Equal("mcollective"))
			Expect(r.DataPlugins()).To(Equal([]interface{}{"choria", "scout"}))
			fj, err := json.Marshal(r.Facts())
			Expect(err).ToNot(HaveOccurred())
			Expect(fj).To(MatchJSON(`{"bool":false,"float":1.1,"int":1,"string":"hello world","struct":{"foo":"bar"}}`))
			Expect(r.Version()).To(Equal(build.Version))
		})
	})

	Describe("ping action", func() {
		It("Should do the correct pong", func() {
			res, err := rpcutilClient.OptionTargets([]string{"localhost"}).Ping().Do(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveOnlySuccessfulResponses())

			r := res.AllOutputs()[0]
			Expect(err).ToNot(HaveOccurred())
			Expect(r.Pong()).To(BeNumerically("==", time.Now().Unix(), 1))
		})
	})
})
