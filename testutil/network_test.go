package testutil

import (
	"github.com/choria-io/go-config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)


var _ = Describe("Network", func() {
	var n *ChoriaNetwork

	BeforeEach(func() {
		n = &ChoriaNetwork{cfg: config.NewConfigForTests()}
		n.cfg.DisableTLS = true
	})

	It("Should start a networks", func() {
		err := n.Start()
		Expect(err).ToNot(HaveOccurred())
		defer n.Stop()

		Expect(n.ServerInstance().ConnectedServer()).To(Equal(n.broker.ClientURL()))
	})
})
