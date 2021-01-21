package flatfile

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/choria-io/go-choria/protocol"
)

func TestFlatfile(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Client/Discovery/Flatfile")
}

var _ = Describe("Flatfile", func() {
	It("Should expect a source", func() {
		ff := &FlatFile{}
		n, err := ff.Discover(context.Background())
		Expect(n).To(HaveLen(0))
		Expect(err).To(MatchError("source file not specified"))
	})

	It("Should validate the filter", func() {
		filter := protocol.NewFilter()
		filter.AddAgentFilter("test")
		ff := &FlatFile{}
		n, err := ff.Discover(context.Background(), Filter(filter), File("testdata/nodes.txt"))
		Expect(n).To(HaveLen(0))
		Expect(err).To(MatchError("only identity filters are supported"))
	})

	It("Should support flat files", func() {
		ff := &FlatFile{}
		n, err := ff.Discover(context.Background(), File("testdata/nodes.txt"), Format(TextFormat))
		Expect(n).To(Equal([]string{"one", "two", "three"}))
		Expect(err).ToNot(HaveOccurred())
	})

	It("Should support json files", func() {
		ff := &FlatFile{}
		n, err := ff.Discover(context.Background(), File("testdata/nodes.json"), Format(JSONFormat))
		Expect(err).ToNot(HaveOccurred())
		Expect(n).To(Equal([]string{"one.json", "two.json", "three.json"}))
	})

	It("Should support yaml files", func() {
		ff := &FlatFile{}
		n, err := ff.Discover(context.Background(), File("testdata/nodes.yaml"), Format(YAMLFormat))
		Expect(err).ToNot(HaveOccurred())
		Expect(n).To(Equal([]string{"one.yaml", "two.yaml", "three.yaml"}))
	})

	It("Should support choria response files", func() {
		ff := &FlatFile{}
		n, err := ff.Discover(context.Background(), File("testdata/choria.json"), Format(ChoriaResponsesFormat))
		Expect(err).ToNot(HaveOccurred())
		Expect(n).To(Equal([]string{"n1.example.net", "n2.example.net", "n3.example.net"}))
	})

	It("Should support identity filters", func() {
		filter := protocol.NewFilter()
		filter.AddIdentityFilter("n1.example.net")
		filter.AddIdentityFilter("/n2/")

		ff := &FlatFile{}
		n, err := ff.Discover(context.Background(), File("testdata/choria.json"), Format(ChoriaResponsesFormat), Filter(filter))
		Expect(err).ToNot(HaveOccurred())
		Expect(n).To(Equal([]string{"n1.example.net", "n2.example.net"}))
	})

	It("Should validate node names", func() {
		ff := &FlatFile{}
		n, err := ff.Discover(context.Background(), File("testdata/badnodes.txt"), Format(TextFormat))
		Expect(err).To(MatchError(`invalid identity string "foo bar"`))
		Expect(n).To(BeEmpty())
	})

	It("Should support gjson matches", func() {
		ff := &FlatFile{}
		n, err := ff.Discover(context.Background(), File("testdata/choria.json"), Format(JSONFormat), DiscoveryOptions(map[string]string{
			"filter": "replies.#.sender",
		}))
		Expect(err).ToNot(HaveOccurred())
		Expect(n).To(Equal([]string{"n1.example.net", "n2.example.net", "n3.example.net"}))
	})

	It("Should support file overrides", func() {
		ff := &FlatFile{}
		n, err := ff.Discover(context.Background(), File("testdata/choria.json"), DiscoveryOptions(map[string]string{
			"format": "text",
			"file":   "testdata/nodes.txt",
		}))
		Expect(err).ToNot(HaveOccurred())
		Expect(n).To(Equal([]string{"one", "two", "three"}))
	})
})
