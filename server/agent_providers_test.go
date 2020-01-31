package server

import (
	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/choria"

	"github.com/golang/mock/gomock"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Server/AgentProviders", func() {
	var (
		mockctl  *gomock.Controller
		provider *MockAgentProvider
	)

	BeforeEach(func() {
		build.TLS = "false"

		mockctl = gomock.NewController(GinkgoT())
		provider = NewMockAgentProvider(mockctl)
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	It("Should add the provider to the list of providers", func() {
		bi := choria.BuildInfo()

		Expect(additionalAgentProviders).To(HaveLen(0))
		Expect(bi.AgentProviders()).To(BeEmpty())

		provider.EXPECT().Version().Return("Mock Provider").AnyTimes()

		RegisterAdditionalAgentProvider(provider)

		Expect(additionalAgentProviders).To(HaveLen(1))
		Expect(bi.AgentProviders()).To(HaveLen(1))
		Expect(bi.AgentProviders()[0]).To(Equal("Mock Provider"))
	})
})
