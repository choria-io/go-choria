package metricwatcher

import (
	"encoding/json"
	"testing"
	"time"

	gomock "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/choria-io/go-choria/aagent/watchers/watcher"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AAgent/Watchers/MetricWatcher")
}

var _ = Describe("MetricWatcher", func() {
	var (
		mockctl     *gomock.Controller
		mockMachine *watcher.MockMachine
		watch       *Watcher
		now         time.Time
	)

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())
		mockMachine = watcher.NewMockMachine(mockctl)

		now = time.Unix(1606924953, 0)
		mockMachine.EXPECT().Name().Return("metric").AnyTimes()
		mockMachine.EXPECT().Identity().Return("ginkgo").AnyTimes()
		mockMachine.EXPECT().InstanceID().Return("1234567890").AnyTimes()
		mockMachine.EXPECT().Version().Return("1.0.0").AnyTimes()
		mockMachine.EXPECT().TimeStampSeconds().Return(now.Unix()).AnyTimes()

		watch = &Watcher{previousRunTime: 500 * time.Millisecond, machine: mockMachine, name: "ginkgo"}
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	Describe("setProperties", func() {
		It("Should parse valid properties", func() {
			w := &Watcher{}

			err := w.setProperties(map[string]interface{}{
				"command":  "cmd",
				"interval": "1s",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(w.properties.Command).To(Equal("cmd"))
			Expect(w.properties.Interval).To(Equal(time.Second))
		})

		It("Should handle errors", func() {
			w := &Watcher{}
			err := w.setProperties(map[string]interface{}{
				"interval": "500ms",
			})
			Expect(err).To(MatchError("command is required"))
		})

		It("Should enforce 1 second intervals", func() {
			w := &Watcher{}
			err := w.setProperties(map[string]interface{}{
				"command":  "cmd",
				"interval": "500ms",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(w.properties.Command).To(Equal("cmd"))
			Expect(w.properties.Interval).To(Equal(time.Second))
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
				"time":        "2020-12-02T16:02:33Z",
				"type":        "io.choria.machine.watcher.metric.v1.state",
				"subject":     "ginkgo",
				"specversion": "1.0",
				"source":      "io.choria.machine",
				"data": map[string]interface{}{
					"id":        "1234567890",
					"identity":  "ginkgo",
					"machine":   "metric",
					"name":      "ginkgo",
					"protocol":  "io.choria.machine.watcher.metric.v1.state",
					"type":      "metric",
					"version":   "1.0.0",
					"timestamp": float64(now.Unix()),
					"metrics": map[string]interface{}{
						"labels": map[string]interface{}{},
						"metrics": map[string]interface{}{
							"choria_runtime_seconds": 0.5,
						},
					},
				},
			}))
		})
	})
})
