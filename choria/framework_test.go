package choria

import (
	"testing"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestChoria(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Choria Framework")
}

var _ = Describe("Choria", func() {
	var _ = Describe("NewChoria", func() {
		It("Should initialize choria correctly", func() {
			cfg, _ := config.NewDefaultConfig()
			c := cfg.Choria
			Expect(c.DiscoveryHost).To(Equal("puppet"))
			Expect(c.DiscoveryPort).To(Equal(8085))
			Expect(c.UseSRVRecords).To(BeTrue())
		})
	})

	var _ = Describe("ProvisionMode", func() {
		It("Should use the default when not configured and brokers are compiled in", func() {
			c, err := config.NewDefaultConfig()
			Expect(err).ToNot(HaveOccurred())
			c.DisableTLS = true

			fw, err := NewWithConfig(c)
			Expect(err).ToNot(HaveOccurred())

			Expect(fw.ProvisionMode()).To(Equal(false))

			build.ProvisionBrokerURLs = "nats://n1:4222"
			build.ProvisionModeDefault = "true"
			Expect(fw.ProvisionMode()).To(Equal(true))
		})

		It("Should use the configured value when set and when brokers are compiled in", func() {
			c, err := config.NewConfig("testdata/provision.cfg")
			Expect(err).ToNot(HaveOccurred())
			c.DisableTLS = true

			fw, err := NewWithConfig(c)
			Expect(err).ToNot(HaveOccurred())

			build.ProvisionBrokerURLs = "nats://n1:4222"

			Expect(fw.ProvisionMode()).To(Equal(true))

			c.Choria.Provision = false
			build.ProvisionModeDefault = "true"

			Expect(fw.ProvisionMode()).To(Equal(false))
		})

		It("Should be false if there are no brokers", func() {
			c, err := config.NewConfig("testdata/provision.cfg")
			Expect(err).ToNot(HaveOccurred())
			c.DisableTLS = true

			fw, err := NewWithConfig(c)
			Expect(err).ToNot(HaveOccurred())

			build.ProvisionBrokerURLs = ""
			Expect(fw.ProvisionMode()).To(Equal(false))
		})
	})
})
