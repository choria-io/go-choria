// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package mcorpc

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/filter/classes"
	"github.com/choria-io/go-choria/inter"
	imock "github.com/choria-io/go-choria/inter/imocks"
	"github.com/choria-io/go-choria/protocol"
	"github.com/choria-io/go-choria/server/agents"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("RegoPolicy", func() {
	var (
		mockctl  *gomock.Controller
		fw       *imock.MockFramework
		cfg      *config.Config
		requests = make(chan inter.ConnectorMessage)
		authz    *regoPolicy
		conn     *imock.MockConnector
		connInfo *imock.MockConnectorInfo
		srvInfo  *MockServerInfoSource
		ctx      context.Context
		am       *agents.Manager
		facts    json.RawMessage
		err      error
	)

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())
		fw, cfg = imock.NewFrameworkForTests(mockctl, GinkgoWriter)
		fw.EXPECT().ProvisionMode().Return(false).AnyTimes()
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	Describe(" tests", func() {
		BeforeEach(func() {
			cfg.ClassesFile = "testdata/policies/rego/classes.txt"
			cfg.FactSourceFile = "testdata/policies/rego/facts.json"
			cfg.ConfigFile = "testdata/server.conf"
			cfg.DisableSecurityProviderVerify = true

			conn = imock.NewMockConnector(mockctl)
			conn.EXPECT().QueueSubscribe(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			conn.EXPECT().AgentBroadcastTarget(gomock.AssignableToTypeOf("collective"), gomock.AssignableToTypeOf("agent")).DoAndReturn(func(c, a string) string {
				return fmt.Sprintf("broadcast.%s.%s", c, a)
			}).AnyTimes()

			connInfo = imock.NewMockConnectorInfo(mockctl)

			facts, err = os.ReadFile(cfg.FactSourceFile)
			Expect(err).ToNot(HaveOccurred())
			Expect(facts).ToNot(HaveLen(0))

			klasses, err := classes.ReadClasses(cfg.ClassesFile)
			Expect(err).ToNot(HaveOccurred())

			srvInfo = NewMockServerInfoSource(mockctl)
			srvInfo.EXPECT().Classes().Return(klasses).AnyTimes()
			srvInfo.EXPECT().KnownAgents().Return([]string{"ginkgo", "stub_agent", "buts_agent"}).AnyTimes()
			srvInfo.EXPECT().Facts().DoAndReturn(func() json.RawMessage { return facts })

			ctx = context.Background()

			am = agents.New(requests, fw, connInfo, srvInfo, fw.Logger("test"))
			testAgents := []string{"stub_agent", "buts_agent"}

			// Additional agents for testing comparisons in rego files
			for i := range testAgents {
				metadata := &agents.Metadata{
					Name:    testAgents[i],
					Author:  "stub@example.com",
					License: "Apache-2.0",
					Timeout: 10,
					URL:     "https://choria.io",
					Version: "1.0.0",
				}

				agent := New(metadata.Name, metadata, am.Choria(), fw.Logger("test"))
				action := func(ctx context.Context, req *Request, reply *Reply, agent *Agent, conn inter.ConnectorInfo) {}
				agent.MustRegisterAction("boop", action)
				Expect(err).ToNot(HaveOccurred())

				agent.SetServerInfo(srvInfo)

				err = am.RegisterAgent(ctx, metadata.Name, agent, conn)
				// err = cn.ServerInstance().RegisterAgent(ctx, agent.Name(), agent)
				Expect(err).ToNot(HaveOccurred())
			}

			metadata := &agents.Metadata{
				Name:    "ginkgo",
				Author:  "stub@example.com",
				License: "Apache-2.0",
				Timeout: 10,
				URL:     "https://choria.io",
				Version: "1.0.0",
			}

			ginkgoAgent := New(metadata.Name, metadata, am.Choria(), fw.Logger("test"))
			action := func(ctx context.Context, req *Request, reply *Reply, agent *Agent, conn inter.ConnectorInfo) {}
			ginkgoAgent.MustRegisterAction("boop", action)
			ginkgoAgent.SetServerInfo(srvInfo)
			Expect(err).ToNot(HaveOccurred())

			authz = &regoPolicy{
				cfg: cfg,
				log: fw.Logger("x"),
				req: &Request{
					Agent:    ginkgoAgent.meta.Name,
					Action:   "boop",
					CallerID: "choria=ginkgo.mcollective",
					Data:     json.RawMessage(`{"foo": "bar"}`),
					TTL:      60,
					Time:     time.Now(),
					Filter:   protocol.NewFilter(),
				},
				agent: ginkgoAgent,
			}
		})

		AfterEach(func() {
			mockctl.Finish()
			// cn.Stop()
			ctx.Done()
		})

		Describe("Basic tests", func() {
			Context("When the user agent or caller is right", func() {
				It("Should succeed", func() {
					auth, err := authz.authorize()
					Expect(err).ToNot(HaveOccurred())
					Expect(auth).To(BeTrue())
				})

				It("Default policy should fail", func() {
					authz.agent.meta.Name = "boop"
					auth, err := authz.authorize()

					Expect(err).ToNot(HaveOccurred())
					Expect(auth).To(BeFalse())
				})

			})

			Context("When facts are correct", func() {
				It("Should succeed", func() {
					authz.agent.meta.Name = "facts"
					auth, err := authz.authorize()
					Expect(err).ToNot(HaveOccurred())
					Expect(auth).To(BeTrue())

				})
			})

			Context("When classes are present and available", func() {
				It("Should succeed", func() {
					authz.agent.meta.Name = "classes"
					auth, err := authz.authorize()

					Expect(err).ToNot(HaveOccurred())
					Expect(auth).To(BeTrue())
				})
			})
		})

		Describe("Failing tests", func() {
			Context("When the user agent or caller is wrong", func() {
				It("Should fail if agent isn't ginkgo", func() {
					authz.req.CallerID = "not=it"
					auth, err := authz.authorize()

					Expect(err).ToNot(HaveOccurred())
					Expect(auth).To(BeFalse())
				})

				It("Should fail with a default policy", func() {
					authz.req.CallerID = "not=it"
					authz.agent.meta.Name = "boop"
					Expect(authz.agent.Name()).To(Equal("boop"))

					authz.cfg.SetOption("plugin.regopolicy.enable_default", "y")
					auth, err := authz.authorize()

					Expect(err).ToNot(HaveOccurred())
					Expect(auth).To(BeFalse())
				})
			})
		})

		Describe("Agents", func() {
			Context("If agent exists on the server", func() {
				It("Should succeed", func() {
					authz.agent.meta.Name = "agent"
					auth, err := authz.authorize()

					Expect(err).ToNot(HaveOccurred())
					Expect(auth).To(BeTrue())
				})
			})
		})

		Describe("Request data", func() {
			Context("It should succeed if the request parameters are set right", func() {
				It("Should succeed", func() {
					authz.agent.meta.Name = "data"
					auth, err := authz.authorize()

					Expect(err).ToNot(HaveOccurred())
					Expect(auth).To(BeTrue())
				})
			})
		})
	})

	Describe("Auth deny tests", func() {
		BeforeEach(func() {
			cfg.ClassesFile = "testdata/policies/rego/classes_fail.txt"
			cfg.FactSourceFile = "testdata/policies/rego/facts_fail.json"
			cfg.ConfigFile = "testdata/server.conf"
			cfg.DisableSecurityProviderVerify = true

			conn = imock.NewMockConnector(mockctl)
			conn.EXPECT().QueueSubscribe(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			conn.EXPECT().AgentBroadcastTarget(gomock.AssignableToTypeOf("collective"), gomock.AssignableToTypeOf("agent")).DoAndReturn(func(c, a string) string {
				return fmt.Sprintf("broadcast.%s.%s", c, a)
			}).AnyTimes()

			connInfo = imock.NewMockConnectorInfo(mockctl)

			facts, err = os.ReadFile(cfg.FactSourceFile)
			Expect(err).ToNot(HaveOccurred())
			Expect(facts).ToNot(HaveLen(0))

			klasses, err := classes.ReadClasses(cfg.ClassesFile)
			Expect(err).ToNot(HaveOccurred())

			srvInfo = NewMockServerInfoSource(mockctl)
			srvInfo.EXPECT().Classes().Return(klasses).AnyTimes()
			srvInfo.EXPECT().KnownAgents().Return([]string{"ginkgo"}).AnyTimes()
			srvInfo.EXPECT().Facts().DoAndReturn(func() json.RawMessage { return facts }).AnyTimes()

			ctx = context.Background()

			am = agents.New(requests, fw, conn, srvInfo, fw.Logger("test"))

			metadata := &agents.Metadata{
				Name:    "ginkgo",
				Author:  "stub@example.com",
				License: "Apache-2.0",
				Timeout: 10,
				URL:     "https://choria.io",
				Version: "1.0.0",
			}

			ginkgoAgent := New(metadata.Name, metadata, am.Choria(), fw.Logger("test"))
			action := func(ctx context.Context, req *Request, reply *Reply, agent *Agent, conn inter.ConnectorInfo) {}
			ginkgoAgent.MustRegisterAction("boop", action)
			Expect(err).ToNot(HaveOccurred())

			err = am.RegisterAgent(ctx, ginkgoAgent.Name(), ginkgoAgent, conn)
			Expect(err).ToNot(HaveOccurred())
			Expect(ginkgoAgent.ServerInfoSource.Facts()).ToNot(BeNil())

			authz = &regoPolicy{
				cfg: cfg,
				log: fw.Logger(""),
				req: &Request{
					Agent:    ginkgoAgent.meta.Name,
					Action:   "boop",
					CallerID: "choria=rip.mcollective",
					SenderID: "choria=rip.mcollective",
					Data:     json.RawMessage(`{"bar": "foo"}`), // reversed from above
					TTL:      60,
					Time:     time.Now(),
					Filter:   protocol.NewFilter(),
				},
				agent: ginkgoAgent,
			}

		})

		AfterEach(func() {
			ctx.Done()
		})

		Describe("Basic tests", func() {
			Context("When the user agent or caller is wrong", func() {
				It("Should deny", func() {
					auth, err := authz.authorize()
					Expect(err).ToNot(HaveOccurred())
					Expect(auth).To(BeFalse())
				})

				It("Default policy should fail", func() {
					authz.agent.meta.Name = "boop"
					auth, err := authz.authorize()

					Expect(err).ToNot(HaveOccurred())
					Expect(auth).To(BeFalse())
				})

			})

			Context("When facts are incorrect", func() {
				It("Should deny", func() {

					authz.agent.meta.Name = "facts"
					auth, err := authz.authorize()
					Expect(err).ToNot(HaveOccurred())
					Expect(auth).To(BeFalse())

				})
			})

			Context("When classes are different but available", func() {
				It("Should fail", func() {
					authz.agent.meta.Name = "classes"
					auth, err := authz.authorize()

					Expect(err).ToNot(HaveOccurred())
					Expect(auth).To(BeFalse())
				})
			})
		})

		Describe("Agents", func() {
			Context("If agent does not exist on the server", func() {
				It("Should fail", func() {
					authz.agent.meta.Name = "agent"
					auth, err := authz.authorize()

					Expect(err).ToNot(HaveOccurred())
					Expect(auth).To(BeFalse())
				})
			})
		})

		Describe("Request data", func() {
			Context("The request parameters aren't set right", func() {
				It("Should fail", func() {
					authz.agent.meta.Name = "data"
					auth, err := authz.authorize()

					Expect(err).ToNot(HaveOccurred())
					Expect(auth).To(BeFalse())
				})
			})
		})
	})

	Describe("Multiple allow statement tests", func() {
		BeforeEach(func() {
			cfg.ClassesFile = "testdata/policies/rego/classes_fail.txt"
			cfg.FactSourceFile = "testdata/policies/rego/facts_fail.json"
			cfg.ConfigFile = "testdata/server.conf"
			cfg.DisableSecurityProviderVerify = true

			conn = imock.NewMockConnector(mockctl)
			conn.EXPECT().QueueSubscribe(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			conn.EXPECT().AgentBroadcastTarget(gomock.AssignableToTypeOf("collective"), gomock.AssignableToTypeOf("agent")).DoAndReturn(func(c, a string) string {
				return fmt.Sprintf("broadcast.%s.%s", c, a)
			}).AnyTimes()

			connInfo = imock.NewMockConnectorInfo(mockctl)

			facts, err = os.ReadFile(cfg.FactSourceFile)
			Expect(err).ToNot(HaveOccurred())
			Expect(facts).ToNot(HaveLen(0))

			klasses, err := classes.ReadClasses(cfg.ClassesFile)
			Expect(err).ToNot(HaveOccurred())

			srvInfo = NewMockServerInfoSource(mockctl)
			srvInfo.EXPECT().Classes().Return(klasses).AnyTimes()
			srvInfo.EXPECT().KnownAgents().Return([]string{"ginkgo"}).AnyTimes()
			srvInfo.EXPECT().Facts().DoAndReturn(func() json.RawMessage { return facts }).AnyTimes()

			ctx = context.Background()

			am = agents.New(requests, fw, conn, srvInfo, fw.Logger("test"))

		})

		AfterEach(func() {
			ctx.Done()
		})

		Describe("With the agent set to gingko", func() {
			BeforeEach(func() {
				metadata := &agents.Metadata{
					Name:    "ginkgo",
					Author:  "stub@example.com",
					License: "Apache-2.0",
					Timeout: 10,
					URL:     "https://choria.io",
					Version: "1.0.0",
				}

				ginkgoAgent := New(metadata.Name, metadata, am.Choria(), fw.Logger("test"))
				action := func(ctx context.Context, req *Request, reply *Reply, agent *Agent, conn inter.ConnectorInfo) {}
				ginkgoAgent.MustRegisterAction("boop", action)
				Expect(err).ToNot(HaveOccurred())

				err = am.RegisterAgent(ctx, ginkgoAgent.Name(), ginkgoAgent, conn)
				Expect(err).ToNot(HaveOccurred())
				Expect(ginkgoAgent.ServerInfoSource.Facts()).ToNot(BeNil())

				authz = &regoPolicy{
					cfg: cfg,
					log: fw.Logger(""),
					req: &Request{
						Agent:    ginkgoAgent.meta.Name,
						Action:   "boop",
						CallerID: "choria=rip.mcollective",
						SenderID: "choria=rip.mcollective",
						Data:     json.RawMessage(`{"bar": "foo"}`), // reversed from above
						TTL:      60,
						Time:     time.Now(),
						Filter:   protocol.NewFilter(),
					},
					agent: ginkgoAgent,
				}
			})

			Context("with multiple allow statements", func() {
				It("Should allow", func() {
					authz.agent.meta.Name = "multiple"
					auth, err := authz.authorize()

					Expect(err).ToNot(HaveOccurred())
					Expect(auth).To(BeTrue())
				})
			})
		})

		Describe("With the agent set to other", func() {
			BeforeEach(func() {
				metadata := &agents.Metadata{
					Name:    "other",
					Author:  "stub@example.com",
					License: "Apache-2.0",
					Timeout: 10,
					URL:     "https://choria.io",
					Version: "1.0.0",
				}

				ginkgoAgent := New(metadata.Name, metadata, am.Choria(), fw.Logger("test"))
				action := func(ctx context.Context, req *Request, reply *Reply, agent *Agent, conn inter.ConnectorInfo) {}
				ginkgoAgent.MustRegisterAction("boop", action)
				Expect(err).ToNot(HaveOccurred())

				err = am.RegisterAgent(ctx, ginkgoAgent.Name(), ginkgoAgent, conn)
				Expect(err).ToNot(HaveOccurred())
				Expect(ginkgoAgent.ServerInfoSource.Facts()).ToNot(BeNil())

				authz = &regoPolicy{
					cfg: cfg,
					log: fw.Logger(""),
					req: &Request{
						Agent:    ginkgoAgent.meta.Name,
						Action:   "poob",
						CallerID: "choria=rip.mcollective",
						SenderID: "choria=rip.mcollective",
						Data:     json.RawMessage(`{"bar": "foo"}`), // reversed from above
						TTL:      60,
						Time:     time.Now(),
						Filter:   protocol.NewFilter(),
					},
					agent: ginkgoAgent,
				}
			})

			Context("with multiple allow statements", func() {
				It("Should allow", func() {
					authz.agent.meta.Name = "multiple"
					auth, err := authz.authorize()

					Expect(err).ToNot(HaveOccurred())
					Expect(auth).To(BeTrue())
				})
			})
		})

		Describe("With the agent set to somethingelse", func() {
			BeforeEach(func() {
				metadata := &agents.Metadata{
					Name:    "somethingelse",
					Author:  "stub@example.com",
					License: "Apache-2.0",
					Timeout: 10,
					URL:     "https://choria.io",
					Version: "1.0.0",
				}

				ginkgoAgent := New(metadata.Name, metadata, am.Choria(), fw.Logger("test"))
				action := func(ctx context.Context, req *Request, reply *Reply, agent *Agent, conn inter.ConnectorInfo) {}
				ginkgoAgent.MustRegisterAction("boop", action)
				Expect(err).ToNot(HaveOccurred())

				err = am.RegisterAgent(ctx, ginkgoAgent.Name(), ginkgoAgent, conn)
				Expect(err).ToNot(HaveOccurred())
				Expect(ginkgoAgent.ServerInfoSource.Facts()).ToNot(BeNil())

				authz = &regoPolicy{
					cfg: cfg,
					log: fw.Logger(""),
					req: &Request{
						Agent:    ginkgoAgent.meta.Name,
						Action:   "poob",
						CallerID: "choria=rip.mcollective",
						SenderID: "choria=rip.mcollective",
						Data:     json.RawMessage(`{"bar": "foo"}`), // reversed from above
						TTL:      60,
						Time:     time.Now(),
						Filter:   protocol.NewFilter(),
					},
					agent: ginkgoAgent,
				}
			})

			Context("with multiple allow statements", func() {
				It("Should deny", func() {
					authz.agent.meta.Name = "multiple"
					auth, err := authz.authorize()

					Expect(err).ToNot(HaveOccurred())
					Expect(auth).To(BeFalse())
				})
			})
		})
	})
})
