package federation

import (
	"bufio"
	"bytes"
	"context"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/protocol"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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
		logtxt      *bufio.Writer
		logbuf      *bytes.Buffer
		logger      *log.Entry
		ctx         context.Context
		cancel      func()
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
		logger, logtxt, logbuf = newDiscardLogger()

		c, err = choria.New("testdata/federation.cfg")
		Expect(err).ToNot(HaveOccurred())

		request, err = c.NewRequest(protocol.RequestV1, "test", "tester", "choria=tester", 60, c.NewRequestID(), "mcollective")
		Expect(err).ToNot(HaveOccurred())

		request.SetMessage(`{"hello":"world"}`)

		srequest, err = c.NewSecureRequest(request)
		Expect(err).ToNot(HaveOccurred())

		in.Message, err = c.NewTransportForSecureRequest(srequest)
		Expect(err).ToNot(HaveOccurred())

		broker, _ := NewFederationBroker("testing", c)

		transformer, err = NewChoriaRequestTransformer(1, 10, broker, logger)
		Expect(err).ToNot(HaveOccurred())

		go transformer.Run(ctx)
	}, 10)

	AfterEach(func() {
		cancel()
	}, 10)

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
		waitForLogLines(logtxt, logbuf)

		Expect(logbuf.String()).To(MatchRegexp("Received a message from test.example.net that is not federated"))
	})

	It("Should fail for messages with no targets", func() {
		in.Message.SetFederationRequestID("80a1ac20463745c0b12cfe6e3db61dff")
		transformer.Input() <- in

		waitForLogLines(logtxt, logbuf)

		Expect(logbuf.String()).To(MatchRegexp("Received a message 80a1ac20463745c0b12cfe6e3db61dff from test.example.net that does not have any targets"))
	})

	It("Should fail for messages with no reply-to", func() {
		in.Message.SetFederationRequestID("80a1ac20463745c0b12cfe6e3db61dff")
		in.Message.SetFederationTargets([]string{"reply.1"})

		transformer.Input() <- in
		waitForLogLines(logtxt, logbuf)

		Expect(logbuf.String()).To(MatchRegexp("Received a message 80a1ac20463745c0b12cfe6e3db61dff with no reply-to set"))
	})

	It("Should support Quit", func() {
		cancel()
		waitForLogLines(logtxt, logbuf)
		Expect(logbuf.String()).To(MatchRegexp("Worker routine choria_request_transformer exiting"))
	})
})
