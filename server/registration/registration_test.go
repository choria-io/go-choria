package registration

import (
	"errors"
	"os"
	"testing"

	framework "github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/server/data"
	gomock "github.com/golang/mock/gomock"
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
			conn    *MockConnection
			err     error
			choria  *framework.Framework
			cfg     *config.Config
			log     *logrus.Entry
			manager *Manager
			mockctl *gomock.Controller
		)

		BeforeSuite(func() {
			cfg = config.NewConfigForTests()
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
			mockctl = gomock.NewController(GinkgoT())
			conn = NewMockConnection(mockctl)
			manager = New(choria, conn, log)
		})

		AfterEach(func() {
			mockctl.Finish()
		})

		It("Should do nothing when the message is nil", func() {
			manager.publish(nil)
		})

		It("Should do nothing when the  data is nil", func() {
			manager.publish(&data.RegistrationItem{})
		})

		It("Should do nothing for empty data", func() {
			dat := []byte{}
			manager.publish(&data.RegistrationItem{Data: &dat})
		})

		It("Should publish to registration agent when not set", func() {
			dat := []byte("hello world")

			msg := &framework.Message{}
			conn.EXPECT().IsConnected().Return(true)
			conn.EXPECT().Publish(gomock.AssignableToTypeOf(msg)).DoAndReturn(func(m *framework.Message) {
				Expect(m.Agent).To(Equal("registration"))
			}).Return(nil).AnyTimes()

			manager.publish(&data.RegistrationItem{Data: &dat})
		})

		It("Should publish to the configured agent when set", func() {
			dat := []byte("hello world")
			msg := &framework.Message{}
			conn.EXPECT().IsConnected().Return(true)
			conn.EXPECT().Publish(gomock.AssignableToTypeOf(msg)).DoAndReturn(func(m *framework.Message) {
				Expect(m.Agent).To(Equal("ginkgo"))
			}).Return(nil).AnyTimes()

			manager.publish(&data.RegistrationItem{Data: &dat, TargetAgent: "ginkgo"})
		})

		It("Should handle publish failures gracefully", func() {
			dat := []byte("hello world")
			msg := &framework.Message{}
			conn.EXPECT().IsConnected().Return(true)
			conn.EXPECT().Publish(gomock.AssignableToTypeOf(msg)).Return(errors.New("simulated failure")).AnyTimes()
			manager.publish(&data.RegistrationItem{Data: &dat, TargetAgent: "ginkgo"})
		})

		It("Should not publish when not connected", func() {
			dat := []byte("hello world")
			conn.EXPECT().IsConnected().Return(false)
			manager.publish(&data.RegistrationItem{Data: &dat, TargetAgent: "ginkgo"})
		})
	})
})
