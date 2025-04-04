// Copyright (c) 2020-2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/choria-io/go-choria/inter"
	imock "github.com/choria-io/go-choria/inter/imocks"
	"github.com/choria-io/go-choria/message"
	v1 "github.com/choria-io/go-choria/protocol/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

func TestClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Client/Client")
}

var _ = Describe("Client", func() {
	var (
		fw      *imock.MockFramework
		mockctl *gomock.Controller
		conn    *imock.MockConnector
		err     error
		client  *Client
		mu      = &sync.Mutex{}
	)

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())
		conn = imock.NewMockConnector(mockctl)

		fw, _ = imock.NewFrameworkForTests(mockctl, GinkgoWriter, imock.WithCallerID())
		fw.Configuration().Collectives = []string{"mcollective", "test"}

		client, err = New(fw, Connection(conn), Timeout(100*time.Millisecond), Name("test"))
		Expect(err).ToNot(HaveOccurred())
		Expect(client.name).To(Equal("test"))
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	Describe("Request", func() {
		It("Should support fire and forget requests", func(ctx context.Context) {
			pubStarted := false
			pubEnded := false

			pubStartCB := func() { pubStarted = true }
			pubEndCB := func() { pubEnded = true }

			OnPublishStart(pubStartCB)(client)
			OnPublishFinish(pubEndCB)(client)

			ping := base64.StdEncoding.EncodeToString([]byte("ping"))
			msg, err := message.NewMessage([]byte(ping), "discovery", "mcollective", "request", nil, fw)
			Expect(err).ToNot(HaveOccurred())
			msg.SetReplyTo("custom")

			conn.EXPECT().Publish(gomock.Any()).AnyTimes()

			err = client.Request(ctx, msg, nil)
			Expect(err).ToNot(HaveOccurred())

			Expect(pubStarted).To(BeTrue())
			Expect(pubEnded).To(BeTrue())
		})

		It("Should perform the request and call the handler for each reply", func(ctx context.Context) {
			seen := []string{}
			pubStarted := false
			pubEnded := false

			handler := func(ctx context.Context, m inter.ConnectorMessage) {
				mu.Lock()
				defer mu.Unlock()

				reply, err := v1.NewTransportFromJSON(m.Data())
				Expect(err).ToNot(HaveOccurred())

				seen = append(seen, reply.SenderID())
			}

			pubStartCB := func() { pubStarted = true }
			pubEndCB := func() { pubEnded = true }

			OnPublishStart(pubStartCB)(client)
			OnPublishFinish(pubEndCB)(client)

			ping := base64.StdEncoding.EncodeToString([]byte("ping"))
			msg, err := message.NewMessage([]byte(ping), "discovery", "mcollective", "request", nil, fw)
			Expect(err).ToNot(HaveOccurred())

			msg.SetReplyTo(fmt.Sprintf("%s.reply.%s.%s", msg.Collective(), fmt.Sprintf("%x", md5.Sum([]byte(msg.CallerID()))), msg.RequestID()))

			conn.EXPECT().QueueSubscribe(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), client.replies).
				AnyTimes().
				Do(func(ctx context.Context, name string, subject string, group string, output chan inter.ConnectorMessage) {
					defer GinkgoRecover()

					Expect(name).To(Equal("replies"))
					Expect(subject).To(Equal(msg.ReplyTo()))
					for i := 0; i < 10; i++ {
						t, err := v1.NewTransportMessage(fmt.Sprintf("test.sender.%d", i))
						Expect(err).ToNot(HaveOccurred())

						j, err := t.JSON()
						Expect(err).ToNot(HaveOccurred())

						cm := imock.NewMockConnectorMessage(mockctl)
						cm.EXPECT().Data().Return([]byte(j)).AnyTimes()

						output <- cm
					}
				})
			//
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
