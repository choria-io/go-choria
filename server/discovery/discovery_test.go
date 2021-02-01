package discovery

import (
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/choria-io/go-choria/filter/classes"
	"github.com/choria-io/go-choria/filter/facts"
	"github.com/choria-io/go-choria/protocol"
	"github.com/choria-io/go-choria/providers/data/ddl"

	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/config"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Server/Discovery")
}

var _ = Describe("Server/Discovery", func() {
	var (
		fw     *choria.Framework
		log    *logrus.Entry
		err    error
		mgr    *Manager
		req    protocol.Request
		filter *protocol.Filter
		si     *MockServerInfoSource
		ctrl   *gomock.Controller
	)

	BeforeSuite(func() {
		log = logrus.WithFields(logrus.Fields{"test": true})
		cfg := config.NewConfigForTests()
		cfg.DisableTLS = true

		fw, err = choria.NewWithConfig(cfg)
		Expect(err).ToNot(HaveOccurred())

		fw.Config.Identity = "test.example.net"
	})

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		si = NewMockServerInfoSource(ctrl)
		mgr = New(fw, si, log)
		rid, err := fw.NewRequestID()
		Expect(err).ToNot(HaveOccurred())

		req, err = fw.NewRequest(protocol.RequestV1, "test", "testid", "callerid", 60, rid, "mcollective")
		Expect(err).ToNot(HaveOccurred())

		filter = req.NewFilter()
		req.SetFilter(filter)

		klasses, err := classes.ReadClasses("testdata/classes.txt")
		Expect(err).ToNot(HaveOccurred())
		factsj, err := facts.JSON("testdata/facts.yaml", log)
		Expect(err).ToNot(HaveOccurred())

		si.EXPECT().Identity().Return("test.example.net").AnyTimes()
		si.EXPECT().Classes().Return(klasses).AnyTimes()
		si.EXPECT().Facts().Return(factsj).AnyTimes()
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("Should match on empty filters", func() {
		si.EXPECT().KnownAgents().Return([]string{}).AnyTimes()
		Expect(mgr.ShouldProcess(req)).To(BeTrue())
	})

	It("Should match if all filters matched", func() {
		filter.AddAgentFilter("apache")
		filter.AddClassFilter("role::testing")
		filter.AddClassFilter("/test/")
		filter.AddFactFilter("nested.string", "=~", "/hello/")
		filter.AddIdentityFilter("/test/")

		si.EXPECT().KnownAgents().Return([]string{"apache", "rpcutil"}).AnyTimes()
		Expect(mgr.ShouldProcess(req)).To(BeTrue())
	})

	It("Should fail if some filters matched", func() {
		filter.AddAgentFilter("apache")
		filter.AddClassFilter("role::test")
		filter.AddFactFilter("nested.string", "=~", "/meh/")
		si.EXPECT().KnownAgents().Return([]string{"apache", "rpcutil"}).AnyTimes()
		Expect(mgr.ShouldProcess(req)).To(BeFalse())
	})

	It("Should handle compound filters", func() {
		filter.AddCompoundFilter("with('apache') and with('/testing/') and with('fnumber=1.2') and fact('nested.string') matches('h?llo') and include(fact('sarray'), '1') and include(fact('iarray'), 1)")
		si.EXPECT().DataFuncMap().Return(ddl.FuncMap{}, nil).AnyTimes()
		si.EXPECT().KnownAgents().Return([]string{"apache", "rpcutil"}).AnyTimes()
		Expect(mgr.ShouldProcess(req)).To(BeTrue())
	})
})
