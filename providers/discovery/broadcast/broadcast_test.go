// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package broadcast

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/client/client"
	"github.com/choria-io/go-choria/inter"
	imock "github.com/choria-io/go-choria/inter/imocks"
	"github.com/choria-io/go-choria/message"
	"github.com/choria-io/go-choria/protocol"
	v1 "github.com/choria-io/go-choria/protocol/v1"
	"github.com/choria-io/go-choria/providers/security/filesec"

	"github.com/golang/mock/gomock"

	"github.com/choria-io/go-choria/config"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestBroadcast(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Providers/Discovery/Broadcast")
}

var _ = Describe("Broadcast", func() {
	var (
		fw      *imock.MockFramework
		cfg     *config.Config
		mockctl *gomock.Controller
		cl      *MockChoriaClient
		b       *Broadcast
	)

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())
		cl = NewMockChoriaClient(mockctl)
		fw, cfg = imock.NewFrameworkForTests(mockctl, GinkgoWriter, imock.WithCallerID())
		cfg.Collectives = []string{"mcollective", "test"}

		fw.EXPECT().NewMessage(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(payload string, agent string, collective string, msgType string, request inter.Message) (msg inter.Message, err error) {
			return message.NewMessage(payload, agent, collective, msgType, request, fw)
		}).AnyTimes()

		sec, err := filesec.New(filesec.WithChoriaConfig(&build.Info{}, cfg), filesec.WithLog(fw.Logger("")))
		Expect(err).ToNot(HaveOccurred())

		fw.EXPECT().NewTransportFromJSON(gomock.Any()).DoAndReturn(func(data string) (message protocol.TransportMessage, err error) {
			return v1.NewTransportFromJSON(data)
		}).AnyTimes()
		fw.EXPECT().NewReplyTransportForMessage(gomock.Any(), gomock.Any()).DoAndReturn(func(msg inter.Message, request protocol.Request) (protocol.TransportMessage, error) {
			reply, err := v1.NewReply(request, cfg.Identity)
			Expect(err).ToNot(HaveOccurred())
			reply.SetMessage(msg.Payload())

			sreply, err := v1.NewSecureReply(reply, sec)
			Expect(err).ToNot(HaveOccurred())

			transport, err := v1.NewTransportMessage(cfg.Identity)
			Expect(err).ToNot(HaveOccurred())

			err = transport.SetReplyData(sreply)
			Expect(err).ToNot(HaveOccurred())

			return transport, nil
		}).AnyTimes()

		b = New(fw)
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	Describe("New", func() {
		It("Should initialize timeout to default", func() {
			Expect(b.timeout).To(Equal(2 * time.Second))
			cfg.DiscoveryTimeout = 100
			b = New(fw)
			Expect(b.timeout).To(Equal(100 * time.Second))
		})
	})

	Describe("Discover", func() {
		It("Should request and return discovered nodes", func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			f := protocol.NewFilter()
			f.AddAgentFilter("choria")

			cl.EXPECT().Request(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Do(func(ctx context.Context, msg *message.Message, handler client.Handler) {
				Expect(msg.Collective()).To(Equal("test"))
				Expect(msg.Payload()).To(Equal("cGluZw=="))

				req, err := v1.NewRequest(msg.Agent(), msg.SenderID(), msg.CallerID(), msg.TTL(), msg.RequestID(), msg.Collective())
				Expect(err).ToNot(HaveOccurred())
				req.SetMessage(msg.Payload())

				reply, err := message.NewMessageFromRequest(req, msg.ReplyTo(), fw)
				Expect(err).ToNot(HaveOccurred())

				t, err := reply.Transport()
				Expect(err).ToNot(HaveOccurred())

				for i := 0; i < 10; i++ {
					t.SetSender(fmt.Sprintf("test.sender.%d", i))

					j, err := t.JSON()
					Expect(err).ToNot(HaveOccurred())

					cm := imock.NewMockConnectorMessage(mockctl)
					cm.EXPECT().Data().Return([]byte(j))

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
