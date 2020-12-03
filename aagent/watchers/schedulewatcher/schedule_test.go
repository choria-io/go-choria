package schedulewatcher

import (
	"encoding/json"
	"testing"
	"time"

	gomock "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AAgent/Watchers/ScheduleWatcher")
}

var _ = Describe("ScheduleWatcher", func() {
	Describe("setProperties", func() {
		It("Should parse valid properties", func() {
			w := &Watcher{}

			err := w.setProperties(map[string]interface{}{
				"duration":  "1h",
				"schedules": []string{"* * * * *", "1 * * * *"},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(w.duration).To(Equal(time.Hour))
			Expect(w.items).To(HaveLen(2))
			Expect(w.schedules).To(HaveLen(2))
			Expect(w.items[0].spec).To(Equal("* * * * *"))
			Expect(w.items[1].spec).To(Equal("1 * * * *"))
		})

		It("Should handle errors", func() {
			w := &Watcher{}

			err := w.setProperties(map[string]interface{}{})
			Expect(err).To(MatchError("no schedules defined"))

			w = &Watcher{}
			err = w.setProperties(map[string]interface{}{
				"schedules": []string{"* * * * *", "1 * * * *"},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(w.duration).To(Equal(time.Minute))
			Expect(w.items).To(HaveLen(2))
			Expect(w.schedules).To(HaveLen(2))
		})
	})

	Describe("CurrentState", func() {
		var (
			mockctl     *gomock.Controller
			mockMachine *MockMachine
			watcher     *Watcher
			now         time.Time
		)

		BeforeEach(func() {
			mockctl = gomock.NewController(GinkgoT())
			mockMachine = NewMockMachine(mockctl)

			now = time.Unix(1606924953, 0)
			mockMachine.EXPECT().Name().Return("schedule").AnyTimes()
			mockMachine.EXPECT().Identity().Return("ginkgo").AnyTimes()
			mockMachine.EXPECT().InstanceID().Return("1234567890").AnyTimes()
			mockMachine.EXPECT().Version().Return("1.0.0").AnyTimes()
			mockMachine.EXPECT().TimeStampSeconds().Return(now.Unix()).AnyTimes()

			watcher = &Watcher{state: On, machine: mockMachine, name: "ginkgo"}
		})

		AfterEach(func() {
			mockctl.Finish()
		})

		It("Should be a valid state", func() {
			cs := watcher.CurrentState()
			csj, err := cs.(*StateNotification).JSON()
			Expect(err).ToNot(HaveOccurred())

			event := map[string]interface{}{}
			err = json.Unmarshal(csj, &event)
			Expect(err).ToNot(HaveOccurred())
			delete(event, "id")

			Expect(event).To(Equal(map[string]interface{}{
				"time":        "2020-12-02T16:02:33Z",
				"type":        "io.choria.machine.watcher.schedule.v1.state",
				"subject":     "ginkgo",
				"specversion": "1.0",
				"source":      "io.choria.machine",
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
