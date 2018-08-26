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
		})
	})

	Describe("Component", func() {
		It("Should return the right component", func() {
			e := &ShutdownEvent{Comp: "test"}
			Expect(e.Component()).To(Equal("test"))
		})
	})

	Describe("SetComponent", func() {
		It("Should set the component", func() {
			e := &ShutdownEvent{}
			e.SetComponent("component")
			Expect(e.Comp).To(Equal("component"))
		})
	})

	Describe("SetIdentity", func() {
		It("Should set the identity", func() {
			e := &ShutdownEvent{}
			e.SetIdentity("node.example")
			Expect(e.Identity).To(Equal("node.example"))
		})
	})

	Describe("Target", func() {
		It("Should detect incomplete events", func() {
			e := &ShutdownEvent{}
			_, err := e.Target()
			Expect(err).To(MatchError("event is not complete, component has not been set"))
		})

		It("Should return the right target", func() {
			e := newShutdownEvent(Component("ginkgo"))
			t, err := e.Target()
			Expect(err).ToNot(HaveOccurred())
			Expect(t).To(Equal("choria.lifecycle.event.shutdown.ginkgo"))
		})
	})

	Describe("String", func() {
		It("Should return the right string", func() {
			e := newShutdownEvent(Component("ginkgo"), Identity("node.example"))
			Expect(e.String()).To(Equal("[shutdown] node.example: ginkgo"))
		})
	})

	Describe("Type", func() {
		It("Should return the right Type", func() {
			e := &ShutdownEvent{}
			Expect(e.Type()).To(Equal(Shutdown))
		})
	})
})
