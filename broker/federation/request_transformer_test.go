// Copyright (c) 2017-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package federation

import (
	"context"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/integration/testutil"
	"github.com/choria-io/go-choria/protocol"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	log "github.com/sirupsen/logrus"
)

var _ = Describe("RequestTransformer", func() {
	var (
		c           *choria.Framework
		request     protocol.Request
		srequest    protocol.SecureRequest
		transformer *pooledWorker
		in          chainmessage
		err         error
		logbuf      *gbytes.Buffer
		logger      *log.Entry
		ctx         context.Context
		cancel      func()
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
		var gblogger *log.Logger
		logbuf, gblogger = testutil.GbytesLogger(log.DebugLevel)
		logger = log.NewEntry(gblogger)

		c, err = choria.New("testdata/federation.cfg")
		Expect(err).ToNot(HaveOccurred())

		rid, err := c.NewRequestID()
		Expect(err).ToNot(HaveOccurred())

		request, err = c.NewRequest(protocol.RequestV1, "test", "tester", "choria=tester", 60, rid, "mcollective")
		Expect(err).ToNot(HaveOccurred())

		request.SetMessage([]byte(`{"hello":"world"}`))

		srequest, err = c.NewSecureRequest(request)
		Expect(err).ToNot(HaveOccurred())

		in.Message, err = c.NewTransportForSecureRequest(srequest)
		Expect(err).ToNot(HaveOccurred())

		broker, _ := NewFederationBroker("testing", c)

		transformer, err = NewChoriaRequestTransformer(1, 10, broker, logger)
		Expect(err).ToNot(HaveOccurred())

		go transformer.Run(ctx)
	})

	AfterEach(func() {
		cancel()
	})

	It("should correctly transform a message", func() {
		tr, err := c.NewTransportForSecureRequest(srequest)
		Expect(err).ToNot(HaveOccurred())

		tr.SetFederationRequestID(request.RequestID())
		tr.SetFederationTargets([]string{"mcollective.discovery"})
		tr.SetReplyTo("mcollective.reply")

		in.Message = tr
		in.RequestID = request.RequestID()

		transformer.Input() <- in
		out := <-transformer.Output()

		Expect(out.Message.ReplyTo()).To(Equal("choria.federation.testing.collective"))

		id, _ := out.Message.FederationRequestID()
		Expect(id).To(Equal(request.RequestID()))

		replyto, _ := out.Message.FederationReplyTo()
		Expect("mcollective.reply").To(Equal(replyto))

		targets, _ := out.Message.FederationTargets()
		Expect(targets).To(BeEmpty())
		Expect(out.Targets).To(Equal([]string{"mcollective.discovery"}))
	})

	It("should fail for unfederated messages", func() {
		transformer.Input() <- in

		Eventually(logbuf).Should(gbytes.Say("Received a message from rip.mcollective that is not federated"))
	})

	It("Should fail for messages with no targets", func() {
		in.Message.SetFederationRequestID("80a1ac20463745c0b12cfe6e3db61dff")
		transformer.Input() <- in

		Eventually(logbuf).Should(gbytes.Say("Received a message 80a1ac20463745c0b12cfe6e3db61dff from rip.mcollective that does not have any targets"))
	})

	It("Should fail for messages with no reply-to", func() {
		in.Message.SetFederationRequestID("80a1ac20463745c0b12cfe6e3db61dff")
		in.Message.SetFederationTargets([]string{"reply.1"})

		transformer.Input() <- in

		Eventually(logbuf).Should(gbytes.Say("Received a message 80a1ac20463745c0b12cfe6e3db61dff with no reply-to set"))
	})

	It("Should support Quit", func() {
		cancel()

		Eventually(logbuf).Should(gbytes.Say("Worker routine choria_request_transformer exiting"))
	})
})
