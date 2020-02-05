package tally

import (
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Options", func() {
	Describe("Validate", func() {
		It("Should detect missing connectors", func() {
			opt := options{
				Component: "ginkgo",
			}
			Expect(opt.Validate()).To(MatchError("needs a connector"))
		})

		It("Should default the optionals", func() {
			ctrl := gomock.NewController(GinkgoT())
			defer ctrl.Finish()

			opt := options{
				Component: "ginkgo",
				Connector: NewMockConnector(ctrl),
			}
			Expect(opt.Validate()).To(BeNil())
			Expect(opt.StatPrefix).To(Equal("lifecycle_tally"))
			Expect(opt.Log).ToNot(BeNil())
		})
	})
})
