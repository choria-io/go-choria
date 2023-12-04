// Copyright (c) 2018-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/internal/util"

	"github.com/golang/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
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
		bi := util.BuildInfo()

		Expect(additionalAgentProviders).To(BeEmpty())
		Expect(bi.AgentProviders()).To(BeEmpty())

		provider.EXPECT().Version().Return("Mock Provider").AnyTimes()

		RegisterAdditionalAgentProvider(provider)

		Expect(additionalAgentProviders).To(HaveLen(1))
		Expect(bi.AgentProviders()).To(HaveLen(1))
		Expect(bi.AgentProviders()[0]).To(Equal("Mock Provider"))
	})
})
