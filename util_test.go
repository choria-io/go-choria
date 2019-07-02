package srvcache

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Util", func() {
	Describe("StringHostsToServers", func() {
		It("Should produce correct servers", func() {
			servers, err := StringHostsToServers([]string{"c1:4222", "c2:4222"}, "nats")
			Expect(err).ToNot(HaveOccurred())
			Expect(servers.Count()).To(Equal(2))
			instances := servers.Servers()
			Expect(instances[0].String()).To(Equal("nats://c1:4222"))
			Expect(instances[1].String()).To(Equal("nats://c2:4222"))
		})

		It("Should return valid Servers even on error", func() {
			servers, err := StringHostsToServers([]string{"x"}, "nats")
			Expect(err).To(HaveOccurred())
			Expect(servers.Count()).To(Equal(0))

			servers, err = StringHostsToServers([]string{"x:x"}, "nats")
			Expect(err).To(HaveOccurred())
			Expect(servers.Count()).To(Equal(0))

			servers, err = StringHostsToServers([]string{"x:10"}, "")
			Expect(err).To(HaveOccurred())
			Expect(servers.Count()).To(Equal(0))
		})
	})
})
