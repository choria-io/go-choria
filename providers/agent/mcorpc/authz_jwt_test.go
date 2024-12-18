// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package mcorpc

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"os"
	"time"

	imock "github.com/choria-io/go-choria/inter/imocks"
	iu "github.com/choria-io/go-choria/internal/util"
	"github.com/choria-io/go-choria/protocol"
	"github.com/choria-io/go-choria/server/agents"
	"github.com/choria-io/tokens"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"
)

var _ = Describe("McoRPC/JWTAuthorizer", func() {
	var log *logrus.Entry
	var req *Request
	var claims *tokens.ClientIDClaims

	readFixture := func(f string) string {
		c, err := os.ReadFile(f)
		if err != nil {
			panic(err)
		}

		return string(c)
	}

	BeforeEach(func() {
		logger := logrus.New()
		logger.Out = GinkgoWriter
		logger.Level = logrus.DebugLevel
		log = logrus.NewEntry(logger)
		claims = &tokens.ClientIDClaims{}

		req = &Request{
			Agent:      "myco",
			Action:     "deploy",
			Data:       json.RawMessage(`{"component":"frontend"}`),
			SenderID:   "some.node",
			Collective: "ginkgo",
			TTL:        60,
			Time:       time.Now(),
			Filter:     protocol.NewFilter(),
		}
	})

	Describe("aaasvcPolicyAuthorize", func() {
		var agent *Agent
		var logBuff *gbytes.Buffer
		var pubk ed25519.PublicKey
		var prik ed25519.PrivateKey
		var err error

		BeforeEach(func() {
			logBuff = gbytes.NewBuffer()
			mockctl := gomock.NewController(GinkgoT())
			DeferCleanup(func() {
				mockctl.Finish()
			})

			fw, cfg := imock.NewFrameworkForTests(mockctl, logBuff)
			log = fw.Logger("ginkgo")
			log.Logger.SetLevel(logrus.DebugLevel)

			agent = &Agent{
				meta:   &agents.Metadata{Name: "myco"},
				Log:    log,
				Config: cfg,
				Choria: fw,
			}

			pubk, prik, err = iu.Ed25519KeyPair()
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should fail for no caller public data", func() {
			allowed, err := aaasvcPolicyAuthorize(req, agent, log)
			Expect(err).To(MatchError("no policy received in request"))
			Expect(allowed).To(BeFalse())
		})

		It("Should handle invalid tokens", func() {
			req.CallerPublicData = "blah"
			allowed, err := aaasvcPolicyAuthorize(req, agent, log)
			Expect(err).To(MatchError("invalid token in request: token contains an invalid number of segments"))
			Expect(allowed).To(BeFalse())
		})

		It("Should allow discovery agent", func() {
			claims, err = tokens.NewClientIDClaims("ginkgo", nil, "choria", nil, "", "", time.Hour, nil, pubk)
			Expect(err).ToNot(HaveOccurred())
			req.CallerPublicData, err = tokens.SignToken(claims, prik)
			Expect(err).ToNot(HaveOccurred())

			req.Agent = "discovery"
			allowed, err := aaasvcPolicyAuthorize(req, agent, log)
			Expect(err).ToNot(HaveOccurred())
			Expect(allowed).To(BeTrue())
			Expect(logBuff).To(gbytes.Say("Allowing discovery request"))
		})

		It("Should require a policy", func() {
			claims, err = tokens.NewClientIDClaims("ginkgo", nil, "choria", nil, "", "", time.Hour, nil, pubk)
			Expect(err).ToNot(HaveOccurred())
			req.CallerPublicData, err = tokens.SignToken(claims, prik)
			Expect(err).ToNot(HaveOccurred())

			allowed, err := aaasvcPolicyAuthorize(req, agent, log)
			Expect(err).To(MatchError("no policy received in token"))
			Expect(allowed).To(BeFalse())
		})

		Context("Allowed Agents", func() {
			It("Should handle failures", func() {
				claims, err = tokens.NewClientIDClaims("ginkgo", []string{"fail"}, "choria", nil, "", "", time.Hour, nil, pubk)
				Expect(err).ToNot(HaveOccurred())
				req.CallerPublicData, err = tokens.SignToken(claims, prik)
				Expect(err).ToNot(HaveOccurred())

				allowed, err := aaasvcPolicyAuthorize(req, agent, log)
				Expect(err).To(MatchError("invalid agent policy: fail"))
				Expect(allowed).To(BeFalse())
			})

			It("Should allow valid requests", func() {
				claims, err = tokens.NewClientIDClaims("ginkgo", []string{"myco.deploy"}, "choria", nil, "", "", time.Hour, nil, pubk)
				Expect(err).ToNot(HaveOccurred())
				req.CallerPublicData, err = tokens.SignToken(claims, prik)
				Expect(err).ToNot(HaveOccurred())

				allowed, err := aaasvcPolicyAuthorize(req, agent, log)
				Expect(err).ToNot(HaveOccurred())
				Expect(allowed).To(BeTrue())
			})
		})

		Context("OPA Policy", func() {
			It("Should handle failures", func() {
				claims, err = tokens.NewClientIDClaims("ginkgo", nil, "choria", nil, "invalid rego", "", time.Hour, nil, pubk)
				Expect(err).ToNot(HaveOccurred())
				req.CallerPublicData, err = tokens.SignToken(claims, prik)
				Expect(err).ToNot(HaveOccurred())

				allowed, err := aaasvcPolicyAuthorize(req, agent, log)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(MatchRegexp("could not initialize opa evaluator"))
				Expect(allowed).To(BeFalse())
			})

			It("Should allow valid requests", func() {
				claims, err = tokens.NewClientIDClaims("ginkgo", nil, "choria", nil, readFixture("testdata/policies/rego/aaa_scenario1.rego"), "", time.Hour, nil, pubk)
				Expect(err).ToNot(HaveOccurred())
				req.CallerPublicData, err = tokens.SignToken(claims, prik)
				Expect(err).ToNot(HaveOccurred())

				allowed, err := aaasvcPolicyAuthorize(req, agent, log)
				Expect(err).ToNot(HaveOccurred())
				Expect(allowed).To(BeTrue())
			})
		})
	})

	Describe("EvaluateAgentListPolicy", func() {
		It("Should support '*' agents", func() {
			ok, err := EvaluateAgentListPolicy("agent", "action", []string{"*"}, log)
			Expect(ok).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should support action wildcards", func() {
			ok, err := EvaluateAgentListPolicy("rpcutil", "action", []string{"rpcutil.*"}, log)
			Expect(ok).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())

			ok, err = EvaluateAgentListPolicy("other", "action", []string{"rpcutil.*"}, log)
			Expect(err).ToNot(HaveOccurred())
			Expect(ok).To(BeFalse())
		})

		It("Should support specific agent.action", func() {
			ok, err := EvaluateAgentListPolicy("rpcutil", "ping", []string{"rpcutil.ping"}, log)
			Expect(ok).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())

			ok, err = EvaluateAgentListPolicy("rpcutil", "other", []string{"rpcutil.ping"}, log)
			Expect(err).ToNot(HaveOccurred())
			Expect(ok).To(BeFalse())

			ok, err = EvaluateAgentListPolicy("other", "action", []string{"rpcutil.ping"}, log)
			Expect(err).ToNot(HaveOccurred())
			Expect(ok).To(BeFalse())
		})

		It("Should handle invalid policies", func() {
			ok, err := EvaluateAgentListPolicy("rpcutil", "ping", []string{"rpcutil"}, log)
			Expect(ok).To(BeFalse())
			Expect(err).To(MatchError("invalid agent policy: rpcutil"))

		})
	})

	Describe("EvaluateOpenPolicyAgentPolicy", func() {
		It("Should allow common scenarios", func() {
			req.Filter.AddClassFilter("apache")
			req.Filter.AddIdentityFilter("some.node")
			req.Filter.AddFactFilter("country", "==", "mt")

			claims.CallerID = "up=bob"
			claims.UserProperties = map[string]string{
				"group": "admins",
			}

			for r := 1; r <= 5; r++ {
				policy := readFixture(fmt.Sprintf("testdata/policies/rego/aaa_scenario%d.rego", r))
				claims.OPAPolicy = policy

				allowed, err := EvaluateOpenPolicyAgentPolicy(req, policy, claims, "ginkgo", log)
				Expect(err).ToNot(HaveOccurred())
				Expect(allowed).To(BeTrue())
			}
		})

		It("Should fail on all common scenarios", func() {
			policy := readFixture("testdata/policies/rego/aaa_scenario5.rego")
			claims.OPAPolicy = policy
			claims.CallerID = "up=bob"
			claims.UserProperties = map[string]string{
				"group": "admins",
			}

			req.Filter.AddClassFilter("apache")
			req.Filter.AddIdentityFilter("some.node")
			req.Filter.AddFactFilter("country", "==", "mt")

			allowed, err := EvaluateOpenPolicyAgentPolicy(req, policy, claims, "ginkgo", log)
			Expect(err).ToNot(HaveOccurred())
			Expect(allowed).To(BeTrue())

			allowed, err = EvaluateOpenPolicyAgentPolicy(req, policy, claims, "x", log)
			Expect(err).ToNot(HaveOccurred())
			Expect(allowed).To(BeFalse())
		})
	})
})
