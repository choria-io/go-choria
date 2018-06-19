package client

import (
	context "context"
	"encoding/base64"
	"fmt"
	"os"
	"sort"
	"sync"
	"testing"
	"time"

	choria "github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-protocol/protocol"
	gomock "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestClient(t *testing.T) {
	os.Setenv("MCOLLECTIVE_CERTNAME", "rip.mcollective")
	RegisterFailHandler(Fail)
	RunSpecs(t, "Client")
}

var _ = Describe("Client", func() {
	var (
		fw      *choria.Framework
		mockctl *gomock.Controller
		conn    *MockConnector
		err     error
		client  *Client
		mu      = &sync.Mutex{}
	)

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())
		conn = NewMockConnector(mockctl)

		cfg, _ := config.NewDefaultConfig()
		cfg.Collectives = []string{"mcollective", "test"}

		fw, _ = choria.NewWithConfig(cfg)

		client, err = New(fw, connection(conn), Timeout(100*time.Millisecond))
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	Describe("Request", func() {
		It("Should perform the request and call the handler for each reply", func() {
			seen := []string{}
			pubStarted := false
			pubEnded := false

			handler := func(ctx context.Context, m *choria.ConnectorMessage) {
				mu.Lock()
				defer mu.Unlock()

				reply, err := fw.NewTransportFromJSON(string(m.Data))
				Expect(err).ToNot(HaveOccurred())

				seen = append(seen, reply.SenderID())
			}

			pubStartCB := func() { pubStarted = true }
			pubEndCB := func() { pubEnded = true }

			OnPublishStart(pubStartCB)(client)
			OnPublishFinish(pubEndCB)(client)

			msg, err := fw.NewMessage(base64.StdEncoding.EncodeToString([]byte("ping")), "discovery", "mcollective", "request", nil)
			Expect(err).ToNot(HaveOccurred())

			msg.SetProtocolVersion(protocol.RequestV1)
			msg.SetReplyTo(choria.ReplyTarget(msg, msg.RequestID))

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			conn.EXPECT().QueueSubscribe(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), client.replies).
				AnyTimes().
				Do(func(ctx context.Context, name string, subject string, group string, output chan *choria.ConnectorMessage) {
					defer GinkgoRecover()

					Expect(name).To(Equal("replies"))
					Expect(subject).To(Equal(msg.ReplyTo()))

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
							Subject: group,
							Data:    []byte(j),
						}

						output <- cm
					}
				})

			conn.EXPECT().Publish(gomock.Any()).AnyTimes()

			err = client.Request(ctx, msg, handler)
			Expect(err).ToNot(HaveOccurred())

			Expect(pubStarted).To(BeTrue())
			Expect(pubEnded).To(BeTrue())

			sort.Strings(seen)
			Expect(seen).To(Equal([]string{"test.sender.0", "test.sender.1", "test.sender.2", "test.sender.3", "test.sender.4", "test.sender.5", "test.sender.6", "test.sender.7", "test.sender.8", "test.sender.9"}))
		})
	})
})
