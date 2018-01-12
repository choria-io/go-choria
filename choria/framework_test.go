package choria

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestMCollective(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "MCollective")
}

var _ = Describe("Choria", func() {
	var _ = Describe("NewChoria", func() {
		It("Should initialize choria correctly", func() {
			c := newChoria()
			Expect(c.DiscoveryHost).To(Equal("puppet"))
			Expect(c.DiscoveryPort).To(Equal(8085))
			Expect(c.UseSRVRecords).To(BeTrue())
		})
	})
})
