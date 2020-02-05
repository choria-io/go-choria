package client

import (
	"sort"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("McoRPC/Client/Nodelist", func() {
	var (
		list *NodeList
	)

	BeforeEach(func() {
		list = NewNodeList()
	})

	Describe("AddHosts", func() {
		It("Should add all the hosts", func() {
			list.AddHosts("host1", "host2")
			Expect(list.Count()).To(Equal(2))
			list.AddHosts("host3", "host4")
			Expect(list.Count()).To(Equal(4))

			hosts := list.Hosts()
			sort.Strings(hosts)
			Expect(hosts).To(Equal(strings.Fields("host1 host2 host3 host4")))
		})
	})

	Describe("Clear", func() {
		It("Should clear the list", func() {
			list.AddHosts("host1", "host2")
			Expect(list.Count()).To(Equal(2))

			list.Clear()
			Expect(list.Count()).To(Equal(0))
		})
	})

	Describe("DeleteIfKnown", func() {
		It("Should delete correct nodes", func() {
			list.AddHosts("host1")
			deleted := list.DeleteIfKnown("host2")
			Expect(deleted).To(BeFalse())
			Expect(list.Count()).To(Equal(1))

			deleted = list.DeleteIfKnown("host1")
			Expect(deleted).To(BeTrue())
			Expect(list.Count()).To(Equal(0))
		})
	})

	Describe("Have", func() {
		It("Should find the right node", func() {
			list.AddHosts("host1", "host2")
			Expect(list.Have("host2")).To(BeTrue())
			Expect(list.Have("host3")).To(BeFalse())
		})
	})

	Describe("HaveAny", func() {
		It("Should find the right nodes", func() {
			list.AddHosts("host1", "host2")
			Expect(list.HaveAny("host1")).To(BeTrue())
			Expect(list.HaveAny("host1", "host2")).To(BeTrue())
			Expect(list.HaveAny("host3", "host2")).To(BeTrue())
			Expect(list.HaveAny("host3", "host4")).To(BeFalse())
		})

	})
})
