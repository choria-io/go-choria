package agents

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestFileContent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Server/Discovery/Agents")
}

var _ = Describe("Server/Discovery/Agents", func() {
	It("Should support regex", func() {
		Expect(Match([]string{"/rpc/", "/choria/"}, []string{"rpcutil", "package", "choriautil"})).To(BeTrue())
		Expect(Match([]string{"/foo/"}, []string{"rpcutil", "package", "choriautil"})).To(BeFalse())
	})

	It("Should support exact matches", func() {
		Expect(Match([]string{"rpcutil", "choria_util"}, []string{"rpcutil", "package", "choria_util"})).To(BeTrue())
		Expect(Match([]string{"foo", "choria_util"}, []string{"rpcutil", "package", "choria_util"})).To(BeFalse())
	})
})
