// Copyright (c) 2024-2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package expressionwatcher

import (
	"fmt"
	"testing"
	"time"

	"github.com/choria-io/go-choria/aagent/model"
	"github.com/choria-io/go-choria/aagent/watchers/event"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

func TestMachine(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AAgent/Watchers/ExpressionsWatcher")
}

var _ = Describe("AAgent/Watchers/ExpressionsWatcher", func() {
	var (
		w       *Watcher
		machine *model.MockMachine
		mockctl *gomock.Controller
		td      string
		err     error
	)

	BeforeEach(func() {
		td = GinkgoT().TempDir()

		mockctl = gomock.NewController(GinkgoT())

		machine = model.NewMockMachine(mockctl)
		machine.EXPECT().Directory().Return(td).AnyTimes()
		machine.EXPECT().SignerKey().Return("").AnyTimes()

		var wi any
		wi, err = New(machine, "ginkgo_machine", nil, nil, "FAIL_EVENT", "SUCCESS_EVENT", "1m", time.Hour, map[string]any{
			"success_when": "true",
		})
		Expect(err).ToNot(HaveOccurred())
		w = wi.(*Watcher)
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	Describe("handleCheck", func() {
		var now time.Time

		BeforeEach(func() {
			now = time.Now()
			machine.EXPECT().Identity().Return("ginkgo.example.net").AnyTimes()
			machine.EXPECT().InstanceID().Return("123").AnyTimes()
			machine.EXPECT().Version().Return("1.0.0").AnyTimes()
			machine.EXPECT().TimeStampSeconds().Return(now.Unix()).AnyTimes()
			machine.EXPECT().Name().Return("ginkgo").AnyTimes()
		})

		It("Should handle SuccessWhen", func() {
			w.previous = Skipped

			machine.EXPECT().Infof(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
			machine.EXPECT().Infof(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
			machine.EXPECT().NotifyWatcherState("ginkgo_machine", gomock.Eq(&StateNotification{
				Event:           event.New(w.name, wtype, version, w.machine),
				PreviousOutcome: stateNames[SuccessWhen],
			})).Times(2)

			// noce only since second time would be a flip-flop
			machine.EXPECT().Transition("SUCCESS_EVENT").Times(1)

			err := w.handleCheck(SuccessWhen, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(w.previous).To(Equal(SuccessWhen))

			err = w.handleCheck(SuccessWhen, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(w.previous).To(Equal(SuccessWhen))
		})

		It("Should handle FailWhen", func() {
			w.previous = Skipped

			machine.EXPECT().Infof(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
			machine.EXPECT().Infof(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
			machine.EXPECT().NotifyWatcherState("ginkgo_machine", gomock.Eq(&StateNotification{
				Event:           event.New(w.name, wtype, version, w.machine),
				PreviousOutcome: stateNames[FailWhen],
			})).Times(2)

			// noce only since second time would be a flip-flop
			machine.EXPECT().Transition("FAIL_EVENT").Times(1)

			err := w.handleCheck(FailWhen, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(w.previous).To(Equal(FailWhen))

			err = w.handleCheck(FailWhen, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(w.previous).To(Equal(FailWhen))
		})

		It("Should handle Error", func() {
			machine.EXPECT().Errorf("ginkgo_machine", gomock.Any(), gomock.Any()).Times(1)
			machine.EXPECT().NotifyWatcherState("ginkgo_machine", gomock.Eq(&StateNotification{
				Event:           event.New(w.name, wtype, version, w.machine),
				PreviousOutcome: stateNames[Error],
			})).Times(1)

			err := w.handleCheck(Error, fmt.Errorf("simulated"))
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("watch", func() {
		BeforeEach(func() {
			machine.EXPECT().Data().Return(map[string]any{"test": 1}).AnyTimes()
			machine.EXPECT().Facts().Return([]byte(`{"fqdn":"ginkgo.example.net"}`)).AnyTimes()
			machine.EXPECT().Identity().Return("ginkgo.example.net").AnyTimes()

			w.properties.FailWhen = ""
			w.properties.SuccessWhen = ""
		})

		It("Should handle success_when expressions", func() {
			machine.EXPECT().Debugf(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
			w.properties.SuccessWhen = "data.test == 1"
			state, err := w.watch()
			Expect(err).ToNot(HaveOccurred())
			Expect(state).To(Equal(SuccessWhen))

			w.properties.SuccessWhen = "get_fact('fqdn') == 'ginkgo.example.net'"
			state, err = w.watch()
			Expect(err).ToNot(HaveOccurred())
			Expect(state).To(Equal(SuccessWhen))

			w.properties.SuccessWhen = "data.test == 2"
			state, err = w.watch()
			Expect(err).ToNot(HaveOccurred())
			Expect(state).To(Equal(NoMatch))

			w.properties.SuccessWhen = "1"
			state, err = w.watch()
			Expect(err).To(MatchError("expected bool, but got int"))
			Expect(state).To(Equal(Error))
		})

		It("Should handle fail_when expressions", func() {
			machine.EXPECT().Debugf(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
			w.properties.FailWhen = "data.test == 1"
			state, err := w.watch()
			Expect(err).ToNot(HaveOccurred())
			Expect(state).To(Equal(FailWhen))

			w.properties.FailWhen = "data.test == 2"
			state, err = w.watch()
			Expect(err).ToNot(HaveOccurred())
			Expect(state).To(Equal(NoMatch))

			w.properties.FailWhen = "1"
			state, err = w.watch()
			Expect(err).To(MatchError("expected bool, but got int"))
			Expect(state).To(Equal(Error))
		})
	})

	Describe("setProperties", func() {
		It("Should validate the interval", func() {
			w.interval = time.Millisecond
			Expect(w.setProperties(nil)).To(MatchError("interval should be more than 1 second: 1ms"))
		})

		It("Should require one expressions", func() {
			w.properties.FailWhen = ""
			w.properties.SuccessWhen = ""

			Expect(w.setProperties(nil)).To(MatchError("success_when or fail_when is required"))

			w.properties.FailWhen = "true"
			Expect(w.setProperties(nil)).ToNot(HaveOccurred())

			w.properties.FailWhen = ""
			w.properties.SuccessWhen = "true"
			Expect(w.setProperties(nil)).ToNot(HaveOccurred())
		})
	})
})
