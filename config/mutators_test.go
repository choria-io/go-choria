package config

import (
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Choria/Config/Mutator", func() {
	var (
		mockctl *gomock.Controller
	)

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())
	})

	AfterEach(func() {
		mockctl.Finish()
		mutators = []Mutator{}
	})

	Describe("mutate", func() {
		It("Should call all mutators", func() {
			m1 := NewMockMutator(mockctl)
			m2 := NewMockMutator(mockctl)
			c := &Config{}

			m1.EXPECT().Mutate(gomock.Any()).Do(func(c *Config) { c.Identity = "set_by_1" }).Times(1)
			m2.EXPECT().Mutate(gomock.Any()).Do(func(c *Config) { c.LogFile = "set_by_2" }).Times(1)

			Expect(mutators).To(HaveLen(0))
			RegisterMutator("m1", m1)
			RegisterMutator("m2", m2)
			Expect(mutators).To(HaveLen(2))
			Expect(MutatorNames()).To(Equal([]string{"m1", "m2"}))

			mutate(c)

			Expect(c.Identity).To(Equal("set_by_1"))
			Expect(c.LogFile).To(Equal("set_by_2"))
		})
	})
})
