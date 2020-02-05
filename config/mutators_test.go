package config

import (
	"io/ioutil"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

var _ = Describe("Choria/Config/Mutator", func() {
	var (
		mockctl *gomock.Controller
		log     *logrus.Entry
	)

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())
		logger := logrus.New()
		logger.SetOutput(ioutil.Discard)
		log = logrus.NewEntry(logger)
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

			m1.EXPECT().Mutate(gomock.Any(), gomock.Any()).Do(func(c *Config, _ *logrus.Entry) { c.Identity = "set_by_1" }).Times(1)
			m2.EXPECT().Mutate(gomock.Any(), gomock.Any()).Do(func(c *Config, _ *logrus.Entry) { c.LogFile = "set_by_2" }).Times(1)

			Expect(mutators).To(HaveLen(0))
			RegisterMutator("m1", m1)
			RegisterMutator("m2", m2)
			Expect(mutators).To(HaveLen(2))
			Expect(MutatorNames()).To(Equal([]string{"m1", "m2"}))

			Mutate(c, log)

			Expect(c.Identity).To(Equal("set_by_1"))
			Expect(c.LogFile).To(Equal("set_by_2"))
		})
	})
})
