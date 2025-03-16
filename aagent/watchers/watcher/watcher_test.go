// Copyright (c) 2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package watcher

import (
	"testing"

	"github.com/choria-io/go-choria/aagent/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AAgent/Watchers/Watcher")
}

var _ = Describe("Watcher", func() {
	var w *Watcher
	var mockctl *gomock.Controller
	var mockmachine *model.MockMachine

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())
		mockmachine = model.NewMockMachine(mockctl)

		w = &Watcher{
			machine: mockmachine,
		}
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	Describe("ShouldWatch", func() {
		It("Should handle no states or requirements", func() {
			Expect(w.ShouldWatch()).Should(BeTrue())
		})

		It("Should handle only states", func() {
			w.activeStates = []string{"ok"}
			mockmachine.EXPECT().State().Return("ok").Times(1)
			Expect(w.ShouldWatch()).Should(BeTrue())

			mockmachine.EXPECT().State().Return("notok").Times(1)
			Expect(w.ShouldWatch()).Should(BeFalse())

			w.activeStates = []string{"ok", "notok"}
			mockmachine.EXPECT().State().Return("notok").Times(1)
			Expect(w.ShouldWatch()).Should(BeTrue())
		})

		It("Should handle only requirements", func() {
			w.requiredStates = []model.ForeignMachineState{
				{MachineName: "other", MachineState: "ok"},
			}

			mockmachine.EXPECT().LookupExternalMachineState(gomock.Eq("other")).Return("ok", nil).Times(1)
			Expect(w.ShouldWatch()).Should(BeTrue())

			mockmachine.EXPECT().LookupExternalMachineState(gomock.Eq("other")).Return("notok", nil).Times(1)
			Expect(w.ShouldWatch()).Should(BeFalse())

			mockmachine.EXPECT().LookupExternalMachineState(gomock.Eq("other1")).Return("ok", nil).Times(1)
			mockmachine.EXPECT().LookupExternalMachineState(gomock.Eq("other2")).Return("ok", nil).Times(1)
			w.requiredStates = []model.ForeignMachineState{
				{MachineName: "other1", MachineState: "ok"},
				{MachineName: "other2", MachineState: "ok"},
			}
			Expect(w.ShouldWatch()).Should(BeTrue())

			mockmachine.EXPECT().LookupExternalMachineState(gomock.Eq("other1")).Return("ok", nil).Times(1)
			mockmachine.EXPECT().LookupExternalMachineState(gomock.Eq("other2")).Return("ok", nil).Times(1)
			w.requiredStates = []model.ForeignMachineState{
				{MachineName: "other1", MachineState: "ok"},
				{MachineName: "other2", MachineState: "notok"},
			}
			Expect(w.ShouldWatch()).Should(BeFalse())
		})

		It("Should handle states and requirements", func() {
			w.requiredStates = []model.ForeignMachineState{
				{MachineName: "other", MachineState: "ok"},
			}
			w.activeStates = []string{"ok", "notok"}

			mockmachine.EXPECT().State().Return("notok").Times(1)
			mockmachine.EXPECT().LookupExternalMachineState(gomock.Eq("other")).Return("ok", nil).Times(1)
			Expect(w.ShouldWatch()).Should(BeTrue())
		})
	})
})
