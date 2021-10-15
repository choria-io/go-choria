// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package inventory

import (
	"context"
	"testing"

	imock "github.com/choria-io/go-choria/inter/imocks"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/protocol"
)

func TestExternal(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Providers/Discovery/Inventory")
}

var _ = Describe("Inventory", func() {
	var (
		mockctl *gomock.Controller
		fw      *imock.MockFramework
		cfg     *config.Config
		inv     *Inventory
	)

	BeforeEach(func() {
		logger := logrus.New()
		logger.SetOutput(GinkgoWriter)

		mockctl = gomock.NewController(GinkgoT())
		fw, cfg = imock.NewFrameworkForTests(mockctl, GinkgoWriter)
		cfg.Choria.InventoryDiscoverySource = "testdata/good-inventory.yaml"
		inv = New(fw)
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	Describe("Discover", func() {
		It("Should resolve nodes", func() {
			filter := protocol.NewFilter()
			nodes, err := inv.Discover(context.Background(), Collective("mt_collective"), Filter(filter))
			Expect(err).To(Not(HaveOccurred()))
			Expect(nodes).To(Equal([]string{"dev1.example.net"}))

			nodes, err = inv.Discover(context.Background(), Collective("mcollective"), Filter(filter))
			Expect(err).To(Not(HaveOccurred()))
			Expect(nodes).To(Equal([]string{"dev1.example.net", "dev2.example.net"}))

			filter.AddFactFilter("country", "==", "mt")
			nodes, err = inv.Discover(context.Background(), Collective("mcollective"), Filter(filter))
			Expect(err).To(Not(HaveOccurred()))
			Expect(nodes).To(Equal([]string{"dev1.example.net"}))
		})

		It("Should resolve groups", func() {
			filter := protocol.NewFilter()
			filter.AddIdentityFilter("group:malta")

			nodes, err := inv.Discover(context.Background(), Filter(filter))
			Expect(err).To(Not(HaveOccurred()))
			Expect(nodes).To(Equal([]string{"dev1.example.net"}))

			filter = protocol.NewFilter()
			filter.AddIdentityFilter("group:all")
			nodes, err = inv.Discover(context.Background(), Filter(filter))
			Expect(err).To(Not(HaveOccurred()))
			Expect(nodes).To(Equal([]string{"dev1.example.net", "dev2.example.net"}))

			filter = protocol.NewFilter()
			filter.AddIdentityFilter("group:acme")
			nodes, err = inv.Discover(context.Background(), Filter(filter))
			Expect(err).To(Not(HaveOccurred()))
			Expect(nodes).To(Equal([]string{"dev1.example.net"}))
		})

		It("Should resolve multiple groups", func() {
			filter := protocol.NewFilter()
			filter.AddIdentityFilter("group:malta")
			filter.AddIdentityFilter("group:germany")

			nodes, err := inv.Discover(context.Background(), Filter(filter))
			Expect(err).To(Not(HaveOccurred()))
			Expect(nodes).To(Equal([]string{"dev1.example.net", "dev2.example.net"}))
		})

		It("Should match facts", func() {
			filter := protocol.NewFilter()
			filter.AddFactFilter("customer", "==", "acme")
			nodes, err := inv.Discover(context.Background(), Collective("mcollective"), Filter(filter))
			Expect(err).To(Not(HaveOccurred()))
			Expect(nodes).To(Equal([]string{"dev1.example.net"}))
		})

		It("Should match classes", func() {
			filter := protocol.NewFilter()
			filter.AddClassFilter("one")
			nodes, err := inv.Discover(context.Background(), Collective("mcollective"), Filter(filter))
			Expect(err).To(Not(HaveOccurred()))
			Expect(nodes).To(Equal([]string{"dev1.example.net", "dev2.example.net"}))

			filter.AddClassFilter("three")
			nodes, err = inv.Discover(context.Background(), Collective("mcollective"), Filter(filter))
			Expect(err).To(Not(HaveOccurred()))
			Expect(nodes).To(Equal([]string{"dev2.example.net"}))
		})

		It("Should match agents", func() {
			filter := protocol.NewFilter()
			filter.AddAgentFilter("rpcutil")
			nodes, err := inv.Discover(context.Background(), Collective("mcollective"), Filter(filter))
			Expect(err).To(Not(HaveOccurred()))
			Expect(nodes).To(Equal([]string{"dev1.example.net", "dev2.example.net"}))

			filter = protocol.NewFilter()
			filter.AddAgentFilter("rpcutil")
			filter.AddAgentFilter("other")
			nodes, err = inv.Discover(context.Background(), Collective("mcollective"), Filter(filter))
			Expect(err).To(Not(HaveOccurred()))
			Expect(nodes).To(Equal([]string{"dev2.example.net"}))
		})

		It("Should match compound filters", func() {
			filter := protocol.NewFilter()
			err := filter.AddCompoundFilter(`with("customer=acme") and with("one")`)
			Expect(err).To(Not(HaveOccurred()))
			nodes, err := inv.Discover(context.Background(), Collective("mcollective"), Filter(filter))
			Expect(err).To(Not(HaveOccurred()))
			Expect(nodes).To(Equal([]string{"dev1.example.net"}))
		})
	})
})
