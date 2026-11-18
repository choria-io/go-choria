// Copyright (c) 2026, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package server

import (
	imock "github.com/choria-io/go-choria/inter/imocks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Server/Instance", func() {
	It("uses the configured request queue size", func() {
		ctrl := gomock.NewController(GinkgoT())
		fw, cfg := imock.NewFrameworkForTests(ctrl, GinkgoWriter)
		cfg.Choria.ServerRequestQueueSize = 42

		srv, err := NewInstance(fw)
		Expect(err).ToNot(HaveOccurred())
		Expect(cap(srv.requests)).To(Equal(42))
	})
})
