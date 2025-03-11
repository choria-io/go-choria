// Copyright (c) 2020-2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package timerwatcher

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/choria-io/go-choria/aagent/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AAgent/Watchers/TimerWatcher")
}

var _ = Describe("TimerWatcher", func() {
	var (
		mockctl     *gomock.Controller
		mockMachine *model.MockMachine
		watch       *Watcher
		now         time.Time
	)

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())
		mockMachine = model.NewMockMachine(mockctl)

		now = time.Unix(1606924953, 0)
		mockMachine.EXPECT().Name().Return("timer").AnyTimes()
		mockMachine.EXPECT().Identity().Return("ginkgo").AnyTimes()
		mockMachine.EXPECT().InstanceID().Return("1234567890").AnyTimes()
		mockMachine.EXPECT().Version().Return("1.0.0").AnyTimes()
		mockMachine.EXPECT().TimeStampSeconds().Return(now.Unix()).AnyTimes()
		mockMachine.EXPECT().Infof(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

		wi, err := New(mockMachine, "ginkgo", []string{"always"}, nil, "fail", "", "2m", time.Second, map[string]any{})
		Expect(err).ToNot(HaveOccurred())
		watch = wi.(*Watcher)
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	Describe("setProperties", func() {
		It("Should parse valid properties", func() {
			err := watch.setProperties(map[string]any{
				"timer": "1h",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(watch.properties.Timer).To(Equal(time.Hour))
		})

		It("Should handle errors", func() {
			err := watch.setProperties(map[string]any{})
			Expect(err).ToNot(HaveOccurred())
			Expect(watch.properties.Timer).To(Equal(time.Second))
		})

		It("Should handle splay", func() {
			err := watch.setProperties(map[string]any{
				"timer": "1h",
				"splay": true,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(watch.properties.Timer).To(And(
				BeNumerically("<", time.Hour),
				BeNumerically(">", 0),
			))
		})
	})

	Describe("CurrentState", func() {
		It("Should be a valid state", func() {
			cs := watch.CurrentState()
			csj, err := cs.(*StateNotification).JSON()
			Expect(err).ToNot(HaveOccurred())

			event := map[string]any{}
			err = json.Unmarshal(csj, &event)
			Expect(err).ToNot(HaveOccurred())
			delete(event, "id")

			Expect(event).To(Equal(map[string]any{
				"time":            "2020-12-02T16:02:33Z",
				"type":            "io.choria.machine.watcher.timer.v1.state",
				"subject":         "ginkgo",
				"specversion":     "1.0",
				"source":          "io.choria.machine",
				"datacontenttype": "application/json",
				"data": map[string]any{
					"id":        "1234567890",
					"identity":  "ginkgo",
					"machine":   "timer",
					"name":      "ginkgo",
					"protocol":  "io.choria.machine.watcher.timer.v1.state",
					"state":     "stopped",
					"timer":     float64(time.Second),
					"type":      "timer",
					"version":   "1.0.0",
					"timestamp": float64(now.Unix()),
				},
			}))
		})
	})
})
