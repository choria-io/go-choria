package timerwatcher

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

		wi, err := New(mockMachine, "ginkgo", []string{"always"}, "fail", "", "2m", time.Second, map[string]interface{}{})
		Expect(err).ToNot(HaveOccurred())
		watch = wi.(*Watcher)
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	Describe("setProperties", func() {
		It("Should parse valid properties", func() {
			err := watch.setProperties(map[string]interface{}{
				"timer": "1h",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(watch.properties.Timer).To(Equal(time.Hour))
		})

		It("Should handle errors", func() {
			err := watch.setProperties(map[string]interface{}{})
			Expect(err).ToNot(HaveOccurred())
			Expect(watch.properties.Timer).To(Equal(time.Second))
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
				"type":            "io.choria.machine.watcher.timer.v1.state",
				"subject":         "ginkgo",
				"specversion":     "1.0",
				"source":          "io.choria.machine",
				"datacontenttype": "application/json",
				"data": map[string]interface{}{
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
