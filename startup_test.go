package lifecycle

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("StartupEvent", func() {
	Describe("newStartupEvent", func() {
		It("Should create the event and set options", func() {
			event := newStartupEvent(Component("ginkgo"))
			Expect(event.Component()).To(Equal("ginkgo"))
		})
	})

	Describe("newStartupEventFromJSON", func() {
		It("Should detect invalid protocols", func() {
			_, err := newStartupEventFromJSON([]byte(`{"protocol":"x"}`))
			Expect(err).To(MatchError("invalid protocol 'x'"))
		})

		It("Should parse valid events", func() {
			event, err := newStartupEventFromJSON([]byte(`{"protocol":"choria:lifecycle:startup:1", "component":"ginkgo"}`))
			Expect(err).ToNot(HaveOccurred())
			Expect(event.Component()).To(Equal("ginkgo"))
		})
	})

	Describe("Component", func() {
		It("Should return the right component", func() {
			e := &StartupEvent{Comp: "test"}
			Expect(e.Component()).To(Equal("test"))
		})
	})

	Describe("SetComponent", func() {
		It("Should set the component", func() {
			e := &StartupEvent{}
			e.SetComponent("component")
			Expect(e.Comp).To(Equal("component"))
		})
	})

	Describe("SetVersion", func() {
		It("Set the version", func() {
			e := &StartupEvent{}
			e.SetVersion("1.2.3")
			Expect(e.Version).To(Equal("1.2.3"))
		})
	})

	Describe("SetIdentity", func() {
		It("Should set the identity", func() {
			e := &StartupEvent{}
			e.SetIdentity("node.example")
			Expect(e.Ident).To(Equal("node.example"))
		})
	})

	Describe("Target", func() {
		It("Should detect incomplete events", func() {
			e := &StartupEvent{}
			_, err := e.Target()
			Expect(err).To(MatchError("event is not complete, component has not been set"))
		})

		It("Should return the right target", func() {
			e := newStartupEvent(Component("ginkgo"))
			t, err := e.Target()
			Expect(err).ToNot(HaveOccurred())
			Expect(t).To(Equal("choria.lifecycle.event.startup.ginkgo"))
		})
	})

	Describe("String", func() {
		It("Should return the right string", func() {
			e := newStartupEvent(Component("ginkgo"), Identity("node.example"), Version("1.2.3"))
			Expect(e.String()).To(Equal("[startup] node.example: ginkgo version 1.2.3"))
		})
	})

	Describe("Type", func() {
		It("Should return the right Type", func() {
			e := &StartupEvent{}
			Expect(e.Type()).To(Equal(Startup))
		})
	})
})
