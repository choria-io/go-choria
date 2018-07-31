package protocol

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestProtocol(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Protocol")
}

var _ = Describe("Filter", func() {
	var f Filter

	It("Should support class filters", func() {
		f.AddClassFilter("testing1")
		f.AddClassFilter("testing2")
		f.AddClassFilter("testing2")
		Expect(f.ClassFilters()).To(Equal([]string{"testing1", "testing2"}))
	})

	It("Should support agent filters", func() {
		f.AddAgentFilter("agent1")
		f.AddAgentFilter("agent1")
		f.AddAgentFilter("agent2")
		Expect(f.AgentFilters()).To(Equal([]string{"agent1", "agent2"}))
	})

	It("Should support identity filters", func() {
		f.AddIdentityFilter("id1")
		f.AddIdentityFilter("id1")
		f.AddIdentityFilter("id2")
		Expect(f.IdentityFilters()).To(Equal([]string{"id1", "id2"}))
	})

	It("Should support compound filters", func() {
		err := f.AddCompoundFilter(`[{"fstatement":{"r_compare":"30","operator":">","value":"total_time","name":"resource","params":null}}]`)
		Expect(err).ToNot(HaveOccurred())
		err = f.AddCompoundFilter(`[{"statement":"environment=development"},{"or":"or"},{"statement":"customer=acme"}]`)
		Expect(err).ToNot(HaveOccurred())

		Expect(f.CompoundFilters()).To(HaveLen(2))
	})

	It("Should support fact filters", func() {
		e := f.AddFactFilter("test1", ">=", "1")
		Expect(e).ToNot(HaveOccurred())
		e = f.AddFactFilter("test2", ">=", "2")
		Expect(e).ToNot(HaveOccurred())

		e = f.AddFactFilter("test3", "foo", "3")
		Expect(e).To(HaveOccurred())

		Expect(f.FactFilters()).To(Equal([][3]string{[3]string{"test1", ">=", "1"}, [3]string{"test2", ">=", "2"}}))
	})
})
