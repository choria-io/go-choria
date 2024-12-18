// Copyright (c) 2021-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package provtarget

import (
	"context"
	"io"
	"testing"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/providers/provtarget/builddefaults"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"
)

func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Providers/Provtarget")
}

var _ = Describe("Provision", func() {
	var (
		mockctl      *gomock.Controller
		mockresolver *MockTargetResolver
		log          *logrus.Entry
		ctx          context.Context
		cancel       func()
	)

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())
		mockresolver = NewMockTargetResolver(mockctl)
		mockresolver.EXPECT().Name().Return("Mock Resolver").AnyTimes()
		RegisterTargetResolver(builddefaults.Provider())
		ctx, cancel = context.WithCancel(context.Background())
		log = logrus.NewEntry(logrus.New())
		log.Logger.Out = io.Discard
	})

	AfterEach(func() {
		mockctl.Finish()
		cancel()
	})

	Describe("RegisterTargetResolver", func() {
		It("Should register the resolver", func() {
			Expect(Name()).To(Equal("Choria JWT Resolver"))
			RegisterTargetResolver(mockresolver)
			Expect(Name()).To(Equal("Mock Resolver"))
		})
	})

	Describe("Targets", func() {
		It("Should handle no resolver", func() {
			resolver = nil
			t, err := Targets(ctx, log)
			Expect(err).To(MatchError("no Provisioning Target Resolver registered"))
			Expect(t.Count()).To(Equal(0))
		})

		It("Should handle empty response from the resolver", func() {
			build.ProvisionBrokerURLs = ""
			t, err := Targets(ctx, log)
			Expect(err).To(MatchError("provisioning target plugin Choria JWT Resolver returned no servers"))
			Expect(t.Count()).To(Equal(0))
		})

		It("Should handle invalid format hosts", func() {
			build.ProvisionBrokerURLs = "foo,bar"
			t, err := Targets(ctx, log)
			Expect(err).To(MatchError("could not determine provisioning servers using Choria JWT Resolver provisioning target plugin: could not parse host foo: address foo: missing port in address"))
			Expect(t.Count()).To(Equal(0))
		})

		It("Should handle valid format hosts", func() {
			build.ProvisionBrokerURLs = "foo:4222, nats://bar:4222"
			t, err := Targets(ctx, log)
			Expect(err).ToNot(HaveOccurred())

			servers := t.Servers()
			Expect(servers[0].Host()).To(Equal("foo"))
			Expect(servers[1].Host()).To(Equal("bar"))
		})
	})
})
