// Copyright (c) 2017-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package federation

import (
	"context"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/integration/testutil"
	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/protocol"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	log "github.com/sirupsen/logrus"
)

var _ = Describe("Choria NATS Ingest", func() {
	var (
		request   protocol.Request
		srequest  protocol.SecureRequest
		transport protocol.TransportMessage
		connector *pooledWorker
		manager   *stubConnectionManager
		in        inter.ConnectorMessage
		logbuf    *gbytes.Buffer
		logger    *log.Entry
		broker    *FederationBroker
		ctx       context.Context
		cancel    func()
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
		var gblogger *log.Logger
		logbuf, gblogger = testutil.GbytesLogger(log.DebugLevel)
		logger = log.NewEntry(gblogger)

		c.Config.OverrideCertname = "rip.mcollective"

		rid, err := c.NewRequestID()
		Expect(err).ToNot(HaveOccurred())

		request, err = c.NewRequest(protocol.RequestV1, "test", "tester", "choria=tester", 60, rid, "mcollective")
		Expect(err).ToNot(HaveOccurred())
		request.SetMessage([]byte(`{"hello":"world"}`))

		srequest, err = c.NewSecureRequest(ctx, request)
		Expect(err).ToNot(HaveOccurred())

		transport, err = c.NewTransportForSecureRequest(srequest)
		Expect(err).ToNot(HaveOccurred())
		transport.SetFederationRequestID(request.RequestID())

		j, err := transport.JSON()
		Expect(err).ToNot(HaveOccurred())

		in = choria.NewConnectorMessage("test", "", []byte(j), nil)

		broker, _ = NewFederationBroker("test", c)
		connector, err = NewChoriaNatsIngest(1, Federation, 10, broker, logger)
		Expect(err).ToNot(HaveOccurred())

		manager = &stubConnectionManager{}
		manager.Init()
		connector.connection = manager

		connector.choria.Configuration().Choria.FederationMiddlewareHosts = []string{"c1:4222", "c2:4222"}
		connector.choria.Configuration().Choria.MiddlewareHosts = []string{"c3:4222", "c4:4222"}

		go connector.Run(ctx)
	})

	AfterEach(func() {
		cancel()
	})

	It("Should fail for invalid JSON", func() {
		in = choria.NewConnectorMessage(in.Subject(), in.Reply(), []byte("{}"), nil)
		manager.connection.PublishToQueueSub("ingest", in)
		Eventually(logbuf).Should(gbytes.Say("Could not parse received message into a TransportMessage: do not know how to create a TransportMessage from an expected JSON format message with content: {}"))
	})

	It("Should fail for unfederated messages", func() {
		transport.SetUnfederated()
		j, _ := transport.JSON()
		in = choria.NewConnectorMessage(in.Subject(), in.Reply(), []byte(j), nil)
		manager.connection.PublishToQueueSub("ingest", in)
		Eventually(logbuf).Should(gbytes.Say("Received a message on test that was not federated"))
	})

	It("Should subscribe to the right target in Federation mode", func() {
		manager.connection.PublishToQueueSub("ingest", in)
		<-connector.Output()
		Expect(manager.connection.Subs["ingest"]).To(Equal([3]string{"ingest", "choria.federation.test.federation", "test_federation"}))

	})

	It("Should subscribe to the right target in Collective mode", func() {
		cancel()
		ctx, cancel = context.WithCancel(context.Background())

		connector, _ := NewChoriaNatsIngest(1, Collective, 10, broker, logger)
		manager := &stubConnectionManager{}
		manager.Init()
		connector.connection = manager

		go connector.Run(ctx)

		manager.connection.PublishToQueueSub("ingest", in)
		<-connector.Output()
		Expect(manager.connection.Subs["ingest"]).To(Equal([3]string{"ingest", "choria.federation.test.collective", "test_collective"}))

		cancel()
	})

	It("Should subscribe and process the message", func() {
		manager.connection.PublishToQueueSub("ingest", in)
		out := <-connector.Output()

		Expect(out.Message).To(Equal(transport))
		Expect(out.RequestID).To(Equal(request.RequestID()))
		Expect(out.Seen).To(Equal([]string{"nats://stub:4222", "choria_nats_ingest:0"}))
	})
})
