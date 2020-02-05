package lifecycle

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("AliveEvent", func() {
	Describe("newAliveEvent", func() {
		It("Should create the event and set options", func() {
			event := newAliveEvent(Component("ginkgo"), Version("1.2.3"))
			Expect(event.Component()).To(Equal("ginkgo"))
			Expect(event.Type()).To(Equal(Alive))
			Expect(event.Version).To(Equal("1.2.3"))
		})
	})

	Describe("newAliveEventFromJSON", func() {
		It("Should detect invalid protocols", func() {
			_, err := newAliveEventFromJSON([]byte(`{"protocol":"x"}`))
			Expect(err).To(MatchError("invalid protocol 'x'"))
		})

		It("Should parse valid events", func() {
			event, err := newAliveEventFromJSON([]byte(`{"protocol":"choria:lifecycle:alive:1", "component":"ginkgo", "version":"1.2.3"}`))
			Expect(err).ToNot(HaveOccurred())
			Expect(event.Component()).To(Equal("ginkgo"))
			Expect(event.Type()).To(Equal(Alive))
			Expect(event.TypeString()).To(Equal("alive"))
			Expect(event.Version).To(Equal("1.2.3"))
		})
	})

	Describe("SetVersion", func() {
		It("Set the version", func() {
			e := &AliveEvent{}
			e.SetVersion("1.2.3")
			Expect(e.Version).To(Equal("1.2.3"))
		})
	})

	Describe("Target", func() {
		It("Should detect incomplete events", func() {
			e := &AliveEvent{}
			_, err := e.Target()
			Expect(err).To(MatchError("event is not complete, component has not been set"))
		})

		It("Should return the right target", func() {
			e := newAliveEvent(Component("ginkgo"))
			t, err := e.Target()
			Expect(err).ToNot(HaveOccurred())
			Expect(t).To(Equal("choria.lifecycle.event.alive.ginkgo"))
		})
	})

	Describe("String", func() {
		It("Should return the right string", func() {
			e := newAliveEvent(Component("ginkgo"), Identity("node.example"), Version("1.2.3"))
			Expect(e.String()).To(Equal("[alive] node.example: ginkgo version 1.2.3"))
		})
	})
})
