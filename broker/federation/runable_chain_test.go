package federation

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type plug struct {
	chainbase
}

type socket struct {
	chainbase
}

func (self *plug) Init(name string) {
	self.in = make(chan chainmessage, 1000)
	self.out = make(chan chainmessage, 1000)
	self.name = name
	self.initialized = true
}

func (self *socket) Init(name string) {
	self.in = make(chan chainmessage, 1000)
	self.out = make(chan chainmessage, 1000)
	self.name = name
	self.initialized = true
}

var _ = Describe("Runable Chain", func() {
	It("Should correctly initialize", func() {
		s := socket{}
		Expect(s.Input()).To(BeNil())
		Expect(s.Output()).To(BeNil())
		Expect(s.Ready()).To(BeFalse())

		s.Init("socket")
		Expect(s.Name()).To(Equal("socket"))
		Expect(s.Input()).To(HaveCap(1000))
		Expect(s.Output()).To(HaveCap(1000))
		Expect(s.Ready()).To(BeTrue())
	})

	It("Should correctly plug chainables into each other", func() {
		s := socket{}
		s.Init("socket")

		l := plug{}
		l.Init("left")

		r := plug{}
		r.Init("right")

		Expect(l.Output()).ToNot(Equal(s.Input()))
		s.From(&l)
		Expect(l.Output()).To(Equal(s.Input()))

		Expect(r.Input()).ToNot(Equal(s.Output()))
		s.To(&r)
		Expect(r.Input()).To(Equal(s.Output()))
	})
})
