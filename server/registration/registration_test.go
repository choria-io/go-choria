// Copyright (c) 2017-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package registration

import (
	"errors"
	"os"
	"testing"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/inter"
	imock "github.com/choria-io/go-choria/inter/imocks"
	"github.com/choria-io/go-choria/message"
	"github.com/choria-io/go-choria/server/data"
	"github.com/golang/mock/gomock"
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
			conn    *imock.MockConnector
			si      *MockServerInfoSource
			fw      *imock.MockFramework
			cfg     *config.Config
			log     *logrus.Entry
			manager *Manager
			mockctl *gomock.Controller
		)

		BeforeEach(func() {
			mockctl = gomock.NewController(GinkgoT())
			fw, cfg = imock.NewFrameworkForTests(mockctl, GinkgoWriter, imock.WithCallerID())
			cfg.DisableTLS = true
			cfg.OverrideCertname = "test.example.net"
			cfg.Collectives = []string{"test_collective"}
			cfg.MainCollective = "test_collective"
			cfg.RegistrationCollective = "test_collective"

			fw.EXPECT().NewMessage(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(payload string, agent string, collective string, msgType string, request inter.Message) (msg inter.Message, err error) {
				return message.NewMessage(payload, agent, collective, msgType, request, fw)
			}).AnyTimes()

			log = logrus.WithFields(logrus.Fields{"test": true})
			logrus.SetLevel(logrus.FatalLevel)

			conn = imock.NewMockConnector(mockctl)
			si = NewMockServerInfoSource(mockctl)
			manager = New(fw, si, conn, log)
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
			manager.publish(&data.RegistrationItem{Data: dat})
		})

		It("Should publish to registration agent when not set", func() {
			dat := []byte("hello world")

			msg := &message.Message{}
			conn.EXPECT().IsConnected().Return(true)
			conn.EXPECT().Publish(gomock.AssignableToTypeOf(msg)).DoAndReturn(func(m *message.Message) {
				Expect(m.Agent()).To(Equal("registration"))
			}).Return(nil).AnyTimes()

			manager.publish(&data.RegistrationItem{Data: dat})
		})

		It("Should publish to the configured agent when set", func() {
			dat := []byte("hello world")
			msg := &message.Message{}
			conn.EXPECT().IsConnected().Return(true)
			conn.EXPECT().Publish(gomock.AssignableToTypeOf(msg)).DoAndReturn(func(m *message.Message) {
				Expect(m.Agent()).To(Equal("ginkgo"))
			}).Return(nil).AnyTimes()

			manager.publish(&data.RegistrationItem{Data: dat, TargetAgent: "ginkgo"})
		})

		It("Should handle publish failures gracefully", func() {
			dat := []byte("hello world")
			msg := &message.Message{}
			conn.EXPECT().IsConnected().Return(true)
			conn.EXPECT().Publish(gomock.AssignableToTypeOf(msg)).Return(errors.New("simulated failure")).AnyTimes()
			manager.publish(&data.RegistrationItem{Data: dat, TargetAgent: "ginkgo"})
		})

		It("Should not publish when not connected", func() {
			dat := []byte("hello world")
			conn.EXPECT().IsConnected().Return(false)
			manager.publish(&data.RegistrationItem{Data: dat, TargetAgent: "ginkgo"})
		})
	})
})
