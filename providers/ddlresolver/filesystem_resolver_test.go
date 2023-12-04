// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package ddlresolver

import (
	"context"
	"os"

	"github.com/choria-io/go-choria/config"
	imock "github.com/choria-io/go-choria/inter/imocks"
	agentDDL "github.com/choria-io/go-choria/providers/agent/mcorpc/ddl/agent"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("FileSystemDDLResolver", func() {
	var (
		res     *FileSystemDDLResolver
		fw      *imock.MockFramework
		cfg     *config.Config
		mockctl *gomock.Controller
	)

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())
		fw, cfg = imock.NewFrameworkForTests(mockctl, GinkgoWriter)
		res = &FileSystemDDLResolver{}
		cfg.LibDir = []string{"testdata/dir1"}
		cfg.Choria.RubyLibdir = []string{"testdata/dir2"}
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
			err := res.DDL(context.Background(), "agent", "four", ddl, fw)
			Expect(err).ToNot(HaveOccurred())
			Expect(ddl.Metadata.Name).To(Equal("four"))
		})
	})

	Describe("DDLBytes", func() {
		It("Should handle failures", func() {
			_, err := res.DDLBytes(context.Background(), "foo", "bar", fw)
			Expect(err).To(MatchError("unsupported ddl type \"foo\""))
		})

		It("Should find the correct DDL", func() {
			b, err := res.DDLBytes(context.Background(), "agent", "four", fw)
			Expect(err).ToNot(HaveOccurred())
			expected, err := os.ReadFile("testdata/dir2/mcollective/agent/four.json")
			Expect(err).ToNot(HaveOccurred())
			Expect(expected).ToNot(BeEmpty())
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
			Expect(names).To(Equal([]string{"four", "one", "three", "two"}))
		})
	})
})
