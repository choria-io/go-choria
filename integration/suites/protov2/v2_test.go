// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package protov2

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/client/rpcutilclient"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/integration/testbroker"
	"github.com/choria-io/go-choria/integration/testutil"
	iu "github.com/choria-io/go-choria/internal/util"
	"github.com/choria-io/go-choria/server"
	"github.com/choria-io/tokens"
	"github.com/golang-jwt/jwt/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/sirupsen/logrus"
)

func TestV2Protocol(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Protocol V2")
}

var _ = Describe("Protocol V2", func() {
	var (
		ctx           context.Context
		cancel        context.CancelFunc
		wg            sync.WaitGroup
		logger        *logrus.Logger
		brokerLogBuff *gbytes.Buffer
		signer1Public ed25519.PublicKey
		rootDir       string
		err           error
	)

	BeforeEach(func() {
		rootDir, err = os.MkdirTemp("", "")
		Expect(err).ToNot(HaveOccurred())

		brokerLogBuff, logger = testutil.GbytesLogger(logrus.DebugLevel)
		ctx, cancel = context.WithTimeout(context.Background(), 45*time.Second)
		DeferCleanup(func() {
			cancel()
			Eventually(brokerLogBuff, 5).Should(gbytes.Say("Choria Network Broker shut down"))
			os.RemoveAll(rootDir)
		})

		_, err = testbroker.StartNetworkBrokerWithConfigFile(ctx, &wg, "testdata/broker.conf", logger)
		Expect(err).ToNot(HaveOccurred())
		Eventually(brokerLogBuff, 1).Should(gbytes.Say("Allowing unverified TLS connections for AAA signed clients"))
		Eventually(brokerLogBuff, 1).Should(gbytes.Say("Allowing unverified TLS connections for Provisioner signed servers"))
		Eventually(brokerLogBuff, 1).Should(gbytes.Say("Server is ready"))

		signer1Public, _, err = iu.Ed25519KeyPairFromSeedFile("testdata/signer1.seed")
		Expect(err).ToNot(HaveOccurred())
	})

	createTemp := func(name string, template string, signer string, claimsf func(key ed25519.PublicKey) (jwt.Claims, error)) (td, cfile, tfile, sfile string) {
		td, err := os.MkdirTemp(rootDir, "")
		Expect(err).ToNot(HaveOccurred())

		pubk, _, err := iu.Ed25519KeyPairToFile(filepath.Join(td, "seed"))
		Expect(err).ToNot(HaveOccurred())

		claims, err := claimsf(pubk)
		Expect(err).ToNot(HaveOccurred())

		Expect(tokens.SaveAndSignTokenWithKeyFile(claims, "testdata/signer1.seed", filepath.Join(td, "token.jwt"), 0600)).To(Succeed())

		tfn := filepath.Join(td, "token.jwt")
		sfn := filepath.Join(td, "seed")
		cfg, err := iu.ExecuteTemplateFile(template, map[string]any{
			"name":   name,
			"token":  tfn,
			"seed":   sfn,
			"signer": hex.EncodeToString(signer1Public),
		}, nil)

		Expect(err).ToNot(HaveOccurred())
		tf, err := os.CreateTemp(td, "choria.conf")
		Expect(err).ToNot(HaveOccurred())
		tf.Write(cfg)
		tf.Close()

		return td, tf.Name(), tfn, sfn
	}

	startServerInstance := func(cfgFile string, i int) (*gbytes.Buffer, *server.Instance) {
		logbuff, logger := testutil.GbytesLogger(logrus.DebugLevel)

		name := fmt.Sprintf("srv-%d.example.net", i)
		_, cfile, _, _ := createTemp(name, cfgFile, "testdata/signer1.seed", func(pk ed25519.PublicKey) (jwt.Claims, error) {
			return tokens.NewServerClaims(name, []string{"choria"}, "choria", nil, nil, pk, "ginko", time.Minute)
		})

		srv, err := testutil.StartServerInstance(ctx, &wg, cfile, logger, testutil.ServerWithRPCUtilAgent(), testutil.ServerWithDiscovery())
		Expect(err).ToNot(HaveOccurred())

		Eventually(logbuff).Should(gbytes.Say("Setting JWT authentication with NONCE signatures for NATS connection"))
		Eventually(logbuff).Should(gbytes.Say("Using TLS Configuration from ed25519\\+jwt based security system"))
		Eventually(logbuff).Should(gbytes.Say("Signing nonce using seed file"))
		Eventually(logbuff).Should(gbytes.Say("Connected to nats://localhost:4222"))
		Eventually(logbuff).Should(gbytes.Say("Registering new agent rpcutil of type rpcutil"))

		return logbuff, srv
	}

	createRpcUtilClient := func() (*gbytes.Buffer, *rpcutilclient.RpcutilClient, *config.Config) {
		logBuff, logger := testutil.GbytesLogger(logrus.DebugLevel)

		_, cfile, _, _ := createTemp("localhost", "testdata/client.conf", "testdata/signer1.seed", func(pk ed25519.PublicKey) (jwt.Claims, error) {
			return tokens.NewClientIDClaims("choria=ginkgo", nil, "choria", nil, "", "ginkgo", time.Minute, &tokens.ClientPermissions{FleetManagement: true}, pk)
		})

		cfg, err := config.NewConfig(cfile)
		Expect(err).ToNot(HaveOccurred())

		cfg.CustomLogger = logger

		fw, err := choria.NewWithConfig(cfg)
		Expect(err).ToNot(HaveOccurred())

		client, err := rpcutilclient.New(fw)
		Expect(err).ToNot(HaveOccurred())

		return logBuff, client, cfg
	}

	Describe("Basic Operation", func() {
		It("Should default to the choria collective when no collections are given", func() {
			startServerInstance("testdata/server.conf", 1)
			_, _, cfg := createRpcUtilClient()
			Expect(cfg.Collectives).To(Equal([]string{"choria"}))
		})

		It("Should allow servers and clients to communicate without AAA", func() {
			serverLogbuff, _ := startServerInstance("testdata/server.conf", 1)
			clientLogbuff, client, _ := createRpcUtilClient()

			client.OptionTargets([]string{"srv-1.example.net"})
			res, err := client.Ping().Do(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.Stats().OKCount()).To(Equal(1))

			Eventually(clientLogbuff).Should(gbytes.Say("Setting JWT authentication with NONCE signatures for NATS connection"))
			Eventually(clientLogbuff).Should(gbytes.Say("Signing nonce using seed file"))
			Eventually(serverLogbuff).Should(gbytes.Say("Handling message .+ for rpcutil#ping from choria=ginkgo"))
		})
	})
})
