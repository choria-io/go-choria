package lifecycle

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ShutdownEvent", func() {
	Describe("newShutdownEvent", func() {
		It("Should create the event and set options", func() {
			event := newShutdownEvent(Component("ginkgo"))
			Expect(event.Component()).To(Equal("ginkgo"))
			Expect(event.dtype).To(Equal(Shutdown))
			Expect(event.etype).To(Equal("shutdown"))
		})
	})

	Describe("newShutdownEventFromJSON", func() {
		It("Should detect invalid protocols", func() {
			_, err := newShutdownEventFromJSON([]byte(`{"protocol":"x"}`))
			Expect(err).To(MatchError("invalid protocol 'x'"))
		})

		It("Should parse valid events", func() {
			event, err := newShutdownEventFromJSON([]byte(`{"protocol":"choria:lifecycle:shutdown:1", "component":"ginkgo"}`))
			Expect(err).ToNot(HaveOccurred())
			Expect(event.Component()).To(Equal("ginkgo"))
			Expect(event.dtype).To(Equal(Shutdown))
			Expect(event.etype).To(Equal("shutdown"))
		})
	})
})
