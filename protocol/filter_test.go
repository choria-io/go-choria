package protocol

import (
	"io/ioutil"
	"testing"

	gomock "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestProtocol(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Protocol")
}

var _ = Describe("Filter", func() {
	var (
		filter  *Filter
		log     *logrus.Entry
		mockctl *gomock.Controller
		request *MockRequest
		agents  []string
	)

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())

		log = logrus.NewEntry(logrus.New())
		log.Logger.Out = ioutil.Discard

		agents = []string{"apache", "rpcutil"}
		filter = NewFilter()

		request = NewMockRequest(mockctl)
		request.EXPECT().Filter().Return(filter, false).AnyTimes()
		request.EXPECT().RequestID().Return("mock.request.id").AnyTimes()
	})

	Describe("MatchRequest", func() {
		It("Should match on empty filters", func() {
			Expect(filter.MatchRequest(request, []string{}, "test.example.net", "testdata/classes.txt", "testdata/facts.yaml", log)).To(BeTrue())
		})

		It("Should match if all filters matched", func() {
			filter.AddAgentFilter("apache")
			filter.AddClassFilter("role::testing")
			filter.AddClassFilter("/test/")
			filter.AddFactFilter("nested.string", "=~", "/hello/")
			filter.AddIdentityFilter("/test/")

			Expect(filter.MatchRequest(request, agents, "test.example.net", "testdata/classes.txt", "testdata/facts.yaml", log)).To(BeTrue())
		})

		It("Should fail if some filters matched", func() {
			filter.AddAgentFilter("apache")
			filter.AddClassFilter("role::test")
			filter.AddFactFilter("nested.string", "=~", "/meh/")

			Expect(filter.MatchRequest(request, agents, "test.example.net", "testdata/classes.txt", "testdata/facts.yaml", log)).To(BeFalse())
		})
	})

	It("Should support class filters", func() {
		filter.AddClassFilter("testing1")
		filter.AddClassFilter("testing2")
		filter.AddClassFilter("testing2")
		Expect(filter.ClassFilters()).To(Equal([]string{"testing1", "testing2"}))
	})

	It("Should support agent filters", func() {
		filter.AddAgentFilter("agent1")
		filter.AddAgentFilter("agent1")
		filter.AddAgentFilter("agent2")
		Expect(filter.AgentFilters()).To(Equal([]string{"agent1", "agent2"}))
	})

	It("Should support identity filters", func() {
		filter.AddIdentityFilter("id1")
		filter.AddIdentityFilter("id1")
		filter.AddIdentityFilter("id2")
		Expect(filter.IdentityFilters()).To(Equal([]string{"id1", "id2"}))
	})

	It("Should support compound filters", func() {
		err := filter.AddCompoundFilter(`[{"fstatement":{"r_compare":"30","operator":">","value":"total_time","name":"resource","params":null}}]`)
		Expect(err).ToNot(HaveOccurred())
		err = filter.AddCompoundFilter(`[{"statement":"environment=development"},{"or":"or"},{"statement":"customer=acme"}]`)
		Expect(err).ToNot(HaveOccurred())

		Expect(filter.CompoundFilters()).To(HaveLen(2))
	})

	It("Should support fact filters", func() {
		e := filter.AddFactFilter("test1", ">=", "1")
		Expect(e).ToNot(HaveOccurred())
		e = filter.AddFactFilter("test2", ">=", "2")
		Expect(e).ToNot(HaveOccurred())

		e = filter.AddFactFilter("test3", "foo", "3")
		Expect(e).To(HaveOccurred())

		Expect(filter.FactFilters()).To(Equal([][3]string{[3]string{"test1", ">=", "1"}, [3]string{"test2", ">=", "2"}}))
	})
})
