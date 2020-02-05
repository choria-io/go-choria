package mcorpc

import (
	"context"
	"encoding/json"
	"time"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/server/agents"
	"github.com/choria-io/go-config"
	"github.com/choria-io/go-protocol/protocol"
	"github.com/choria-io/go-testutil"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

var _ = Describe("RegoPolicy", func() {
	Describe(" tests", func() {

		var (
			requests = make(chan *choria.ConnectorMessage)
			authz    *regoPolicy
			logger   *logrus.Entry
			fw       *choria.Framework
			cn       *testutil.ChoriaNetwork
			err      error
			ctx      context.Context
			am       *agents.Manager
		)

		BeforeEach(func() {

			logger = logrus.NewEntry(logrus.New())
			logger.Logger.SetLevel(logrus.DebugLevel)
			logger.Logger.Out = GinkgoWriter

			cfg := config.NewConfigForTests()
			cfg.ClassesFile = "testdata/policies/rego/classes.txt"
			cfg.FactSourceFile = "testdata/policies/rego/facts.json"
			cfg.ConfigFile = "testdata/server.conf"

			cfg.DisableSecurityProviderVerify = true

			fw, err = choria.NewWithConfig(cfg)
			Expect(err).ToNot(HaveOccurred())

			cn, err = testutil.StartChoriaNetwork(cfg)
			Expect(err).ToNot(HaveOccurred())

			ctx = context.Background()

			am = agents.New(requests,
				fw,
				nil,
				cn.ServerInstance(),
				fw.Logger("test"),
			)

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
				action := func(ctx context.Context, req *Request, reply *Reply, agent *Agent, conn choria.ConnectorInfo) {}

				agent.MustRegisterAction("boop", action)
				Expect(err).ToNot(HaveOccurred())

				err = cn.ServerInstance().RegisterAgent(ctx, agent.Name(), agent)
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
			action := func(ctx context.Context, req *Request, reply *Reply, agent *Agent, conn choria.ConnectorInfo) {}
			ginkgoAgent.MustRegisterAction("boop", action)
			Expect(err).ToNot(HaveOccurred())

			err = cn.ServerInstance().RegisterAgent(ctx, ginkgoAgent.Name(), ginkgoAgent)
			Expect(err).ToNot(HaveOccurred())
			Expect(ginkgoAgent.ServerInfoSource.Facts()).ToNot(BeNil())

			authz = &regoPolicy{
				cfg: cfg,
				log: logger,
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
			cn.Stop()
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
		var (
			requests = make(chan *choria.ConnectorMessage)
			authz    *regoPolicy
			logger   *logrus.Entry
			fw       *choria.Framework
			cn       *testutil.ChoriaNetwork
			err      error
			ctx      context.Context
			am       *agents.Manager
		)

		BeforeEach(func() {

			logger = logrus.NewEntry(logrus.New())
			logger.Logger.SetLevel(logrus.DebugLevel)
			logger.Logger.Out = GinkgoWriter

			cfg := config.NewConfigForTests()
			cfg.ClassesFile = "testdata/policies/rego/classes_fail.txt"
			cfg.FactSourceFile = "testdata/policies/rego/facts_fail.json"
			cfg.ConfigFile = "testdata/server.conf"

			cfg.DisableSecurityProviderVerify = true

			fw, err = choria.NewWithConfig(cfg)
			Expect(err).ToNot(HaveOccurred())

			cn, err = testutil.StartChoriaNetwork(cfg)
			Expect(err).ToNot(HaveOccurred())

			ctx = context.Background()

			am = agents.New(requests,
				fw,
				nil,
				cn.ServerInstance(),
				fw.Logger("test"),
			)

			metadata := &agents.Metadata{
				Name:    "ginkgo",
				Author:  "stub@example.com",
				License: "Apache-2.0",
				Timeout: 10,
				URL:     "https://choria.io",
				Version: "1.0.0",
			}

			ginkgoAgent := New(metadata.Name, metadata, am.Choria(), fw.Logger("test"))
			action := func(ctx context.Context, req *Request, reply *Reply, agent *Agent, conn choria.ConnectorInfo) {}
			ginkgoAgent.MustRegisterAction("boop", action)
			Expect(err).ToNot(HaveOccurred())

			err = cn.ServerInstance().RegisterAgent(ctx, ginkgoAgent.Name(), ginkgoAgent)
			Expect(err).ToNot(HaveOccurred())
			Expect(ginkgoAgent.ServerInfoSource.Facts()).ToNot(BeNil())

			authz = &regoPolicy{
				cfg: cfg,
				log: logger,
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
			cn.Stop()
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
		var (
			requests = make(chan *choria.ConnectorMessage)
			authz    *regoPolicy
			logger   *logrus.Entry
			fw       *choria.Framework
			cn       *testutil.ChoriaNetwork
			err      error
			ctx      context.Context
			am       *agents.Manager
			cfg      *config.Config
		)

		BeforeEach(func() {

			logger = logrus.NewEntry(logrus.New())
			logger.Logger.SetLevel(logrus.DebugLevel)
			logger.Logger.Out = GinkgoWriter

			cfg = config.NewConfigForTests()
			cfg.ClassesFile = "testdata/policies/rego/classes_fail.txt"
			cfg.FactSourceFile = "testdata/policies/rego/facts_fail.json"
			cfg.ConfigFile = "testdata/server.conf"

			cfg.DisableSecurityProviderVerify = true

			fw, err = choria.NewWithConfig(cfg)
			Expect(err).ToNot(HaveOccurred())

			cn, err = testutil.StartChoriaNetwork(cfg)
			Expect(err).ToNot(HaveOccurred())

			ctx = context.Background()

		})

		AfterEach(func() {
			cn.Stop()
			ctx.Done()
		})

		Describe("With the agent set to gingko", func() {
			BeforeEach(func() {
				am = agents.New(requests,
					fw,
					nil,
					cn.ServerInstance(),
					fw.Logger("test"),
				)

				metadata := &agents.Metadata{
					Name:    "ginkgo",
					Author:  "stub@example.com",
					License: "Apache-2.0",
					Timeout: 10,
					URL:     "https://choria.io",
					Version: "1.0.0",
				}

				ginkgoAgent := New(metadata.Name, metadata, am.Choria(), fw.Logger("test"))
				action := func(ctx context.Context, req *Request, reply *Reply, agent *Agent, conn choria.ConnectorInfo) {}
				ginkgoAgent.MustRegisterAction("boop", action)
				Expect(err).ToNot(HaveOccurred())

				err = cn.ServerInstance().RegisterAgent(ctx, ginkgoAgent.Name(), ginkgoAgent)
				Expect(err).ToNot(HaveOccurred())
				Expect(ginkgoAgent.ServerInfoSource.Facts()).ToNot(BeNil())

				authz = &regoPolicy{
					cfg: cfg,
					log: logger,
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
				am = agents.New(requests,
					fw,
					nil,
					cn.ServerInstance(),
					fw.Logger("test"),
				)

				metadata := &agents.Metadata{
					Name:    "other",
					Author:  "stub@example.com",
					License: "Apache-2.0",
					Timeout: 10,
					URL:     "https://choria.io",
					Version: "1.0.0",
				}

				ginkgoAgent := New(metadata.Name, metadata, am.Choria(), fw.Logger("test"))
				action := func(ctx context.Context, req *Request, reply *Reply, agent *Agent, conn choria.ConnectorInfo) {}
				ginkgoAgent.MustRegisterAction("boop", action)
				Expect(err).ToNot(HaveOccurred())

				err = cn.ServerInstance().RegisterAgent(ctx, ginkgoAgent.Name(), ginkgoAgent)
				Expect(err).ToNot(HaveOccurred())
				Expect(ginkgoAgent.ServerInfoSource.Facts()).ToNot(BeNil())

				authz = &regoPolicy{
					cfg: cfg,
					log: logger,
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
				am = agents.New(requests,
					fw,
					nil,
					cn.ServerInstance(),
					fw.Logger("test"),
				)

				metadata := &agents.Metadata{
					Name:    "somethingelse",
					Author:  "stub@example.com",
					License: "Apache-2.0",
					Timeout: 10,
					URL:     "https://choria.io",
					Version: "1.0.0",
				}

				ginkgoAgent := New(metadata.Name, metadata, am.Choria(), fw.Logger("test"))
				action := func(ctx context.Context, req *Request, reply *Reply, agent *Agent, conn choria.ConnectorInfo) {}
				ginkgoAgent.MustRegisterAction("boop", action)
				Expect(err).ToNot(HaveOccurred())

				err = cn.ServerInstance().RegisterAgent(ctx, ginkgoAgent.Name(), ginkgoAgent)
				Expect(err).ToNot(HaveOccurred())
				Expect(ginkgoAgent.ServerInfoSource.Facts()).ToNot(BeNil())

				authz = &regoPolicy{
					cfg: cfg,
					log: logger,
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
