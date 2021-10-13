// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package ddlresolver

import (
	"context"

	imock "github.com/choria-io/go-choria/inter/imocks"
	"github.com/choria-io/go-choria/internal/fs"
	agentDDL "github.com/choria-io/go-choria/providers/agent/mcorpc/ddl/agent"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("InternalCachedDDLResolver", func() {
	var (
		res     *InternalCachedDDLResolver
		fw      *imock.MockFramework
		mockctl *gomock.Controller
	)

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())
		fw, _ = imock.NewFrameworkForTests(mockctl, GinkgoWriter)
		res = &InternalCachedDDLResolver{}
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	Describe("DDL", func() {
		It("Should handle failures", func() {
			err := res.DDL(context.Background(), "foo", "bar", nil, fw)
			Expect(err).To(MatchError("unsupported ddl type \"foo\""))
		})

		It("Should correctly find and unmarshal into target", func() {
			ddl := &agentDDL.DDL{}
			err := res.DDL(context.Background(), "agent", "choria_util", ddl, fw)
			Expect(err).ToNot(HaveOccurred())
			Expect(ddl.Metadata.Name).To(Equal("choria_util"))
		})
	})

	Describe("DDLBytes", func() {
		It("Should handle failures", func() {
			_, err := res.DDLBytes(context.Background(), "foo", "bar", fw)
			Expect(err).To(MatchError("unsupported ddl type \"foo\""))
		})

		It("Should find the correct DDL", func() {
			b, err := res.DDLBytes(context.Background(), "agent", "choria_util", fw)
			Expect(err).ToNot(HaveOccurred())
			expected, err := fs.FS.ReadFile("ddl/cache/agent/choria_util.json")
			Expect(err).ToNot(HaveOccurred())
			Expect(expected).ToNot(HaveLen(0))
			Expect(b).To(Equal(expected))
		})
	})

	Describe("DDLNames", func() {
		It("Should handle failures", func() {
			_, err := res.DDLBytes(context.Background(), "foo", "bar", fw)
			Expect(err).To(MatchError("unsupported ddl type \"foo\""))
		})

		It("Should find the correct names", func() {
			names, err := res.DDLNames(context.Background(), "agent", fw)
			Expect(err).ToNot(HaveOccurred())
			Expect(names).To(Equal([]string{"aaa_signer", "choria_provision", "choria_registry", "choria_util", "rpcutil", "scout"}))
		})
	})
})
