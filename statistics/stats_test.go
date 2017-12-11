package statistics

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestFileContent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Statistics")
}

var _ = Describe("Counter", func() {
	It("Should create and retrieve counters", func() {
		a := Counter("this.is.a.test")
		a.Inc(1)
		b := Counter("this.is.a.test")
		c := Counter("other")
		Expect(a).To(Equal(b))
		Expect(b.Count()).To(Equal(int64(1)))
		Expect(a).ToNot(Equal(c))
		Expect(c.Count()).To(Equal(int64(0)))
	})
})
