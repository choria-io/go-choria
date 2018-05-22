package registration

import (
	"os"
	"testing"

	framework "github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/choria/connectortest"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/server/data"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestRegistration(t *testing.T) {
	os.Setenv("MCOLLECTIVE_CERTNAME", "rip.mcollective")
	RegisterFailHandler(Fail)
	RunSpecs(t, "Server/Registration")
}

var _ = Describe("Server/Registration", func() {
	var _ = Describe("publish", func() {
		var (
			conn    *connectortest.PublishableConnector
			err     error
			choria  *framework.Framework
			cfg     *config.Config
			log     *logrus.Entry
			manager *Manager
		)

		BeforeSuite(func() {
			cfg, err = config.NewDefaultConfig()
			cfg.DisableTLS = true

			choria, err = framework.NewWithConfig(cfg)
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
			conn = &connectortest.PublishableConnector{}
			manager = New(choria, conn, log)
		})

		It("Should do nothing when the message is nil", func() {
			manager.publish(nil)
			Expect(conn.PublishedMsgs).To(BeEmpty())
		})

		It("Should do nothing when the  data is nil", func() {
			manager.publish(&data.RegistrationItem{})
			Expect(conn.PublishedMsgs).To(BeEmpty())
		})

		It("Should do nothing for empty data", func() {
			dat := []byte{}
			manager.publish(&data.RegistrationItem{Data: &dat})
			Expect(conn.PublishedMsgs).To(BeEmpty())
		})

		It("Should publish to registration agent when not set", func() {
			dat := []byte("hello world")
			manager.publish(&data.RegistrationItem{Data: &dat})

			published := conn.PublishedMsgs[0]
			Expect(published.Agent).To(Equal("registration"))
		})

		It("Should publish to the configured agent when set", func() {
			dat := []byte("hello world")
			manager.publish(&data.RegistrationItem{Data: &dat, TargetAgent: "ginkgo"})

			published := conn.PublishedMsgs[0]
			Expect(published.Agent).To(Equal("ginkgo"))
		})

		It("Should handle publish failures gracefully", func() {
			dat := []byte("hello world")
			conn.SetNextError("simulated failure")

			manager.publish(&data.RegistrationItem{Data: &dat, TargetAgent: "ginkgo"})
		})
	})
})
