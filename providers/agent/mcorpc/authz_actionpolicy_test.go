// Copyright (c) 2020-2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package mcorpc

import (
	"bytes"

	"github.com/choria-io/go-choria/config"
	imock "github.com/choria-io/go-choria/inter/imocks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"
)

var _ = Describe("ActionPolicy", func() {
	var (
		authz     *actionPolicy
		pol       *actionPolicyPolicy
		logger    *logrus.Entry
		mockctl   *gomock.Controller
		cfg       *config.Config
		logbuffer *bytes.Buffer
	)

	BeforeEach(func() {
		logbuffer = &bytes.Buffer{}
		logger = logrus.NewEntry(logrus.New())
		logger.Logger.Out = logbuffer
		pol = &actionPolicyPolicy{log: logger}

		mockctl = gomock.NewController(GinkgoT())
		_, cfg = imock.NewFrameworkForTests(mockctl, GinkgoWriter)
		cfg.ClassesFile = "testdata/classes.txt"
		cfg.FactSourceFile = "testdata/facts.json"
		cfg.DisableSecurityProviderVerify = true

		authz = &actionPolicy{
			cfg:     cfg,
			log:     logger,
			matcher: pol,
			groups:  make(map[string][]string),
			req: &Request{
				Agent:    "ginkgo",
				Action:   "test",
				CallerID: "choria=ginkgo.mcollective",
			},
		}
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	Describe("parseGroupFile", func() {
		It("Should correctly parse the file", func() {
			err := authz.parseGroupFile("testdata/policies/groups")
			Expect(err).ToNot(HaveOccurred())
			Expect(authz.groups).To(Equal(map[string][]string{
				"sysadmin":     {"cert=sa1", "cert=sa2", "rspec_caller"},
				"app_admin":    {"cert=aa1", "cert=aa2"},
				"single_group": {"rspec_caller"},
			}))
		})
	})

	Describe("evaluatePolicy", func() {
		It("Should allow when default allow is set", func() {
			matched, reason, err := authz.evaluatePolicy("testdata/policies/default_allow")
			Expect(err).ToNot(HaveOccurred())
			Expect(reason).To(Equal(""))
			Expect(matched).To(BeTrue())
		})

		It("Should deny when default deny is set", func() {
			matched, reason, err := authz.evaluatePolicy("testdata/policies/default_deny")
			Expect(err).ToNot(HaveOccurred())
			Expect(reason).To(Equal("Denying based on default policy in default_deny"))
			Expect(matched).To(BeFalse())
		})

		Describe("example1", func() {
			It("Should allow all requests", func() {
				matched, reason, err := authz.evaluatePolicy("testdata/policies/example1")
				Expect(err).ToNot(HaveOccurred())
				Expect(reason).To(Equal(""))
				Expect(matched).To(BeTrue())
			})
		})

		Describe("example2", func() {
			It("Should allow the right caller", func() {
				matched, reason, err := authz.evaluatePolicy("testdata/policies/example2")
				Expect(err).ToNot(HaveOccurred())
				Expect(reason).To(Equal(""))
				Expect(matched).To(BeTrue())
			})

			It("Should deny the wrong caller", func() {
				authz.req.CallerID = "other"
				matched, reason, err := authz.evaluatePolicy("testdata/policies/example2")
				Expect(err).ToNot(HaveOccurred())
				Expect(reason).To(Equal("Denying based on default policy in example2"))
				Expect(matched).To(BeFalse())
			})

			It("Should match the regex caller", func() {
				authz.req.CallerID = "up=bob"
				matched, reason, err := authz.evaluatePolicy("testdata/policies/example2")
				Expect(err).ToNot(HaveOccurred())
				Expect(reason).To(Equal(""))
				Expect(matched).To(BeTrue())
			})
		})

		Describe("example3", func() {
			It("Should allow requests to the matching agent", func() {
				matched, reason, err := authz.evaluatePolicy("testdata/policies/example3")
				Expect(err).ToNot(HaveOccurred())
				Expect(reason).To(Equal(""))
				Expect(matched).To(BeTrue())
			})

			It("Should deny other requests", func() {
				authz.req.Action = "other"
				matched, reason, err := authz.evaluatePolicy("testdata/policies/example3")
				Expect(err).ToNot(HaveOccurred())
				Expect(reason).To(Equal("Denying based on default policy in example3"))
				Expect(matched).To(BeFalse())
			})
		})

		Describe("example4", func() {
			It("Should match correctly", func() {
				cfg.FactSourceFile = "testdata/foo_bar_facts.json"
				matched, reason, err := authz.evaluatePolicy("testdata/policies/example4")
				Expect(err).ToNot(HaveOccurred())
				Expect(reason).To(Equal(""))
				Expect(matched).To(BeTrue())
			})

			It("Should deny correctly", func() {
				cfg.FactSourceFile = "testdata/foo_baz_facts.json"
				matched, reason, err := authz.evaluatePolicy("testdata/policies/example4")
				Expect(err).ToNot(HaveOccurred())
				Expect(reason).To(Equal("Denying based on default policy in example4"))
				Expect(matched).To(BeFalse())
			})
		})

		Describe("example5", func() {
			It("Should match correctly", func() {
				matched, reason, err := authz.evaluatePolicy("testdata/policies/example5")
				Expect(err).ToNot(HaveOccurred())
				Expect(reason).To(Equal(""))
				Expect(matched).To(BeTrue())
			})

			It("Should deny correctly", func() {
				cfg.ClassesFile = "testdata/classes_2.txt"
				matched, reason, err := authz.evaluatePolicy("testdata/policies/example5")
				Expect(err).ToNot(HaveOccurred())
				Expect(reason).To(Equal("Denying based on default policy in example5"))
				Expect(matched).To(BeFalse())
			})
		})

		Describe("example6", func() {
			It("Should match correctly", func() {
				matched, reason, err := authz.evaluatePolicy("testdata/policies/example6")
				Expect(err).ToNot(HaveOccurred())
				Expect(reason).To(Equal(""))
				Expect(matched).To(BeTrue())
			})

			It("Should deny correctly", func() {
				authz.req.Action = "other"
				matched, reason, err := authz.evaluatePolicy("testdata/policies/example6")
				Expect(err).ToNot(HaveOccurred())
				Expect(reason).To(Equal("Denying based on default policy in example6"))
				Expect(matched).To(BeFalse())

				authz.req.CallerID = "other"
				matched, reason, err = authz.evaluatePolicy("testdata/policies/example6")
				Expect(err).ToNot(HaveOccurred())
				Expect(reason).To(Equal("Denying based on default policy in example6"))
				Expect(matched).To(BeFalse())
			})
		})

		Describe("example7", func() {
			It("Should match correctly", func() {
				cfg.FactSourceFile = "testdata/foo_bar_facts.json"
				matched, reason, err := authz.evaluatePolicy("testdata/policies/example7")
				Expect(err).ToNot(HaveOccurred())
				Expect(reason).To(Equal(""))
				Expect(matched).To(BeTrue())
			})

			It("Should deny correctly", func() {
				cfg.FactSourceFile = "testdata/foo_baz_facts.json"
				matched, reason, err := authz.evaluatePolicy("testdata/policies/example7")
				Expect(err).ToNot(HaveOccurred())
				Expect(reason).To(Equal("Denying based on default policy in example7"))
				Expect(matched).To(BeFalse())

				cfg.FactSourceFile = "testdata/foo_bar_facts.json"
				authz.req.CallerID = "other"
				matched, reason, err = authz.evaluatePolicy("testdata/policies/example7")
				Expect(err).ToNot(HaveOccurred())
				Expect(reason).To(Equal("Denying based on default policy in example7"))
				Expect(matched).To(BeFalse())
			})
		})

		Describe("example8", func() {
			It("Should match correctly", func() {
				matched, reason, err := authz.evaluatePolicy("testdata/policies/example8")
				Expect(err).ToNot(HaveOccurred())
				Expect(reason).To(Equal(""))
				Expect(matched).To(BeTrue())
			})

			It("Should deny correctly", func() {
				cfg.ClassesFile = "testdata/classes_2.txt"
				matched, reason, err := authz.evaluatePolicy("testdata/policies/example8")
				Expect(err).ToNot(HaveOccurred())
				Expect(reason).To(Equal("Denying based on default policy in example8"))
				Expect(matched).To(BeFalse())

				cfg.ClassesFile = "testdata/missing"
				matched, reason, err = authz.evaluatePolicy("testdata/policies/example8")
				Expect(err).ToNot(HaveOccurred())
				Expect(reason).To(Equal("Denying based on default policy in example8"))
				Expect(matched).To(BeFalse())

				cfg.ClassesFile = "testdata/classes.txt"
				authz.req.CallerID = "other"
				matched, reason, err = authz.evaluatePolicy("testdata/policies/example8")
				Expect(err).ToNot(HaveOccurred())
				Expect(reason).To(Equal("Denying based on default policy in example8"))
				Expect(matched).To(BeFalse())
			})
		})

		Describe("example9", func() {
			It("Should match correctly", func() {
				cfg.FactSourceFile = "testdata/foo_bar_facts.json"
				matched, reason, err := authz.evaluatePolicy("testdata/policies/example9")
				Expect(err).ToNot(HaveOccurred())
				Expect(reason).To(Equal(""))
				Expect(matched).To(BeTrue())
			})

			It("Should deny correctly", func() {
				authz.req.CallerID = "other"
				cfg.FactSourceFile = "testdata/foo_bar_facts.json"
				matched, reason, err := authz.evaluatePolicy("testdata/policies/example9")
				Expect(err).ToNot(HaveOccurred())
				Expect(reason).To(Equal("Denying based on default policy in example9"))
				Expect(matched).To(BeFalse())

				authz.req.CallerID = "choria=ginkgo.mcollective"
				cfg.FactSourceFile = "testdata/foo_baz_facts.json"
				matched, reason, err = authz.evaluatePolicy("testdata/policies/example9")
				Expect(err).ToNot(HaveOccurred())
				Expect(reason).To(Equal("Denying based on default policy in example9"))
				Expect(matched).To(BeFalse())

				authz.req.CallerID = "choria=ginkgo.mcollective"
				cfg.FactSourceFile = "testdata/foo_bar_facts.json"
				authz.req.Action = "other"
				matched, reason, err = authz.evaluatePolicy("testdata/policies/example9")
				Expect(err).ToNot(HaveOccurred())
				Expect(reason).To(Equal("Denying based on default policy in example9"))
				Expect(matched).To(BeFalse())

			})
		})

		Describe("example10", func() {
			It("Should match correctly", func() {
				matched, reason, err := authz.evaluatePolicy("testdata/policies/example10")
				Expect(err).ToNot(HaveOccurred())
				Expect(reason).To(Equal(""))
				Expect(matched).To(BeTrue())
			})

			It("Should deny correctly", func() {
				authz.req.CallerID = "other"
				matched, reason, err := authz.evaluatePolicy("testdata/policies/example10")
				Expect(err).ToNot(HaveOccurred())
				Expect(reason).To(Equal("Denying based on default policy in example10"))
				Expect(matched).To(BeFalse())

				authz.req.CallerID = "choria=ginkgo.mcollective"
				authz.req.Action = "other"
				matched, reason, err = authz.evaluatePolicy("testdata/policies/example10")
				Expect(err).ToNot(HaveOccurred())
				Expect(reason).To(Equal("Denying based on default policy in example10"))
				Expect(matched).To(BeFalse())

				authz.req.Action = "test"
				cfg.ClassesFile = "testdata/classes_2.txt"
				matched, reason, err = authz.evaluatePolicy("testdata/policies/example10")
				Expect(err).ToNot(HaveOccurred())
				Expect(reason).To(Equal("Denying based on default policy in example10"))
				Expect(matched).To(BeFalse())
			})
		})

		Describe("example11", func() {
			It("Should match correctly", func() {
				cfg.FactSourceFile = "testdata/foo_bar_facts.json"
				matched, reason, err := authz.evaluatePolicy("testdata/policies/example11")
				Expect(err).ToNot(HaveOccurred())
				Expect(reason).To(Equal(""))
				Expect(matched).To(BeTrue())
			})

			It("Should deny correctly", func() {
				authz.req.CallerID = "other"
				matched, reason, err := authz.evaluatePolicy("testdata/policies/example11")
				Expect(err).ToNot(HaveOccurred())
				Expect(reason).To(Equal("Denying based on default policy in example11"))
				Expect(matched).To(BeFalse())

				authz.req.CallerID = "choria=ginkgo.mcollective"
				authz.req.Action = "other"
				matched, reason, err = authz.evaluatePolicy("testdata/policies/example11")
				Expect(err).ToNot(HaveOccurred())
				Expect(reason).To(Equal("Denying based on default policy in example11"))
				Expect(matched).To(BeFalse())

				authz.req.Action = "test"
				cfg.FactSourceFile = "testdata/foo_baz_facts.json"
				matched, reason, err = authz.evaluatePolicy("testdata/policies/example11")
				Expect(err).ToNot(HaveOccurred())
				Expect(reason).To(Equal("Denying based on default policy in example11"))
				Expect(matched).To(BeFalse())

				cfg.FactSourceFile = "testdata/foo_bar_facts.json"
				cfg.ClassesFile = "testdata/classes_2.txt"
				matched, reason, err = authz.evaluatePolicy("testdata/policies/example11")
				Expect(err).ToNot(HaveOccurred())
				Expect(reason).To(Equal("Denying based on default policy in example11"))
				Expect(matched).To(BeFalse())
			})
		})

		Describe("example12", func() {
			It("Should fail due to compound statement", func() {
				matched, reason, err := authz.evaluatePolicy("testdata/policies/example12")
				Expect(logbuffer.Bytes()).To(ContainSubstring("Compound policy statements are not supported"))
				Expect(err).ToNot(HaveOccurred())
				Expect(reason).To(Equal("Denying based on default policy in example12"))
				Expect(matched).To(BeFalse())
			})
		})

		Describe("example13", func() {
			It("Should fail due to compound statement", func() {
				matched, reason, err := authz.evaluatePolicy("testdata/policies/example13")
				Expect(logbuffer.Bytes()).To(ContainSubstring("Compound policy statements are not supported"))
				Expect(err).ToNot(HaveOccurred())
				Expect(reason).To(Equal("Denying based on default policy in example13"))
				Expect(matched).To(BeFalse())
			})
		})

		Describe("example14", func() {
			It("Should fail due to compound statement", func() {
				matched, reason, err := authz.evaluatePolicy("testdata/policies/example14")
				Expect(logbuffer.Bytes()).To(ContainSubstring("Compound policy statements are not supported"))
				Expect(err).ToNot(HaveOccurred())
				Expect(reason).To(Equal("Denying based on default policy in example14"))
				Expect(matched).To(BeFalse())
			})
		})

		Describe("example15", func() {
			It("Should match policy 1", func() {
				matched, reason, err := authz.evaluatePolicy("testdata/policies/example15")
				Expect(err).ToNot(HaveOccurred())
				Expect(reason).To(Equal(""))
				Expect(matched).To(BeTrue())

				authz.req.CallerID = "other"
				matched, reason, err = authz.evaluatePolicy("testdata/policies/example15")
				Expect(err).ToNot(HaveOccurred())
				Expect(reason).To(Equal("Denying based on default policy in example15"))
				Expect(matched).To(BeFalse())
			})

			It("Should match policy 2", func() {
				authz.req.CallerID = "choria=two.mcollective"
				cfg.FactSourceFile = "testdata/foo_bar_facts.json"
				matched, reason, err := authz.evaluatePolicy("testdata/policies/example15")
				Expect(err).ToNot(HaveOccurred())
				Expect(reason).To(Equal(""))
				Expect(matched).To(BeTrue())

				cfg.FactSourceFile = "testdata/foo_baz_facts.json"
				matched, reason, err = authz.evaluatePolicy("testdata/policies/example15")
				Expect(err).ToNot(HaveOccurred())
				Expect(matched).To(BeFalse())
				Expect(reason).To(Equal("Denying based on default policy in example15"))
			})

			It("Should match policy 3", func() {
				authz.req.CallerID = "choria=three.mcollective"
				cfg.FactSourceFile = "testdata/foo_bar_facts.json"

				for _, act := range []string{"enable", "disable", "status"} {
					authz.req.Action = act
					matched, reason, err := authz.evaluatePolicy("testdata/policies/example15")
					Expect(err).ToNot(HaveOccurred())
					Expect(reason).To(Equal(""))
					Expect(matched).To(BeTrue())
				}

				authz.req.Action = "other"
				matched, reason, err := authz.evaluatePolicy("testdata/policies/example15")
				Expect(err).ToNot(HaveOccurred())
				Expect(reason).To(Equal("Denying based on default policy in example15"))
				Expect(matched).To(BeFalse())

				authz.req.Action = "status"
				cfg.FactSourceFile = "testdata/foo_baz_facts.json"
				matched, reason, err = authz.evaluatePolicy("testdata/policies/example15")
				Expect(err).ToNot(HaveOccurred())
				Expect(reason).To(Equal("Denying based on default policy in example15"))
				Expect(matched).To(BeFalse())
			})

			It("Should match policy 4", func() {
				authz.req.CallerID = "choria=four.mcollective"
				authz.req.Action = "restart"

				matched, reason, err := authz.evaluatePolicy("testdata/policies/example15")
				Expect(logbuffer.Bytes()).To(ContainSubstring("Compound policy statements are not supported"))
				Expect(err).ToNot(HaveOccurred())
				Expect(reason).To(Equal("Denying based on default policy in example15"))
				Expect(matched).To(BeFalse())
			})
		})

		Describe("example16", func() {
			It("Should match correctly", func() {
				for _, c := range []string{"uid=500", "uid=600", "uid=700"} {
					authz.req.CallerID = c
					matched, reason, err := authz.evaluatePolicy("testdata/policies/example16")
					Expect(err).ToNot(HaveOccurred())
					Expect(reason).To(Equal(""))
					Expect(matched).To(BeTrue())
				}
			})

			It("Should deny correctly", func() {
				matched, reason, err := authz.evaluatePolicy("testdata/policies/example16")
				Expect(err).ToNot(HaveOccurred())
				Expect(reason).To(Equal("Denying based on default policy in example16"))
				Expect(matched).To(BeFalse())
			})
		})

		Describe("example17", func() {
			It("Should match correctly", func() {
				authz.parseGroupFile("testdata/policies/groups")
				authz.req.CallerID = "cert=sa1"
				matched, _, err := authz.evaluatePolicy("testdata/policies/example17")
				Expect(err).ToNot(HaveOccurred())
				Expect(matched).To(BeTrue())

				authz.req.CallerID = "cert=aa1"
				matched, _, err = authz.evaluatePolicy("testdata/policies/example17")
				Expect(err).ToNot(HaveOccurred())
				Expect(matched).To(BeTrue())
			})

			It("Should deny correctly", func() {
				authz.parseGroupFile("testdata/policies/groups")
				matched, reason, err := authz.evaluatePolicy("testdata/policies/example17")
				Expect(err).ToNot(HaveOccurred())
				Expect(reason).To(Equal("Denying based on default policy in example17"))
				Expect(matched).To(BeFalse())
			})
		})
	})
})

var _ = Describe("Policy", func() {
	var (
		pol       *actionPolicyPolicy
		logger    *logrus.Entry
		logbuffer *bytes.Buffer
		cfg       *config.Config
	)

	BeforeEach(func() {
		logbuffer = &bytes.Buffer{}
		logger = logrus.NewEntry(logrus.New())
		logger.Logger.Out = logbuffer
		pol = &actionPolicyPolicy{log: logger, file: "/nonexisting"}

		cfg = config.NewConfigForTests()
		cfg.DisableSecurityProviderVerify = true
	})

	Describe("matchesFacts", func() {
		It("Should correctly match empty policy", func() {
			matched, err := pol.MatchesFacts(cfg, logger)
			Expect(err).To(MatchError("empty fact policy found"))
			Expect(matched).To(BeFalse())
		})

		It("Should correctly match *", func() {
			pol.facts = "*"
			matched, err := pol.MatchesFacts(cfg, logger)
			Expect(err).ToNot(HaveOccurred())
			Expect(matched).To(BeTrue())
		})

		It("Should correctly match compound filters", func() {
			pol.facts = "this and that"
			matched, err := pol.MatchesFacts(cfg, logger)
			Expect(err).To(MatchError("compound statements are not supported"))
			Expect(matched).To(BeFalse())
		})

		It("Should correctly catch invalid fact filters", func() {
			pol.facts = "foo bar"
			matched, err := pol.MatchesFacts(cfg, logger)
			Expect(err).To(MatchError("invalid fact matcher: could not parse fact foo it does not appear to be in a valid format"))
			Expect(matched).To(BeFalse())
		})

		It("Should correctly match facts", func() {
			cfg.FactSourceFile = "testdata/facts.json"
			pol.facts = "one=one"
			matched, err := pol.MatchesFacts(cfg, logger)
			Expect(err).ToNot(HaveOccurred())
			Expect(matched).To(BeTrue())

			pol.facts = "one=~/n/"
			matched, err = pol.MatchesFacts(cfg, logger)
			Expect(err).ToNot(HaveOccurred())
			Expect(matched).To(BeTrue())

			pol.facts = "nested.facts=~/^al/"
			matched, err = pol.MatchesFacts(cfg, logger)
			Expect(err).ToNot(HaveOccurred())
			Expect(matched).To(BeFalse())

			pol.facts = "nested.facts=~/^val/"
			matched, err = pol.MatchesFacts(cfg, logger)
			Expect(err).ToNot(HaveOccurred())
			Expect(matched).To(BeTrue())
		})
	})

	Describe("matchesClasses", func() {
		It("Should correctly match empty policy", func() {
			matched, err := pol.MatchesClasses("/tmp/classes", logger)
			Expect(err).To(MatchError("empty classes policy found"))
			Expect(matched).To(BeFalse())
		})

		It("Should correctly match empty classes files", func() {
			pol.classes = "one"
			matched, err := pol.MatchesClasses("", logger)
			Expect(err).To(MatchError("do not know how to resolve classes"))
			Expect(matched).To(BeFalse())
		})

		It("Should correctly match *", func() {
			pol.classes = "*"
			matched, err := pol.MatchesClasses("testdata/classes.txt", logger)
			Expect(err).ToNot(HaveOccurred())
			Expect(matched).To(BeTrue())
		})

		It("Should detect fact matches in classes field", func() {
			pol.classes = "foo and bar"
			matched, err := pol.MatchesClasses("testdata/classes.txt", logger)
			Expect(err).To(MatchError("compound statements are not supported"))
			Expect(matched).To(BeFalse())
		})

		It("Should match classes correctly", func() {
			pol.classes = "one two three"
			matched, err := pol.MatchesClasses("testdata/classes.txt", logger)
			Expect(err).ToNot(HaveOccurred())
			Expect(matched).To(BeTrue())

			pol.classes = "one two four"
			matched, err = pol.MatchesClasses("testdata/classes.txt", logger)
			Expect(err).ToNot(HaveOccurred())
			Expect(matched).To(BeFalse())
		})
	})

	Describe("matchesAction", func() {
		It("should correctly match empty policy", func() {
			Expect(pol.MatchesAction("install")).To(BeFalse())
		})

		It("Should support * matches", func() {
			pol.actions = "*"
			Expect(pol.MatchesAction("install")).To(BeTrue())
		})

		It("Should match actions", func() {
			pol.actions = "one two three"
			Expect(pol.MatchesAction("install")).To(BeFalse())
			Expect(pol.MatchesAction("one")).To(BeTrue())
			Expect(pol.MatchesAction("two")).To(BeTrue())
			Expect(pol.MatchesAction("three")).To(BeTrue())
		})
	})

	Describe("matchesCallerID", func() {
		It("Should correctly match empty policy", func() {
			Expect(pol.MatchesCallerID("choria=bob")).To(BeFalse())
		})

		It("Should support * matches", func() {
			pol.caller = "*"
			Expect(pol.MatchesCallerID("choria=bob")).To(BeTrue())
		})

		It("Should match callers", func() {
			pol.caller = "choria=bob choria=jill"
			Expect(pol.MatchesCallerID("choria=bob")).To(BeTrue())
			Expect(pol.MatchesCallerID("choria=jill")).To(BeTrue())
			Expect(pol.MatchesCallerID("choria=jane")).To(BeFalse())
		})

		It("Should support regex policies", func() {
			pol.caller = "choria=bob /^up=/ choria=jill"
			Expect(pol.MatchesCallerID("choria=bob")).To(BeTrue())
			Expect(pol.MatchesCallerID("up=foo")).To(BeTrue())
			Expect(pol.MatchesCallerID("up=other")).To(BeTrue())
			Expect(pol.MatchesCallerID("up^other")).To(BeFalse())
			Expect(pol.MatchesCallerID("choria=jill")).To(BeTrue())

			pol.caller = "choria=bob //"
			Expect(pol.MatchesCallerID("up=foo")).To(BeFalse())
			Expect(logbuffer.String()).To(ContainSubstring("Invalid CallerID matcher '//' found in policy file /nonexisting"))

			pol.caller = "choria=bob /*/"
			Expect(pol.MatchesCallerID("up=foo")).To(BeFalse())
			Expect(logbuffer.String()).To(ContainSubstring("Could not compile regex found in CallerID '/*/' in policy file /nonexisting: error parsing regexp: missing argument to repetition operator: `*`"))

		})
	})

	Describe("isCallerInGroups", func() {
		It("Should match on known groups", func() {
			groups := map[string][]string{
				"sysadmin":     {"cert=sa1", "cert=sa2", "rspec_caller"},
				"app_admin":    {"cert=aa1", "cert=aa2"},
				"single_group": {"rspec_caller"},
			}

			pol.groups = groups
			pol.caller = "app_admin sysadmin"
			Expect(pol.isCallerInGroups("cert=sa1")).To(BeTrue())
			Expect(pol.isCallerInGroups("cert=aa1")).To(BeTrue())
			Expect(pol.isCallerInGroups("other")).To(BeFalse())
		})
	})

	Describe("sCompound", func() {
		It("should detect combound filters correctly", func() {
			Expect(pol.IsCompound("one two")).To(BeFalse())
			Expect(pol.IsCompound("country=mt os=linux")).To(BeFalse())
			Expect(pol.IsCompound("this and that")).To(BeTrue())
			Expect(pol.IsCompound("this or that")).To(BeTrue())
			Expect(pol.IsCompound("this or not that")).To(BeTrue())
		})
	})
})
