package lifecycle

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("GovernorEvent", func() {
	Describe("newGovernorEvent", func() {
		It("Should create the event and set options", func() {
			event := newGovernorEvent(Component("ginkgo"), GovernorName("PUPPET"), GovernorType(GovernorEnterEvent))
			Expect(event.Component()).To(Equal("ginkgo"))
			Expect(event.Type()).To(Equal(Governor))
			Expect(event.Protocol()).To(Equal("io.choria.lifecycle.v1.governor"))
			Expect(event.EventType).To(Equal(GovernorEnterEvent))
			Expect(event.Governor).To(Equal("PUPPET"))
		})
	})

	Describe("newGovernorEventFromJSON", func() {
		It("Should detect invalid protocols", func() {
			_, err := newGovernorEnterEventFromJSON([]byte(`{"protocol":"x"}`))
			Expect(err).To(MatchError("invalid protocol 'x'"))
		})

		It("Should parse valid events", func() {
			event, err := newGovernorEnterEventFromJSON([]byte(`{"protocol":"io.choria.lifecycle.v1.governor","id":"bd57d0ea-b3f6-4715-a3be-a68f3567c187","component":"ginkgo","timestamp":1625587110,"governor":"PUPPET","event_type":"enter", "sequence":10}`))
			Expect(err).ToNot(HaveOccurred())
			Expect(event.Component()).To(Equal("ginkgo"))
			Expect(event.Type()).To(Equal(Governor))
			Expect(event.TypeString()).To(Equal("governor"))
			Expect(event.EventType).To(Equal(GovernorEnterEvent))
			Expect(event.Governor).To(Equal("PUPPET"))
			Expect(event.Sequence).To(Equal(uint64(10)))
		})
	})

	Describe("String", func() {
		It("Should return the right string", func() {
			e := newGovernorEvent(Identity("ginkgo.example.net"), Component("ginkgo"), GovernorName("PUPPET"), GovernorType(GovernorEnterEvent), GovernorSequence(10))
			Expect(e.String()).To(Equal("[governor] ginkgo.example.net: obtained slot 10 on PUPPET"))
			e.EventType = GovernorExitEvent
			Expect(e.String()).To(Equal("[governor] ginkgo.example.net: vacated slot 10 on PUPPET"))
			e.EventType = GovernorTimeoutEvent
			Expect(e.String()).To(Equal("[governor] ginkgo.example.net: failed to obtain a slot on PUPPET"))
		})
	})
})
