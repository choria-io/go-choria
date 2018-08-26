package lifecycle

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Options", func() {
	var event *StartupEvent

	BeforeEach(func() {
		event = &StartupEvent{}
	})

	Describe("Identity", func() {
		It("Should set the identity", func() {
			Identity("ginkgo")(event)
			Expect(event.Identity).To(Equal("ginkgo"))
		})
	})

	Describe("Version", func() {
		It("Should set the version", func() {
			Version("0.99.9")(event)
			Expect(event.Version).To(Equal("0.99.9"))
		})
	})

	Describe("Component", func() {
		It("Should set the component", func() {
			Component("test")(event)
			Expect(event.Comp).To(Equal("test"))
		})
	})
})
