package federation

import (
	"bufio"
	"bytes"

	"github.com/choria-io/go-choria/mcollective"
	"github.com/choria-io/go-choria/protocol"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
)

var _ = Describe("RequestTransformer", func() {
	var (
		choria      *mcollective.Choria
		request     protocol.Request
		srequest    protocol.SecureRequest
		transformer *pooledWorker
		in          chainmessage
		err         error
		logtxt      *bufio.Writer
		logbuf      *bytes.Buffer
		logger      *log.Entry
	)

	BeforeEach(func() {
		logger, logtxt, logbuf = newDiscardLogger()

		choria, err = mcollective.New("testdata/federation.cfg")
		Expect(err).ToNot(HaveOccurred())

		request, err = choria.NewRequest(protocol.RequestV1, "test", "tester", "choria=tester", 60, choria.NewRequestID(), "mcollective")
		Expect(err).ToNot(HaveOccurred())

		request.SetMessage(`{"hello":"world"}`)

		srequest, err = choria.NewSecureRequest(request)
		Expect(err).ToNot(HaveOccurred())

		in.Message, err = choria.NewTransportForSecureRequest(srequest)
		Expect(err).ToNot(HaveOccurred())

		broker, _ := NewFederationBroker("testing", choria)

		transformer, err = NewChoriaRequestTransformer(1, 10, broker, logger)
		Expect(err).ToNot(HaveOccurred())

		go transformer.Run()
	}, 10)

	AfterEach(func() {
		transformer.Quit()
	}, 10)

	It("should correctly transform a message", func() {
		tr, err := choria.NewTransportForSecureRequest(srequest)
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

		Expect(logbuf.String()).To(MatchRegexp("Received a message from rip.mcollective that is not federated"))
	})

	It("Should fail for messages with no targets", func() {
		in.Message.SetFederationRequestID("80a1ac20463745c0b12cfe6e3db61dff")
		transformer.Input() <- in

		waitForLogLines(logtxt, logbuf)

		Expect(logbuf.String()).To(MatchRegexp("Received a message 80a1ac20463745c0b12cfe6e3db61dff from rip.mcollective that does not have any targets"))
	})

	It("Should fail for messages with no reply-to", func() {
		in.Message.SetFederationRequestID("80a1ac20463745c0b12cfe6e3db61dff")
		in.Message.SetFederationTargets([]string{"reply.1"})

		transformer.Input() <- in
		waitForLogLines(logtxt, logbuf)

		Expect(logbuf.String()).To(MatchRegexp("Received a message 80a1ac20463745c0b12cfe6e3db61dff with no reply-to set"))
	})

	It("Should support Quit", func() {
		transformer.Quit()
		waitForLogLines(logtxt, logbuf)
		Expect(logbuf.String()).To(MatchRegexp("Worker routine choria_request_transformer exiting"))
	})
})
