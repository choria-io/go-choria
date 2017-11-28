package server

import (
	"errors"

	"github.com/choria-io/go-choria/choria"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
)

type StubRegistrator struct {
	Err error
	Dat *[]byte
}

func (sr *StubRegistrator) RegistrationData() (*[]byte, error) {
	return sr.Dat, sr.Err
}

var _ = Describe("pollAndPublish", func() {
	var (
		conn   StubPublishingConnector
		reg    StubRegistrator
		server *Instance
	)

	BeforeEach(func() {
		conn = StubPublishingConnector{}
		reg = StubRegistrator{}

		choria, err := choria.New("/dev/null")
		choria.Config.Collectives = []string{"test_collective"}
		choria.Config.MainCollective = "test_collective"
		choria.Config.RegistrationCollective = "test_collective"
		choria.Config.Identity = "test.example.net"

		Expect(err).ToNot(HaveOccurred())
		server, err = NewInstance(choria)
		Expect(err).ToNot(HaveOccurred())

		log.SetLevel(log.FatalLevel)
	})

	It("Should do nothing when the RegistrationData poll failed", func() {
		reg.Err = errors.New("Simulated error")
		server.pollAndPublish(&reg, &conn)
		Expect(conn.PublishedMsgs).To(BeEmpty())
	})

	It("Should do nothing for nil data", func() {
		reg.Dat = nil
		server.pollAndPublish(&reg, &conn)
		Expect(conn.PublishedMsgs).To(BeEmpty())
	})

	It("Should do nothing for empty data", func() {
		reg.Dat = &[]byte{}
		server.pollAndPublish(&reg, &conn)
		Expect(conn.PublishedMsgs).To(BeEmpty())
	})

	It("Should publish a message for the discovery agent when it finds data", func() {
		dat := []byte("hello world")
		reg.Dat = &dat

		server.pollAndPublish(&reg, &conn)
		Expect(conn.PublishedMsgs).ToNot(BeEmpty())

		msg := conn.PublishedMsgs[0]
		Expect(msg.Agent).To(Equal("discovery"))
		Expect(msg.Collective()).To(Equal("test_collective"))
		Expect(msg.Payload).To(Equal("hello world"))

	})

	It("Should handle publish failures gracefully", func() {
		dat := []byte("hello world")
		reg.Dat = &dat

		conn.SetNextError("simulated failure")
		server.pollAndPublish(&reg, &conn)
	})
})
