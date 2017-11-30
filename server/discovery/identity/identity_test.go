package identity

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestFileContent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Server/Discovery/Identity")
}

var _ = Describe("Server/Discovery/Identity", func() {
	It("Should support regex", func() {
		Expect(Match([]string{"foo", "bar", "/example/"}, "tests.example.net")).To(BeTrue())
		Expect(Match([]string{"foo", "bar", "/baz/"}, "tests.example.net")).To(BeFalse())
	})

	It("Should support exact matches", func() {
		Expect(Match([]string{"foo", "bar", "tests.example.net"}, "tests.example.net")).To(BeTrue())
		Expect(Match([]string{"foo", "bar", "test"}, "tests.example.net")).To(BeFalse())
	})
})
