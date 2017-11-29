package registration

import (
	"errors"
	"testing"

	framework "github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/choria/connectortest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestFileContent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Server/Registration")
}

type StubRegistrator struct {
	Err error
	Dat *[]byte
}

func (sr *StubRegistrator) RegistrationData() (*[]byte, error) {
	return sr.Dat, sr.Err
}

var _ = Describe("pollAndPublish", func() {
	var (
		conn    *connectortest.StubPublishingConnector
		reg     *StubRegistrator
		err     error
		choria  *framework.Framework
		cfg     *framework.Config
		log     *logrus.Entry
		manager *Manager
	)

	BeforeSuite(func() {
		choria, err = framework.New("/dev/null")
		Expect(err).ToNot(HaveOccurred())

		cfg = choria.Config
		cfg.DisableTLS = true
		cfg.OverrideCertname = "test.example.net"
		cfg.Collectives = []string{"test_collective"}
		cfg.MainCollective = "test_collective"
		cfg.RegistrationCollective = "test_collective"

		log = logrus.WithFields(logrus.Fields{"test": true})
		logrus.SetLevel(logrus.FatalLevel)
	})

	BeforeEach(func() {
		conn = &connectortest.StubPublishingConnector{}
		reg = &StubRegistrator{}
		manager = New(choria, conn, log)
	})

	It("Should do nothing when the RegistrationData poll failed", func() {
		reg.Err = errors.New("Simulated error")
		manager.pollAndPublish(reg)
		Expect(conn.PublishedMsgs).To(BeEmpty())
	})

	It("Should do nothing for nil data", func() {
		reg.Dat = nil
		manager.pollAndPublish(reg)
		Expect(conn.PublishedMsgs).To(BeEmpty())
	})

	It("Should do nothing for empty data", func() {
		reg.Dat = &[]byte{}
		manager.pollAndPublish(reg)
		Expect(conn.PublishedMsgs).To(BeEmpty())
	})

	It("Should publish a message for the discovery agent when it finds data", func() {
		dat := []byte("hello world")
		reg.Dat = &dat

		manager.pollAndPublish(reg)
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
		manager.pollAndPublish(reg)
	})
})
