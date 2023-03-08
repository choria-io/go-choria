// Copyright (c) 2022-2023, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package protov2

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
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
	"github.com/choria-io/go-choria/inter"
	iu "github.com/choria-io/go-choria/internal/util"
	v2 "github.com/choria-io/go-choria/protocol/v2"
	"github.com/choria-io/go-choria/server"
	"github.com/choria-io/tokens"
	"github.com/golang-jwt/jwt/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/sirupsen/logrus"
)

type remoteSignerFunc func(context.Context, []byte, inter.RequestSignerConfig) ([]byte, error)

func TestV2Protocol(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration/Protocol V2")
}

var _ = Describe("Protocol V2", func() {
	var (
		ctx            context.Context
		cancel         context.CancelFunc
		wg             sync.WaitGroup
		logger         *logrus.Logger
		brokerLogBuff  *gbytes.Buffer
		issuerPubK     ed25519.PublicKey
		issuerPubKFile string
		rootDir        string
		err            error
	)

	BeforeEach(func() {
		rootDir, err = os.MkdirTemp("", "")
		Expect(err).ToNot(HaveOccurred())

		issuerPubKFile = filepath.Join(rootDir, "issuer")
		issuerPubK, _, err = iu.Ed25519KeyPairToFile(issuerPubKFile)
		Expect(err).ToNot(HaveOccurred())

		brokerLogBuff, logger = testutil.GbytesLogger(logrus.DebugLevel)
		ctx, cancel = context.WithTimeout(context.Background(), 45*time.Second)
		DeferCleanup(func() {
			cancel()
			Eventually(brokerLogBuff, 5).Should(gbytes.Say("Choria Network Broker shut down"))
			os.RemoveAll(rootDir)
		})

		tokenFile, _, _, priFile, err := testutil.CreateChoriaTokenAndKeys(rootDir, issuerPubKFile, nil, func(pk ed25519.PublicKey) (jwt.Claims, error) {
			return tokens.NewServerClaims("localhost", []string{"choria"}, "choria", nil, nil, pk, "", time.Hour)
		})
		Expect(err).ToNot(HaveOccurred())

		cfg, err := iu.ExecuteTemplateFile("testdata/broker.conf", map[string]any{
			"seed":   priFile,
			"token":  tokenFile,
			"issuer": hex.EncodeToString(issuerPubK),
		}, nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(os.WriteFile(filepath.Join(rootDir, "broker.conf"), cfg, 0644)).To(Succeed())

		_, err = testbroker.StartNetworkBrokerWithConfigFile(ctx, &wg, filepath.Join(rootDir, "broker.conf"), logger)
		Expect(err).ToNot(HaveOccurred())
		Eventually(brokerLogBuff, 1).Should(gbytes.Say("Allowing unverified TLS connections for Organization Issuer issued connections"))
		Eventually(brokerLogBuff, 1).Should(gbytes.Say("Loaded Organization Issuer choria with public key"))
		Eventually(brokerLogBuff, 1).Should(gbytes.Say("Server is ready"))
	})

	// creates a temporary directory and in it seed, public, jwt file and a config file from template
	createTemp := func(name string, template string, issuerSeedFile string, claimsf func(key ed25519.PublicKey) (jwt.Claims, error)) (td string, cfile string, tfile string, sfile string) {
		var err error
		issPubK := issuerPubK

		if issuerSeedFile != "" {
			issPubK, _, err = iu.Ed25519KeyPairFromSeedFile(issuerSeedFile)
			Expect(err).ToNot(HaveOccurred())
		}

		td, err = os.MkdirTemp(rootDir, "")
		Expect(err).ToNot(HaveOccurred())

		tfn, _, _, sfn, err := testutil.CreateChoriaTokenAndKeys(td, issuerSeedFile, nil, claimsf)
		Expect(err).ToNot(HaveOccurred())

		cfg, err := iu.ExecuteTemplateFile(template, map[string]any{
			"name":   name,
			"token":  tfn,
			"seed":   sfn,
			"issuer": hex.EncodeToString(issPubK),
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
		_, cfile, _, _ := createTemp(name, cfgFile, filepath.Join(rootDir, "issuer"), func(pk ed25519.PublicKey) (jwt.Claims, error) {
			return tokens.NewServerClaims(name, []string{"choria"}, "choria", nil, nil, pk, "ginkgo", time.Minute)
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

	createRpcUtilClient := func(perms *tokens.ClientPermissions, signer string, remoteSigner remoteSignerFunc) (*gbytes.Buffer, *rpcutilclient.RpcutilClient, *choria.Framework, *config.Config) {
		logBuff, logger := testutil.GbytesLogger(logrus.DebugLevel)

		if signer == "" {
			signer = filepath.Join(rootDir, "issuer")
		}

		_, cfile, _, _ := createTemp("localhost", "testdata/client.conf", signer, func(pk ed25519.PublicKey) (jwt.Claims, error) {
			return tokens.NewClientIDClaims("choria=ginkgo", nil, "choria", nil, "", "ginkgo", time.Minute, perms, pk)
		})

		cfg, err := config.NewConfig(cfile)
		Expect(err).ToNot(HaveOccurred())

		opts := []choria.Option{}
		if remoteSigner != nil {
			opts = append(opts, choria.WithCustomRequestSigner(testutil.NewFuncSigner(remoteSigner)))
		}
		cfg.CustomLogger = logger

		fw, err := choria.NewWithConfig(cfg, opts...)
		Expect(err).ToNot(HaveOccurred())

		client, err := rpcutilclient.New(fw)
		Expect(err).ToNot(HaveOccurred())

		return logBuff, client, fw, cfg
	}

	aaaSignGen := func(forceInvalid bool) remoteSignerFunc {
		return func(ctx context.Context, req []byte, scfg inter.RequestSignerConfig) ([]byte, error) {
			v2Req, err := v2.NewRequest("", "", "", 0, "", "choria")
			if err != nil {
				return nil, err
			}

			err = json.Unmarshal(req, v2Req)
			if err != nil {
				return nil, err
			}

			signerPub, _, err := iu.Ed25519KeyPairToFile(filepath.Join(rootDir, "fn_signer.seed"))
			Expect(err).ToNot(HaveOccurred())

			signer := filepath.Join(rootDir, "issuer")
			// we support generating failing signatures on purpose
			if forceInvalid {
				signer = filepath.Join(rootDir, "fn_signer.seed")
			}

			// create a new directory with our aaa signer tokens, seed etc with the delegator permission
			_, cfile, tfile, _ := createTemp("localhost", "testdata/client.conf", signer, func(pk ed25519.PublicKey) (jwt.Claims, error) {
				return tokens.NewClientIDClaims("fn_signer", nil, "choria", nil, "", "", time.Hour, &tokens.ClientPermissions{AuthenticationDelegator: true}, signerPub)
			})

			// we now create a config for that delegated signer and make sure we use the right seed since createTemp() will have made one too
			cfg, err := config.NewConfig(cfile)
			Expect(err).ToNot(HaveOccurred())
			cfg.CustomLogger = logger
			cfg.Choria.ChoriaSecuritySeedFile = filepath.Join(rootDir, "fn_signer.seed")

			// signer needs its own security instances
			fw, err := choria.NewWithConfig(cfg)
			Expect(err).ToNot(HaveOccurred())

			// this is what aaa service does
			v2Req.SetCallerID("delegated_client")
			v2SReq, err := fw.NewSecureRequest(context.Background(), v2Req)
			if err != nil {
				return nil, err
			}

			token, err := os.ReadFile(tfile)
			Expect(err).ToNot(HaveOccurred())

			v2SReq.SetSigner(token)

			return v2SReq.JSON()
		}
	}

	Describe("Basic Operation", func() {
		It("Should default to the choria collective when no collections are given", func() {
			startServerInstance("testdata/server.conf", 1)
			_, _, _, cfg := createRpcUtilClient(nil, "", nil)

			// ensures server.conf does not in fact have these settings set
			Expect(cfg.HasOption("collectives")).To(BeFalse())
			Expect(cfg.HasOption("main_collective")).To(BeFalse())

			Expect(cfg.Collectives).To(Equal([]string{"choria"}))
		})

		It("Should fail clients with unknown issuers", func() {
			_, _, err := iu.Ed25519KeyPairToFile(filepath.Join(rootDir, "rogue_issuer"))
			Expect(err).ToNot(HaveOccurred())

			serverLogbuff, _ := startServerInstance("testdata/server.conf", 1)
			clientLogbuff, client, _, _ := createRpcUtilClient(&tokens.ClientPermissions{FleetManagement: true}, filepath.Join(rootDir, "rogue_issuer"), nil)

			client.OptionTargets([]string{"srv-1.example.net"})
			ctx, cancl := context.WithTimeout(ctx, time.Second)
			defer cancl()

			_, err = client.Ping().Do(ctx)
			Expect(err).To(HaveOccurred())
			Eventually(clientLogbuff).Should(gbytes.Say("Setting JWT authentication with NONCE signatures for NATS connection"))
			Eventually(clientLogbuff).Should(gbytes.Say("Signing nonce using seed file"))
			Eventually(clientLogbuff).Should(gbytes.Say("Initial connection to the Broker failed on try 1: nats: Authorization Violation"))
			Expect(serverLogbuff).ShouldNot(gbytes.Say("Handling message .+ for rpcutil#ping from choria=ginkgo"))
		})

		It("Should fail for clients without fleet access", func() {
			serverLogbuff, _ := startServerInstance("testdata/server.conf", 1)

			// first just no fleet management
			clientLogbuff, client, _, _ := createRpcUtilClient(&tokens.ClientPermissions{FleetManagement: false}, "", nil)

			client.OptionTargets([]string{"srv-1.example.net"})
			res, err := client.Ping().Do(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.Stats().OKCount()).To(Equal(0))

			Eventually(clientLogbuff).Should(gbytes.Say("Setting JWT authentication with NONCE signatures for NATS connection"))
			Eventually(clientLogbuff).Should(gbytes.Say("Signing nonce using seed file"))

			// without fleet access one cannot communicate with the signer even so this should fail
			Eventually(brokerLogBuff).Should(gbytes.Say(`Publish Violation.+choria.node.srv-1.example.net`))

			// second we allow it only when signed, and we're not signing here
			clientLogbuff, client, _, _ = createRpcUtilClient(&tokens.ClientPermissions{SignedFleetManagement: true}, "", nil)

			client.OptionTargets([]string{"srv-1.example.net"})
			res, err = client.Ping().Do(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.Stats().OKCount()).To(Equal(0))

			Eventually(clientLogbuff).Should(gbytes.Say("Setting JWT authentication with NONCE signatures for NATS connection"))
			Eventually(clientLogbuff).Should(gbytes.Say("Signing nonce using seed file"))
			Eventually(serverLogbuff).Should(gbytes.Say("access denied: requires authority delegation"))
		})

		It("Should support signed clients", func() {
			serverLogbuff, _ := startServerInstance("testdata/server.conf", 1)

			var forceFail bool

			// we create a client that has a custom remote signer configured, the remote signer does not call any remote AAA server but instead calls a local callback
			// the local callback will do 1 valid request followed by all future ones signed by an invalid issuer.  This should fully allow 1 request as if it was against
			// AAA Server that's correctly configured in the issuer and then just forever fail as being from another issuer
			clientLogbuff, client, _, _ := createRpcUtilClient(&tokens.ClientPermissions{SignedFleetManagement: true}, "", func(ctx context.Context, req []byte, scfg inter.RequestSignerConfig) ([]byte, error) {
				signed, err := aaaSignGen(forceFail)(ctx, req, scfg)
				if err != nil {
					return nil, err
				}
				forceFail = true

				return signed, nil
			})

			client.OptionTargets([]string{"srv-1.example.net"})
			res, err := client.Ping().Do(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.Stats().OKCount()).To(Equal(1))

			Eventually(clientLogbuff).Should(gbytes.Say("Setting JWT authentication with NONCE signatures for NATS connection"))
			Eventually(clientLogbuff).Should(gbytes.Say("Signing nonce using seed file"))

			// we need to make sure it was done for delegated_client which is set by the signer
			Eventually(serverLogbuff).Should(gbytes.Say("Allowing delegator fn_signer to authorize caller delegated_client who holds token choria=ginkgo"))
			Eventually(serverLogbuff).Should(gbytes.Say("Handling message .+ for rpcutil#ping from delegated_client"))

			res, err = client.Ping().Do(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.Stats().OKCount()).To(Equal(0))
			Eventually(serverLogbuff).Should(gbytes.Say("could not parse client token: could not parse client id token: ed25519: verification error"))
			Eventually(serverLogbuff).Should(gbytes.Say("Could not decode incoming request: secure request messages created from Transport Message did not pass security validation"))
		})

		It("Should allow servers and clients to communicate without AAA", func() {
			serverLogbuff, _ := startServerInstance("testdata/server.conf", 1)
			clientLogbuff, client, _, _ := createRpcUtilClient(&tokens.ClientPermissions{FleetManagement: true}, "", nil)

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
