package testutil

import (
	"context"
	"sync"
	"time"

	"github.com/choria-io/go-config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Choria", func() {
	var b *Broker
	var c *ChoriaServer
	var err error
	var cfg *config.Config

	BeforeEach(func() {
		b, err = StartBroker()
		Expect(err).NotTo(HaveOccurred())
		cfg = config.NewConfigForTests()
		cfg.DisableSecurityProviderVerify = true
		c = &ChoriaServer{broker: b, cfg: cfg, wg: &sync.WaitGroup{}}
		c.ctx, c.cancel = context.WithTimeout(context.Background(), time.Second)
	})

	Describe("ChoriaServer", func() {
		It("Should connect to the supplied broker", func() {
			Expect(b.NatsServer.NumClients()).To(Equal(0))
			err := c.Start()
			Expect(err).ToNot(HaveOccurred())

			defer b.Stop()
			defer c.Stop()

			Expect(b.NatsServer.NumClients()).To(Equal(1))
			Expect(c.Instance.ConnectedServer()).To(Equal(b.ClientURL()))
		})
	})
})
