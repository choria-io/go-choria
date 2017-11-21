package federation

import (
	"bufio"
	"bytes"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/protocol"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
)

var _ = Describe("Choria NATS Ingest", func() {
	var (
		request   protocol.Request
		srequest  protocol.SecureRequest
		transport protocol.TransportMessage
		connector *pooledWorker
		manager   *stubConnectionManager
		in        *choria.ConnectorMessage
		err       error
		logtxt    *bufio.Writer
		logbuf    *bytes.Buffer
		logger    *log.Entry
		broker    *FederationBroker
	)

	BeforeEach(func() {
		logger, logtxt, logbuf = newDiscardLogger()

		request, err = c.NewRequest(protocol.RequestV1, "test", "tester", "choria=tester", 60, c.NewRequestID(), "mcollective")
		Expect(err).ToNot(HaveOccurred())
		request.SetMessage(`{"hello":"world"}`)

		srequest, err = c.NewSecureRequest(request)
		Expect(err).ToNot(HaveOccurred())

		transport, err = c.NewTransportForSecureRequest(srequest)
		Expect(err).ToNot(HaveOccurred())
		transport.SetFederationRequestID(request.RequestID())

		j, err := transport.JSON()
		Expect(err).ToNot(HaveOccurred())

		in = &choria.ConnectorMessage{
			Data:    []byte(j),
			Subject: "test",
		}

		broker, _ = NewFederationBroker("test", c)
		connector, err = NewChoriaNatsIngest(1, Federation, 10, broker, logger)
		Expect(err).ToNot(HaveOccurred())

		manager = &stubConnectionManager{}
		manager.Init()
		connector.connection = manager

		go connector.Run()
	}, 1)

	AfterEach(func() {
		connector.Quit()
	}, 1)

	It("Should fail for invalid JSON", func() {
		in.Data = []byte("{}")
		manager.connection.PublishToQueueSub("ingest", in)
		waitForLogLines(logtxt, logbuf)
		Expect(logbuf.String()).To(MatchRegexp("Could not parse received message into a TransportMessage: Do not know how to create a TransportMessage from an expected JSON format message with content: {}"))
	})

	It("Should fail for unfederated messages", func() {
		transport.SetUnfederated()
		j, _ := transport.JSON()
		in.Data = []byte(j)
		manager.connection.PublishToQueueSub("ingest", in)
		waitForLogLines(logtxt, logbuf)
		Expect(logbuf.String()).To(MatchRegexp("Received a message on test that was not federated"))

	})

	It("Should subscribe to the right target in Federation mode", func() {
		manager.connection.PublishToQueueSub("ingest", in)
		<-connector.Output()
		Expect(manager.connection.Subs["ingest"]).To(Equal([3]string{"ingest", "choria.federation.test.federation", "test_federation"}))

	})

	It("Should subscribe to the right target in Collective mode", func() {
		connector.Quit()

		connector, _ := NewChoriaNatsIngest(1, Collective, 10, broker, logger)
		manager := &stubConnectionManager{}
		manager.Init()
		connector.connection = manager

		go connector.Run()

		manager.connection.PublishToQueueSub("ingest", in)
		<-connector.Output()
		Expect(manager.connection.Subs["ingest"]).To(Equal([3]string{"ingest", "choria.federation.test.collective", "test_collective"}))

		connector.Quit()
	})

	It("Should subscribe and process the message", func() {
		manager.connection.PublishToQueueSub("ingest", in)
		out := <-connector.Output()

		Expect(out.Message).To(Equal(transport))
		Expect(out.RequestID).To(Equal(request.RequestID()))
		Expect(out.Seen).To(Equal([]string{"nats://stub:4222", "choria_nats_ingest"}))
	}, 1)

})
