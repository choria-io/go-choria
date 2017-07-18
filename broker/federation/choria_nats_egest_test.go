package federation

import (
	"bufio"
	"bytes"

	"github.com/choria-io/go-choria/protocol"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
)

var _ = Describe("Choria NATS Egest", func() {
	var (
		request   protocol.Request
		reply     protocol.Reply
		sreply    protocol.SecureReply
		connector *pooledWorker
		manager   *stubConnectionManager
		in        chainmessage
		err       error
		logtxt    *bufio.Writer
		logbuf    *bytes.Buffer
		logger    *log.Entry
	)

	BeforeEach(func() {
		logger, logtxt, logbuf = newDiscardLogger()

		request, err = choria.NewRequest(protocol.RequestV1, "test", "tester", "choria=tester", 60, choria.NewRequestID(), "mcollective")
		Expect(err).ToNot(HaveOccurred())
		request.SetMessage(`{"hello":"world"}`)

		reply, err = choria.NewReply(request)
		Expect(err).ToNot(HaveOccurred())

		sreply, err = choria.NewSecureReply(reply)
		Expect(err).ToNot(HaveOccurred())

		in = chainmessage{}
		in.Message, err = choria.NewTransportForSecureReply(sreply)
		Expect(err).ToNot(HaveOccurred())

		broker, _ := NewFederationBroker("test", choria)
		connector, err = NewChoriaNatsEgest(1, Unconnected, 10, broker, logger)
		Expect(err).ToNot(HaveOccurred())

		manager = &stubConnectionManager{}
		connector.connection = manager

		go connector.Run()
	}, 10)

	AfterEach(func() {
		connector.Quit()
	}, 10)

	It("Should send the message to every target", func() {
		in.RequestID = "80a1ac20463745c0b12cfe6e3db61dff"
		in.Targets = []string{"target.1", "target.2"}

		connector.in <- in

		waitForLogLines(logtxt, logbuf)
		Expect(logbuf.String()).To(MatchRegexp("Publishing message '80a1ac20463745c0b12cfe6e3db61dff' to 2 target\\(s\\)"))

		j, _ := in.Message.JSON()

		msg := <-manager.connection.Outq
		Expect(msg[0]).To(Equal("target.1"))
		Expect(msg[1]).To(Equal(j))

		msg = <-manager.connection.Outq
		Expect(msg[0]).To(Equal("target.2"))
		Expect(msg[1]).To(Equal(j))
	})

	It("Should discard messages with no targets", func() {
		in.RequestID = "80a1ac20463745c0b12cfe6e3db61dff"
		connector.in <- in

		waitForLogLines(logtxt, logbuf)
		Expect(logbuf.String()).To(MatchRegexp("Received message '80a1ac20463745c0b12cfe6e3db61dff' with no targets, discarding"))
	})

	It("Should support Quit", func() {
		connector.Quit()
		waitForLogLines(logtxt, logbuf)
		Expect(logbuf.String()).To(MatchRegexp("Worker routine choria_nats_egest exiting"))
	})
})
