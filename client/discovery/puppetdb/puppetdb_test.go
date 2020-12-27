package puppetdb

import (
	"fmt"
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/choria-io/go-choria/protocol"
)

func TestPuppetDB(t *testing.T) {
	os.Setenv("MCOLLECTIVE_CERTNAME", "rip.mcollective")
	RegisterFailHandler(Fail)
	RunSpecs(t, "Client/Discovery/PuppetDB")
}

var _ = Describe("PuppetDB", func() {
	var (
		discovery PuppetDB
	)

	BeforeEach(func() {
		discovery = PuppetDB{}
	})

	Describe("stringRegex", func() {
		It("Should correctly create case insensitive regex", func() {
			Expect(discovery.stringRegex("a1_$-2bZ")).To(Equal("[aA]1_$-2[bB][zZ]"))
		})
	})

	Describe("capitalizePuppetResource", func() {
		It("Should correctly capitalize resources", func() {
			Expect(discovery.capitalizePuppetResource("foo")).To(Equal("Foo"))
			Expect(discovery.capitalizePuppetResource("Foo")).To(Equal("Foo"))
			Expect(discovery.capitalizePuppetResource("foo::bar")).To(Equal("Foo::Bar"))
			Expect(discovery.capitalizePuppetResource("Foo::Bar")).To(Equal("Foo::Bar"))
		})
	})

	Describe("isNumeric", func() {
		It("Should correctly detect numbers", func() {
			Expect(discovery.isNumeric("100")).To(BeTrue())
			Expect(discovery.isNumeric("100.2")).To(BeTrue())
			Expect(discovery.isNumeric("100.2a")).To(BeFalse())
		})
	})

	Describe("discoverNodes", func() {
		It("Should discover nodes correctly", func() {
			has := discovery.discoverNodes([]string{"/x/", "y", `pql: nodes[certname] { facts_environment = "production" }`})
			expected := `certname ~ "[xX]" or certname = "y" or certname in nodes[certname] { facts_environment = "production" }`

			Expect(has).To(Equal(expected))
		})
	})

	Describe("discoverClasses", func() {
		It("Should correctly discover classes", func() {
			Expect(discovery.discoverClasses([]string{"/foo/", "foo::bar"})).To(Equal(`resources {type = "Class" and title ~ "[fF][oO][oO]"} and resources {type = "Class" and title = "Foo::Bar"}`))
		})
	})

	Describe("discoverAgents", func() {
		It("Should search for correct classes", func() {
			has := discovery.discoverAgents([]string{"rpcutil", "rspec1", "/rs/"})
			expected := `(resources {type = "Class" and title = "Choria::Service"} or resources {type = "Class" and title = "Mcollective::Service"}) and resources {type = "File" and tag = "mcollective_agent_rspec1_server"} and resources {type = "File" and tag ~ "mcollective_agent_.*?[rR][sS].*?_server"}`
			Expect(has).To(Equal(expected))
		})
	})

	Describe("discoverCollective", func() {
		It("Should search in facts", func() {
			Expect(discovery.discoverCollective("rspec_collective")).To(Equal(`certname in inventory[certname] { facts.mcollective.server.collectives.match("\d+") = "rspec_collective" }`))
		})
	})

	Describe("discoverFacts", func() {
		It("Should support all operators", func() {
			cases := []struct {
				Filter  []protocol.FactFilter
				Expects string
				Error   error
			}{
				{[]protocol.FactFilter{{"f", "=~", "v"}}, `inventory {facts.f ~ "[vV]"}`, nil},
				{[]protocol.FactFilter{{"f", "==", "v"}}, `inventory {facts.f = "v"}`, nil},
				{[]protocol.FactFilter{{"f", "!=", "v"}}, `inventory {!(facts.f = "v")}`, nil},
				{[]protocol.FactFilter{{"f", ">=", "v"}}, "", fmt.Errorf("'>=' operator supports only numeric values")},
				{[]protocol.FactFilter{{"f", ">=", "1"}}, `inventory {facts.f >= 1}`, nil},
			}

			for _, tc := range cases {
				if tc.Error == nil {
					Expect(discovery.discoverFacts(tc.Filter)).To(Equal(tc.Expects))
				} else {
					_, err := discovery.discoverFacts(tc.Filter)
					Expect(err).To(MatchError(tc.Error))
				}
			}
		})
	})
})
