package events

import (
	"os"
	"testing"

	"github.com/choria-io/go-choria/config"
	gomock "github.com/golang/mock/gomock"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestChoria(t *testing.T) {
	os.Setenv("MCOLLECTIVE_CERTNAME", "rip.mcollective")
	RegisterFailHandler(Fail)
	RunSpecs(t, "Events")
}

var _ = Describe("Events", func() {
	var (
		mockctl *gomock.Controller
		conn    *MockPublishConnector
		cfg     *config.Config
		err     error
	)

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())
		conn = NewMockPublishConnector(mockctl)
		cfg, err = config.NewDefaultConfig()
		cfg.Identity = "test.ginkgo"
		Expect(err).ToNot(HaveOccurred())
		mockTime = 1535106973
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	Describe("PublishEvent", func() {
		It("Should support startup events", func() {
			conn.EXPECT().PublishRaw("choria.lifecycle.event", gomock.Any()).Return(nil).Do(func(dest string, body []byte) {
				Expect(dest).To(Equal("choria.lifecycle.event"))
				Expect(body).To(MatchJSON(`{
					"protocol": "choria:lifecycle:startup:1",
					"identity": "test.ginkgo",
					"version": "0.5.1",
					"timestamp": 1535106973,
					"component": "ginkgo"
				  }`))
			})

			err := PublishEvent(Startup, "ginkgo", cfg, conn)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("newStartupEvent", func() {
		It("Should create valid events", func() {
			e := newStartupEvent("test.ident", "ginkgo")
			Expect(e.Component).To(Equal("ginkgo"))
			Expect(e.Identity).To(Equal("test.ident"))
			Expect(e.Protocol).To(Equal("choria:lifecycle:startup:1"))
		})
	})
})
