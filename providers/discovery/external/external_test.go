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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

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

	Context("command without federation", func() {
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
			wd, _ := os.Getwd()
			var f *protocol.Filter
			BeforeEach(func() {
				if runtime.GOOS == "windows" {
					Skip("not tested on windows")
				}

				f = protocol.NewFilter()
				f.AddAgentFilter("rpcutil")
				err := f.AddFactFilter("country", "==", "mt")
				Expect(err).ToNot(HaveOccurred())
			})
			It("Should request and return discovered nodes", func() {
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

				cfg.Choria.ExternalDiscoveryCommand = filepath.Join(wd, "testdata/missing.rb")
				cmd := filepath.Join(wd, "testdata/good_with_argument.rb") + " discover --test"
				nodes, err := e.Discover(context.Background(), Filter(f), DiscoveryOptions(map[string]string{"command": cmd, "foo": "bar"}))
				Expect(err).ToNot(HaveOccurred())
				Expect(nodes).To(Equal([]string{"one", "two"}))
			})
		})
	})
	Context("With federation", func() {
		BeforeEach(func() {
			mockctl = gomock.NewController(GinkgoT())
			fw, cfg = imock.NewFrameworkForTests(mockctl, GinkgoWriter, imock.WithFederations([]string{"alpha", "beta"}))
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
			wd, _ := os.Getwd()
			var f *protocol.Filter
			var err error
			BeforeEach(func() {
				if runtime.GOOS == "windows" {
					Skip("not tested on windows")
				}
				// err := os.Setenv("CHORIA_FED_COLLECTIVE", "alpha,beta")
				// Expect(err).ToNot(HaveOccurred())

				f = protocol.NewFilter()
				f.AddAgentFilter("rpcutil")
				err = f.AddFactFilter("country", "==", "mt")
				Expect(err).ToNot(HaveOccurred())
			})
			It("Should request and return discovered nodes", func() {
				cfg.Choria.ExternalDiscoveryCommand = filepath.Join(wd, "testdata/good_with_federation.rb")
				nodes, err := e.Discover(context.Background(), Filter(f), DiscoveryOptions(map[string]string{"foo": "bar"}))
				Expect(err).ToNot(HaveOccurred())
				Expect(nodes).To(Equal([]string{"one", "two"}))
			})
		})

	})
})
