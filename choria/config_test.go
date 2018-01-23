package choria

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Choria/Config", func() {
	var _ = Describe("NewConfig", func() {
		It("Should correctly parse config files", func() {
			c, err := NewConfig("testdata/choria.cfg")
			Expect(err).ToNot(HaveOccurred())

			Expect(c.Choria.DiscoveryHost).To(Equal("pdb.example.com"))
			Expect(c.Registration).To(Equal([]string{"foo"}))
			Expect(c.RegisterInterval).To(Equal(10))
			Expect(c.RegistrationSplay).To(BeTrue())
			Expect(c.Collectives).To(Equal([]string{"c_1", "c_2", "c_3"}))
			Expect(c.MainCollective).To(Equal("c_1"))
			Expect(c.KeepLogs).To(Equal(5))
			Expect(c.LibDir).To(Equal([]string{"/dir1", "/dir2", "/dir3", "/dir4"}))
			Expect(c.DefaultDiscoveryOptions).To(Equal([]string{"one", "two"}))
			Expect(c.Choria.RandomizeMiddlewareHosts).To(BeTrue())

			Expect(c.Option("plugin.package.setting", "default")).To(Equal("1"))
			Expect(c.Option("plugin.package.other_setting", "default")).To(Equal("default"))
		})
	})
})
