// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package tally

import (
	imock "github.com/choria-io/go-choria/inter/imocks"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
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
				Connector: imock.NewMockConnector(ctrl),
			}
			Expect(opt.Validate()).To(Succeed())
			Expect(opt.StatPrefix).To(Equal("lifecycle_tally"))
			Expect(opt.Log).ToNot(BeNil())
		})
	})
})
