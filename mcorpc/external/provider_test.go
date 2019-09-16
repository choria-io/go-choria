package external

import (
	"io/ioutil"
	"path/filepath"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-config"
	addl "github.com/choria-io/mcorpc-agent-provider/mcorpc/ddl/agent"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	logrus "github.com/sirupsen/logrus"
)

var _ = Describe("McoRPC/External", func() {
	var (
		mockctl *gomock.Controller
		fw      *MockChoriaFramework
		cfg     *config.Config
		logger  *logrus.Entry
		prov    *Provider
		// err     error
	)

	BeforeEach(func() {
		build.TLS = "false"
		logger = logrus.NewEntry(logrus.New())
		logger.Logger.Out = ioutil.Discard

		mockctl = gomock.NewController(GinkgoT())
		fw = NewMockChoriaFramework(mockctl)

		cfg = config.NewConfigForTests()
		cfg.DisableSecurityProviderVerify = true

		lib, err := filepath.Abs("testdata")
		Expect(err).ToNot(HaveOccurred())

		cfg.LibDir = []string{lib}
		fw.EXPECT().Configuration().Return(cfg).AnyTimes()

		prov = &Provider{
			cfg:    cfg,
			log:    logger,
			agents: []*addl.DDL{},
			paths:  make(map[string]string),
		}

		prov.loadAgents()
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	Describe("Agents", func() {
		It("Should return all the agent ddls", func() {
			agents := prov.Agents()
			Expect(agents).To(HaveLen(2))
			Expect(agents[0].Metadata.Name).To(Equal("echo"))
			Expect(agents[1].Metadata.Name).To(Equal("one"))
		})
	})
})
