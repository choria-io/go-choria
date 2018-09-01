package lifecycle

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("basicEvent", func() {
	Describe("Component", func() {
		It("Should return the right component", func() {
			e := &basicEvent{Comp: "test"}
			Expect(e.Component()).To(Equal("test"))
		})
	})

	Describe("SetComponent", func() {
		It("Should set the component", func() {
			e := &basicEvent{}
			e.SetComponent("component")
			Expect(e.Comp).To(Equal("component"))
		})
	})

	Describe("SetIdentity", func() {
		It("Should set the identity", func() {
			e := &basicEvent{}
			e.SetIdentity("node.example")
			Expect(e.Ident).To(Equal("node.example"))
		})
	})

	Describe("Target", func() {
		It("Should detect incomplete events", func() {
			e := &basicEvent{etype: "basic"}
			_, err := e.Target()
			Expect(err).To(MatchError("event is not complete, component has not been set"))
		})

		It("Should return the right target", func() {
			e := &basicEvent{etype: "basic", Comp: "ginkgo"}
			t, err := e.Target()
			Expect(err).ToNot(HaveOccurred())
			Expect(t).To(Equal("choria.lifecycle.event.basic.ginkgo"))
		})
	})

	Describe("String", func() {
		It("Should return the right string", func() {
			e := &basicEvent{etype: "basic", Comp: "ginkgo", Ident: "node.example"}
			Expect(e.String()).To(Equal("[basic] node.example: ginkgo"))
		})
	})

	Describe("Type", func() {
		It("Should return the right Type", func() {
			e := &basicEvent{etype: "basic", dtype: Shutdown, Comp: "ginkg", Ident: "node.example"}
			Expect(e.Type()).To(Equal(Shutdown))
		})
	})
})
