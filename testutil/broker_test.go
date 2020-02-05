package testutil

import (
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestChoria(t *testing.T) {
	os.Setenv("MCOLLECTIVE_CERTNAME", "rip.mcollective")
	RegisterFailHandler(Fail)
	RunSpecs(t, "Choria Test Utils")
}

var _ = Describe("Broker", func() {
	var b *Broker

	BeforeEach(func() {
		b = &Broker{}
	})

	It("Should start a broker", func() {
		err := b.Start()
		Expect(err).NotTo(HaveOccurred())
		defer b.Stop()

		Expect(b.ClientURL()).ToNot(Equal(""))
		Expect(b.ClientURL()).To(Equal(b.NatsServer.ClientURL()))
		Expect(b.NatsServer.ReadyForConnections(time.Second)).To(BeTrue())
	})

	It("Should not start twice", func() {
		err := b.Start()
		Expect(err).NotTo(HaveOccurred())
		defer b.Stop()

		err = b.Start()
		Expect(err).NotTo(MatchError(" broker already exist, cannot start again"))
	})

	It("Should cleanly stop the server", func() {
		err := b.Start()
		Expect(err).NotTo(HaveOccurred())
		b.Stop()
		Expect(b.NatsServer).To(BeNil())
	})

	It("Should be empty client url when not running", func() {
		Expect(b.ClientURL()).To(Equal(""))
		err := b.Start()
		Expect(err).NotTo(HaveOccurred())
		b.NatsServer.Shutdown()
		Expect(b.NatsServer.Addr()).To(BeNil())
	})
})
