package discovery

import (
	"testing"

	"github.com/choria-io/go-choria/protocol"

	"github.com/choria-io/go-choria/choria"
	"github.com/sirupsen/logrus"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestFileContent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Server/Discovery")
}

var _ = Describe("Server/Discovery", func() {
	var fw *choria.Framework
	var log *logrus.Entry
	var err error
	var mgr *Manager
	var req protocol.Request
	var filter *protocol.Filter
	var agents []string

	BeforeSuite(func() {
		log = logrus.WithFields(logrus.Fields{"test": true})
		fw, err = choria.New("/dev/null")
		Expect(err).ToNot(HaveOccurred())

		fw.Config.ClassesFile = "testdata/classes.txt"
		fw.Config.Choria.FactSourceFile = "testdata/facts.yaml"
		fw.Config.Identity = "test.example.net"
	})

	BeforeEach(func() {
		mgr = New(fw, log)
		req, err = fw.NewRequest(protocol.RequestV1, "test", "testid", "callerid", 60, fw.NewRequestID(), "mcollective")
		Expect(err).ToNot(HaveOccurred())

		filter = req.NewFilter()
		req.SetFilter(filter)

		agents = []string{"apache", "rpcutil"}
	})

	It("Should match on empty filters", func() {
		Expect(mgr.ShouldProcess(req, []string{})).To(BeTrue())
	})

	It("Should match if all filters matched", func() {
		filter.AddAgentFilter("apache")
		filter.AddClassFilter("role::testing")
		filter.AddClassFilter("/test/")
		filter.AddFactFilter("nested.string", "=~", "/hello/")
		filter.AddIdentityFilter("/test/")

		Expect(mgr.ShouldProcess(req, agents)).To(BeTrue())
	})

	It("Should fail if some filters matched", func() {
		filter.AddAgentFilter("apache")
		filter.AddClassFilter("role::test")
		filter.AddFactFilter("nested.string", "=~", "/meh/")

		Expect(mgr.ShouldProcess(req, agents)).To(BeFalse())
	})
})
