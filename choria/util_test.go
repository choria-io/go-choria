package choria

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Util", func() {
	var _ = Describe("StringHostsToServers", func() {
		It("Should handle urls with a scheme", func() {
			s, err := StringHostsToServers([]string{"nats://c1:4222", "nats://c2:4222"}, "nats")
			Expect(err).ToNot(HaveOccurred())
			Expect(s).To(HaveLen(2))

			Expect(s[0].Host).To(Equal("c1"))
			Expect(s[0].Port).To(Equal(4222))
			Expect(s[0].Scheme).To(Equal("nats"))

			Expect(s[1].Host).To(Equal("c2"))
			Expect(s[1].Port).To(Equal(4222))
			Expect(s[1].Scheme).To(Equal("nats"))
		})

		It("Should handle host:port without a scheme and a scheme is provided", func() {
			s, err := StringHostsToServers([]string{"c1:4222", "c2:4222"}, "nats")
			Expect(err).ToNot(HaveOccurred())
			Expect(s).To(HaveLen(2))

			Expect(s[0].Host).To(Equal("c1"))
			Expect(s[0].Port).To(Equal(4222))
			Expect(s[0].Scheme).To(Equal("nats"))

			Expect(s[1].Host).To(Equal("c2"))
			Expect(s[1].Port).To(Equal(4222))
			Expect(s[1].Scheme).To(Equal("nats"))
		})

		It("Should handle full urls with an override scheme", func() {
			s, err := StringHostsToServers([]string{"foo://c1:4222", "foo://c2:4222"}, "nats")
			Expect(err).ToNot(HaveOccurred())
			Expect(s).To(HaveLen(2))

			Expect(s[0].Host).To(Equal("c1"))
			Expect(s[0].Port).To(Equal(4222))
			Expect(s[0].Scheme).To(Equal("nats"))

			Expect(s[1].Host).To(Equal("c2"))
			Expect(s[1].Port).To(Equal(4222))
			Expect(s[1].Scheme).To(Equal("nats"))
		})

		It("Should handle full urls without an override scheme", func() {
			s, err := StringHostsToServers([]string{"foo://c1:4222", "foo://c2:4222"}, "")
			Expect(err).ToNot(HaveOccurred())
			Expect(s).To(HaveLen(2))

			Expect(s[0].Host).To(Equal("c1"))
			Expect(s[0].Port).To(Equal(4222))
			Expect(s[0].Scheme).To(Equal("foo"))

			Expect(s[1].Host).To(Equal("c2"))
			Expect(s[1].Port).To(Equal(4222))
			Expect(s[1].Scheme).To(Equal("foo"))
		})
	})
})
