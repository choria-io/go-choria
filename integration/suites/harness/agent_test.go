// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package harness

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/choria-io/go-choria/providers/agent/mcorpc/golang/discovery"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/client/rpcutilclient"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/integration/agentharness"
	"github.com/choria-io/go-choria/integration/testbroker"
	"github.com/choria-io/go-choria/integration/testutil"
	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/internal/fs"
	"github.com/choria-io/go-choria/providers/agent/mcorpc"
	"github.com/choria-io/go-choria/server"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/sirupsen/logrus"
)

func TestAgentHarnessAgent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "agent testing harness")
}

var _ = Describe("testing harness agent", func() {
	var (
		ctx           context.Context
		cancel        context.CancelFunc
		wg            sync.WaitGroup
		srv           *server.Instance
		rpcutilAgent  *agentharness.AgentHarness
		rpcutilClient *rpcutilclient.RpcutilClient
		brokerLogger  *logrus.Logger
		brokerLogbuff *gbytes.Buffer
		serverLogbuff *gbytes.Buffer
		mockCtl       *gomock.Controller
		err           error
	)

	startServerInstance := func(cfgFile string, logbuff *gbytes.Buffer) (*server.Instance, error) {
		logger := logrus.New()
		logger.SetOutput(logbuff)

		srv, err = testutil.StartServerInstance(ctx, &wg, cfgFile, logger)
		Expect(err).ToNot(HaveOccurred())

		Eventually(logbuff).Should(gbytes.Say("Connected to nats://localhost:4222"))

		da, err := discovery.New(srv.AgentManager())
		Expect(err).ToNot(HaveOccurred())

		err = srv.AgentManager().RegisterAgent(ctx, "discovery", da, srv.Connector())
		Expect(err).ToNot(HaveOccurred())
		Eventually(logbuff).Should(gbytes.Say("Registering new agent discovery of type discovery"))

		return srv, nil
	}

	createAgent := func(fw inter.Framework) *mcorpc.Agent {
		ddl, err := fs.FS.ReadFile("ddl/cache/agent/rpcutil.json")
		Expect(err).ToNot(HaveOccurred())

		rpcutilAgent, err = agentharness.NewWithDDLBytes(fw, mockCtl, "rpcutil", ddl)
		Expect(err).ToNot(HaveOccurred())

		a, err := rpcutilAgent.Agent()
		Expect(err).ToNot(HaveOccurred())

		return a
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
		mockCtl = gomock.NewController(GinkgoT())

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

		err = srv.AgentManager().RegisterAgent(ctx, "rpcutil", createAgent(srv.Choria()), srv.Connector())
		Expect(err).ToNot(HaveOccurred())

		Eventually(serverLogbuff).Should(gbytes.Say("Registering new agent rpcutil of type rpcutil"))

		_, rpcutilClient, err = createRpcUtilClient()
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		mockCtl.Finish()
	})

	Describe("action stubbing", func() {
		It("Should do the correct stubs", FlakeAttempts(5), func() {
			rpcutilAgent.Stub("ping", func(_ context.Context, _ *mcorpc.Request, reply *mcorpc.Reply, _ *mcorpc.Agent, _ inter.ConnectorInfo) {
				reply.Data = map[string]interface{}{
					"pong": time.Now().Unix(),
				}
			}).AnyTimes()

			for i := 0; i < 5; i++ {
				res, err := rpcutilClient.OptionFactFilter("hello=world").Ping().Do(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(res.Stats().OKCount()).To(Equal(1))
				r := res.AllOutputs()[0]
				Expect(err).ToNot(HaveOccurred())
				Expect(r.Pong()).To(BeNumerically("==", time.Now().Unix(), 1))
			}

			// checks that the server did indeed sent back 10 - 5+discovery replies ie. round trip calls all happened
			Expect(srv.Stats().Replies).To(Equal(int64(10)))
		})
	})
})
