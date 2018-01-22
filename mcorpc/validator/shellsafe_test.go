package validator

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ShellSafe", func() {
	It("Should match bad strings", func() {
		badchars := []string{"`", "$", ";", "|", "&&", ">", "<"}

		for _, c := range badchars {
			Expect(ShellSafe(fmt.Sprintf("thing%sthing", c))).To(BeFalse())
		}
	})

	It("Should allow good things", func() {
		Expect(ShellSafe("ok")).To(BeTrue())
		Expect(ShellSafe("")).To(BeTrue())
		Expect(ShellSafe("ok ok ok")).To(BeTrue())
	})
})
