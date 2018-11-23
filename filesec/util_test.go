package filesec

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("MatchAnyRegex", func() {
	It("Should correctly match valid patterns", func() {
		patterns := []string{
			"bare",
			"/this.+other/",
		}

		Expect(MatchAnyRegex([]byte("this is a bare word sentence"), patterns)).To(BeTrue())
		Expect(MatchAnyRegex([]byte("this, that and the other"), patterns)).To(BeTrue())
		Expect(MatchAnyRegex([]byte("no match"), patterns)).To(BeFalse())
	})
})
