package external

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/protocol"
)

func TestExternal(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Client/Discovery/External")
}

var _ = Describe("Broadcast", func() {
	var (
		fw *choria.Framework
		e  *External
	)

	BeforeEach(func() {
		cfg := config.NewConfigForTests()
		cfg.Collectives = []string{"mcollective", "test"}

		fw, _ = choria.NewWithConfig(cfg)

		e = New(fw)
	})

	Describe("New", func() {
		It("Should initialize timeout to default", func() {
			Expect(e.timeout).To(Equal(2 * time.Second))
			fw.Config.DiscoveryTimeout = 100
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
			fw.Config.Choria.ExternalDiscoveryCommand = filepath.Join(wd, "testdata/good.rb")
			nodes, err := e.Discover(context.Background(), Filter(f))
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
			fw.Config.Choria.ExternalDiscoveryCommand = filepath.Join(wd, "testdata/missing.rb")
			cmd := filepath.Join(wd, "testdata/good.rb")
			nodes, err := e.Discover(context.Background(), Filter(f), DiscoveryOptions(map[string]string{"command": cmd}))
			Expect(err).ToNot(HaveOccurred())
			Expect(nodes).To(Equal([]string{"one", "two"}))

		})
	})
})
