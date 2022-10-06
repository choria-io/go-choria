// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package leafnodes

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/providers/discovery/broadcast"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/integration/testbroker"
	"github.com/choria-io/go-choria/integration/testutil"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/sirupsen/logrus"
)

// TestBrokerLeafnode tests leafnode connections especially as relates to new auth behavior
func TestBrokerLeafnode(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Leafnodes")
}

var _ = Describe("Leafnodes", func() {
	var (
		ctx     context.Context
		cancel  context.CancelFunc
		wg      sync.WaitGroup
		logger  *logrus.Logger
		logbuff *gbytes.Buffer
	)

	BeforeEach(func() {
		ctx, cancel = context.WithTimeout(context.Background(), 45*time.Second)
		DeferCleanup(func() {
			cancel()
			Eventually(logbuff, 5).Should(gbytes.Say("Choria Network Broker shut down"))
		})

		logbuff, logger = testutil.GbytesLogger(logrus.DebugLevel)
	})

	Describe("Basic Leafnode connection in mTLS mode", func() {
		It("Should connect to a remote server", func() {
			// start a core broker listening for leafnode connections
			_, err := testbroker.StartNetworkBrokerWithConfigFile(ctx, &wg, "testdata/core.conf", logger)
			Expect(err).ToNot(HaveOccurred())
			Eventually(logbuff, 1).Should(gbytes.Say("Starting Broker Leafnode support listening on :::7422"))

			// start a choria server against the core broker with discovery agent
			sbuff, slog := testutil.GbytesLogger(logrus.DebugLevel)
			_, err = testutil.StartServerInstance(ctx, &wg, "testdata/core_server.conf", slog, testutil.ServerWithDiscovery())
			Expect(err).ToNot(HaveOccurred())
			Eventually(sbuff).Should(gbytes.Say("Connected to nats://localhost:4222"))
			Eventually(sbuff).Should(gbytes.Say("Registering new agent discovery of type discovery"))

			// start a leafnode broker connecting to core
			lbuff, llog := testutil.GbytesLogger(logrus.DebugLevel)
			_, err = testbroker.StartNetworkBrokerWithConfigFile(ctx, &wg, "testdata/leaf.conf", llog)
			Expect(err).ToNot(HaveOccurred())
			Eventually(lbuff, 1).Should(gbytes.Say("Starting Broker Leafnode support with 1 remote"))
			Eventually(lbuff, 1).Should(gbytes.Say("Leafnode connection created for account: choria"))

			// create a choria client against the leaf
			cfg, err := config.NewConfig("testdata/leaf_client.conf")
			Expect(err).ToNot(HaveOccurred())
			cfg.CustomLogger = llog
			cfg.OverrideCertname = "rip.mcollective"
			lfw, err := choria.NewWithConfig(cfg)
			Expect(err).ToNot(HaveOccurred())

			// discover the server running in the core
			res, err := broadcast.New(lfw).Discover(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(Equal([]string{"localhost"}))
		})
	})
})
