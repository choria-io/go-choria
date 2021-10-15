// Copyright (c) 2017-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package discovery

import (
	"testing"

	"github.com/choria-io/go-choria/inter"
	imock "github.com/choria-io/go-choria/inter/imocks"
	v1 "github.com/choria-io/go-choria/protocol/v1"
	"github.com/golang/mock/gomock"

	"github.com/choria-io/go-choria/filter/classes"
	"github.com/choria-io/go-choria/filter/facts"
	"github.com/choria-io/go-choria/protocol"
	"github.com/choria-io/go-choria/providers/data/ddl"

	"github.com/choria-io/go-choria/config"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Server/Discovery")
}

var _ = Describe("Server/Discovery", func() {
	var (
		fw     inter.Framework
		cfg    *config.Config
		mgr    *Manager
		req    protocol.Request
		filter *protocol.Filter
		si     *MockServerInfoSource
		ctrl   *gomock.Controller
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		fw, cfg = imock.NewFrameworkForTests(ctrl, GinkgoWriter)
		si = NewMockServerInfoSource(ctrl)

		mgr = New(cfg, si, fw.Logger(""))
		rid, err := fw.NewRequestID()
		Expect(err).ToNot(HaveOccurred())

		req, err = v1.NewRequest("test", "testid", "callerid", 60, rid, "mcollective")
		Expect(err).ToNot(HaveOccurred())

		filter = req.NewFilter()
		req.SetFilter(filter)

		klasses, err := classes.ReadClasses("testdata/classes.txt")
		Expect(err).ToNot(HaveOccurred())
		factsj, err := facts.JSON("testdata/facts.yaml", fw.Logger(""))
		Expect(err).ToNot(HaveOccurred())

		si.EXPECT().Identity().Return("test.example.net").AnyTimes()
		si.EXPECT().Classes().Return(klasses).AnyTimes()
		si.EXPECT().Facts().Return(factsj).AnyTimes()
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("Should match on empty filters", func() {
		si.EXPECT().KnownAgents().Return([]string{}).AnyTimes()
		Expect(mgr.ShouldProcess(req)).To(BeTrue())
	})

	It("Should match if all filters matched", func() {
		filter.AddAgentFilter("apache")
		filter.AddClassFilter("role::testing")
		filter.AddClassFilter("/test/")
		filter.AddFactFilter("nested.string", "=~", "/hello/")
		filter.AddIdentityFilter("/test/")

		si.EXPECT().KnownAgents().Return([]string{"apache", "rpcutil"}).AnyTimes()
		Expect(mgr.ShouldProcess(req)).To(BeTrue())
	})

	It("Should fail if some filters matched", func() {
		filter.AddAgentFilter("apache")
		filter.AddClassFilter("role::test")
		filter.AddFactFilter("nested.string", "=~", "/meh/")
		si.EXPECT().KnownAgents().Return([]string{"apache", "rpcutil"}).AnyTimes()
		Expect(mgr.ShouldProcess(req)).To(BeFalse())
	})

	It("Should handle compound filters", func() {
		filter.AddCompoundFilter("with('apache') and with('/testing/') and with('fnumber=1.2') and fact('nested.string') matches('h?llo') and include(fact('sarray'), '1') and include(fact('iarray'), 1)")
		si.EXPECT().DataFuncMap().Return(ddl.FuncMap{}, nil).AnyTimes()
		si.EXPECT().KnownAgents().Return([]string{"apache", "rpcutil"}).AnyTimes()
		Expect(mgr.ShouldProcess(req)).To(BeTrue())
	})
})
