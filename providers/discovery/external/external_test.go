// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package external

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	imock "github.com/choria-io/go-choria/inter/imocks"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/protocol"
)

func TestExternal(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Providers/Discovery/External")
}

var _ = Describe("External", func() {
	var (
		mockctl *gomock.Controller
		fw      *imock.MockFramework
		cfg     *config.Config
		e       *External
	)

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())
		fw, cfg = imock.NewFrameworkForTests(mockctl, GinkgoWriter)
		cfg.Collectives = []string{"mcollective", "test"}

		e = New(fw)
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	Describe("New", func() {
		It("Should initialize timeout to default", func() {
			Expect(e.timeout).To(Equal(2 * time.Second))
			cfg.DiscoveryTimeout = 100
			e = New(fw)
			Expect(e.timeout).To(Equal(100 * time.Second))
		})
	})

	Describe("Discover", func() {
		It("Should request and return discovered nodes", func() {
			if runtime.GOOS == "windows" {
				Skip("not tested on windows")
			}

			f := protocol.NewFilter()
			f.AddAgentFilter("rpcutil")
			f.AddFactFilter("country", "==", "mt")

			wd, _ := os.Getwd()
			cfg.Choria.ExternalDiscoveryCommand = filepath.Join(wd, "testdata/good.rb")
			nodes, err := e.Discover(context.Background(), Filter(f), DiscoveryOptions(map[string]string{"foo": "bar"}))
			Expect(err).ToNot(HaveOccurred())
			Expect(nodes).To(Equal([]string{"one", "two"}))

			cfg.Choria.ExternalDiscoveryCommand = filepath.Join(wd, "testdata/good_with_argument.rb") + " discover --test"
			nodes, err = e.Discover(context.Background(), Filter(f), DiscoveryOptions(map[string]string{"foo": "bar"}))
			Expect(err).ToNot(HaveOccurred())
			Expect(nodes).To(Equal([]string{"one", "two"}))
		})

		It("Should support command overrides via options", func() {
			if runtime.GOOS == "windows" {
				Skip("not tested on windows")
			}

			f := protocol.NewFilter()
			f.AddAgentFilter("rpcutil")
			f.AddFactFilter("country", "==", "mt")

			wd, _ := os.Getwd()
			cfg.Choria.ExternalDiscoveryCommand = filepath.Join(wd, "testdata/missing.rb")
			cmd := filepath.Join(wd, "testdata/good_with_argument.rb") + " discover --test"
			nodes, err := e.Discover(context.Background(), Filter(f), DiscoveryOptions(map[string]string{"command": cmd, "foo": "bar"}))
			Expect(err).ToNot(HaveOccurred())
			Expect(nodes).To(Equal([]string{"one", "two"}))
		})
	})
})
