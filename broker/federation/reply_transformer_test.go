package federation

import (
	"bufio"
	"bytes"

	log "github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/protocol"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Reply Transformer", func() {
	var (
		c           *choria.Framework
		request     protocol.Request
		reply       protocol.Reply
		sreply      protocol.SecureReply
		transformer *pooledWorker
		in          chainmessage
		err         error
		logtxt      *bufio.Writer
		logbuf      *bytes.Buffer
		logger      *log.Entry
	)

	BeforeEach(func() {
		logger, logtxt, logbuf = newDiscardLogger()

		c, err = choria.New("testdata/federation.cfg")
		Expect(err).ToNot(HaveOccurred())

		request, err = c.NewRequest(protocol.RequestV1, "test", "tester", "choria=tester", 60, c.NewRequestID(), "mcollective")
		Expect(err).ToNot(HaveOccurred())
		request.SetMessage(`{"hello":"world"}`)

		reply, err = c.NewReply(request)
		Expect(err).ToNot(HaveOccurred())

		sreply, err = c.NewSecureReply(reply)
		Expect(err).ToNot(HaveOccurred())

		in.Message, err = c.NewTransportForSecureReply(sreply)
		Expect(err).ToNot(HaveOccurred())

		broker, _ := NewFederationBroker("test", c)

		transformer, err = NewChoriaReplyTransformer(1, 10, broker, logger)
		Expect(err).ToNot(HaveOccurred())

		go transformer.Run()
	}, 10)

	AfterEach(func() {
		transformer.Quit()
	}, 10)

	It("should correctly transform a message", func() {
		tr, err := c.NewTransportForSecureReply(sreply)
		Expect(err).ToNot(HaveOccurred())

		tr.SetFederationRequestID(request.RequestID())
		tr.SetFederationReplyTo("mcollective.reply")

		in.Message = tr
		in.RequestID = reply.RequestID()

		transformer.Input() <- in
		out := <-transformer.Output()

		Expect(out.Targets).To(Equal([]string{"mcollective.reply"}))

		id, federated := out.Message.FederationRequestID()
		Expect(id).To(BeEmpty())
		Expect(federated).To(BeFalse())
	})

	It("should fail for unfederated messages", func() {
		transformer.Input() <- in

		waitForLogLines(logtxt, logbuf)

		Expect(logbuf.String()).To(MatchRegexp("Received a message from rip.mcollective that is not federated"))
	})

	It("Should fail for messages with no reply-to", func() {
		in.Message.SetFederationRequestID("80a1ac20463745c0b12cfe6e3db61dff")
		transformer.Input() <- in

		waitForLogLines(logtxt, logbuf)

		Expect(logbuf.String()).To(MatchRegexp("Received message 80a1ac20463745c0b12cfe6e3db61dff with no reply-to set"))
	})

	It("Should support Quit", func() {
		transformer.Quit()
		waitForLogLines(logtxt, logbuf)
		Expect(logbuf.String()).To(MatchRegexp("Worker routine choria_reply_transformer exiting"))
	})
})
