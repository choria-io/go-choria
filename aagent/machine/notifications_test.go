// Copyright (c) 2019-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package machine

import (
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Aagent/Machine/Notifications", func() {
	var (
		mockctl  *gomock.Controller
		service1 *MockNotificationService
		service2 *MockNotificationService
		manager  *MockWatcherManager
		event    *MockWatcherStateNotification
		machine  *Machine
	)

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())
		service1 = NewMockNotificationService(mockctl)
		service2 = NewMockNotificationService(mockctl)
		manager = NewMockWatcherManager(mockctl)
		event = NewMockWatcherStateNotification(mockctl)
		machine = &Machine{
			notifiers:   []NotificationService{},
			manager:     manager,
			MachineName: "ginkgo",
		}
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	Describe("RegisterNotifier", func() {
		It("Should add the notifier to the list", func() {
			Expect(machine.notifiers).To(BeEmpty())
			machine.RegisterNotifier(service1)
			Expect(machine.notifiers[0]).To(Equal(service1))
			Expect(machine.notifiers).To(HaveLen(1))
		})
	})

	Describe("Notifications", func() {
		BeforeEach(func() {
			machine.RegisterNotifier(service1, service2)
		})

		It("Should support notifying state", func() {
			service1.EXPECT().NotifyWatcherState("w1", event)
			service2.EXPECT().NotifyWatcherState("w1", event)

			machine.NotifyWatcherState("w1", event)
		})

		It("Should support common loggers", func() {
			service1.EXPECT().Debugf(machine, "w1", "format", "debugarg")
			service2.EXPECT().Debugf(machine, "w1", "format", "debugarg")
			service1.EXPECT().Infof(machine, "w1", "format", "infoarg")
			service2.EXPECT().Infof(machine, "w1", "format", "infoarg")
			service1.EXPECT().Warnf(machine, "w1", "format", "warnarg")
			service2.EXPECT().Warnf(machine, "w1", "format", "warnarg")
			service1.EXPECT().Errorf(machine, "w1", "format", "errorarg")
			service2.EXPECT().Errorf(machine, "w1", "format", "errorarg")

			machine.Debugf("w1", "format", "debugarg")
			machine.Infof("w1", "format", "infoarg")
			machine.Warnf("w1", "format", "warnarg")
			machine.Errorf("w1", "format", "errorarg")
		})
	})
})
