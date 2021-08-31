package external

import (
	"path/filepath"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/config"
	imock "github.com/choria-io/go-choria/inter/imocks"
	addl "github.com/choria-io/go-choria/providers/agent/mcorpc/ddl/agent"
	"github.com/choria-io/go-choria/server"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("McoRPC/External", func() {
	var (
		mockctl *gomock.Controller
		fw      *imock.MockFramework
		cfg     *config.Config
		prov    *Provider
	)

	BeforeEach(func() {
		build.TLS = "false"

		mockctl = gomock.NewController(GinkgoT())
		fw, cfg = imock.NewFrameworkForTests(mockctl, GinkgoWriter)

		lib, err := filepath.Abs("testdata")
		Expect(err).ToNot(HaveOccurred())

		cfg.Choria.RubyLibdir = []string{lib}
		fw.EXPECT().Configuration().Return(cfg).AnyTimes()

		prov = &Provider{
			cfg:    cfg,
			log:    fw.Logger("x"),
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

	Describe("Plugin", func() {
		It("Should be a valid AgentProvider", func() {
			p := server.AgentProvider(prov)
			Expect(p).ToNot(BeNil())
		})
	})
})
