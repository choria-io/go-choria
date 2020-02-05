package broadcast

import (
	"context"
	"fmt"
	"os"
	"sort"
	"testing"
	"time"

	client "github.com/choria-io/go-client/client"
	"github.com/choria-io/go-protocol/protocol"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-config"
	gomock "github.com/golang/mock/gomock"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestBroadcast(t *testing.T) {
	os.Setenv("MCOLLECTIVE_CERTNAME", "rip.mcollective")
	RegisterFailHandler(Fail)
	RunSpecs(t, "Broadcast")
}

var _ = Describe("Broadcast", func() {
	var (
		fw      *choria.Framework
		mockctl *gomock.Controller
		cl      *MockChoriaClient
		b       *Broadcast
	)

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())
		cl = NewMockChoriaClient(mockctl)

		cfg, _ := config.NewDefaultConfig()
		cfg.Collectives = []string{"mcollective", "test"}

		fw, _ = choria.NewWithConfig(cfg)

		b = New(fw)
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	Describe("New", func() {
		It("Should intialize timeout to default", func() {
			Expect(b.timeout).To(Equal(time.Duration(2000000000)))
			fw.Config.DiscoveryTimeout = 100
			b = New(fw)
			Expect(b.timeout).To(Equal(time.Duration(100000000000)))
		})
	})

	Describe("Discover", func() {
		It("Should request and return discovered nodes", func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			f := protocol.NewFilter()
			f.AddAgentFilter("choria")

			cl.EXPECT().Request(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Do(func(ctx context.Context, msg *choria.Message, handler client.Handler) {
				Expect(msg.Collective()).To(Equal("test"))
				Expect(msg.Payload).To(Equal("cGluZw=="))

				req, err := fw.NewRequestFromMessage(protocol.RequestV1, msg)
				Expect(err).ToNot(HaveOccurred())

				reply, err := choria.NewMessageFromRequest(req, msg.ReplyTo(), fw)
				Expect(err).ToNot(HaveOccurred())

				t, err := reply.Transport()
				Expect(err).ToNot(HaveOccurred())

				for i := 0; i < 10; i++ {
					t.SetSender(fmt.Sprintf("test.sender.%d", i))

					j, err := t.JSON()
					Expect(err).ToNot(HaveOccurred())

					cm := &choria.ConnectorMessage{
						Data: []byte(j),
					}

					handler(ctx, cm)
				}
			})

			nodes, err := b.Discover(ctx, choriaClient(cl), Filter(f), Collective("test"))
			Expect(err).ToNot(HaveOccurred())
			sort.Strings(nodes)
			Expect(nodes).To(Equal([]string{"test.sender.0", "test.sender.1", "test.sender.2", "test.sender.3", "test.sender.4", "test.sender.5", "test.sender.6", "test.sender.7", "test.sender.8", "test.sender.9"}))
		})
	})
})
