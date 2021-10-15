// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package schedulewatcher

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/choria-io/go-choria/aagent/model"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AAgent/Watchers/ScheduleWatcher")
}

var _ = Describe("ScheduleWatcher", func() {
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
		mockMachine.EXPECT().Name().Return("schedule").AnyTimes()
		mockMachine.EXPECT().Identity().Return("ginkgo").AnyTimes()
		mockMachine.EXPECT().InstanceID().Return("1234567890").AnyTimes()
		mockMachine.EXPECT().Version().Return("1.0.0").AnyTimes()
		mockMachine.EXPECT().TimeStampSeconds().Return(now.Unix()).AnyTimes()

		wi, err := New(mockMachine, "ginkgo", []string{"always"}, "fail", "success", "2m", time.Second, map[string]interface{}{
			"schedules": []string{"1 * * * *"},
		})
		Expect(err).ToNot(HaveOccurred())
		watch = wi.(*Watcher)
		watch.properties = nil
		watch.items = []*scheduleItem{}
		watch.state = On
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	Describe("setProperties", func() {
		It("Should parse valid properties", func() {
			err := watch.setProperties(map[string]interface{}{
				"duration":  "1h",
				"schedules": []string{"* * * * *", "1 * * * *"},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(watch.properties.Duration).To(Equal(time.Hour))
			Expect(watch.properties.Schedules).To(HaveLen(2))
			Expect(watch.items).To(HaveLen(2))
			Expect(watch.items[0].spec).To(Equal("* * * * *"))
			Expect(watch.items[1].spec).To(Equal("1 * * * *"))
		})

		It("Should handle errors", func() {
			err := watch.setProperties(map[string]interface{}{})
			Expect(err).To(MatchError("no schedules defined"))

			watch.properties = nil
			err = watch.setProperties(map[string]interface{}{
				"schedules": []string{"* * * * *", "1 * * * *"},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(watch.properties.Duration).To(Equal(time.Minute))
			Expect(watch.items).To(HaveLen(2))
			Expect(watch.properties.Schedules).To(HaveLen(2))
		})

		It("Should handle startup splays", func() {
			err := watch.setProperties(map[string]interface{}{
				"start_splay": "1m",
				"duration":    "1m",
				"schedules":   []string{"* * * * *", "1 * * * *"},
			})
			Expect(err).To(MatchError("start splay 1m0s is bigger than half the duration 1m0s"))

			err = watch.setProperties(map[string]interface{}{
				"start_splay": "10s",
				"duration":    "1m",
				"schedules":   []string{"* * * * *", "1 * * * *"},
			})
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("CurrentState", func() {
		It("Should be a valid state", func() {
			cs := watch.CurrentState()
			csj, err := cs.(*StateNotification).JSON()
			Expect(err).ToNot(HaveOccurred())

			event := map[string]interface{}{}
			err = json.Unmarshal(csj, &event)
			Expect(err).ToNot(HaveOccurred())
			delete(event, "id")

			Expect(event).To(Equal(map[string]interface{}{
				"time":            "2020-12-02T16:02:33Z",
				"type":            "io.choria.machine.watcher.schedule.v1.state",
				"subject":         "ginkgo",
				"specversion":     "1.0",
				"source":          "io.choria.machine",
				"datacontenttype": "application/json",
				"data": map[string]interface{}{
					"id":        "1234567890",
					"identity":  "ginkgo",
					"machine":   "schedule",
					"name":      "ginkgo",
					"protocol":  "io.choria.machine.watcher.schedule.v1.state",
					"type":      "schedule",
					"version":   "1.0.0",
					"timestamp": float64(now.Unix()),
					"state":     "on",
				},
			}))
		})
	})
})
