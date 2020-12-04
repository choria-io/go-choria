package execwatcher

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/choria-io/go-choria/aagent/watchers/watcher"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AAgent/Watchers/ExecWatcher")
}

var _ = Describe("ExecWatcher", func() {
	var (
		mockctl     *gomock.Controller
		mockMachine *watcher.MockMachine
		watch       *Watcher
		now         time.Time
	)

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())
		mockMachine = watcher.NewMockMachine(mockctl)

		mockMachine.EXPECT().Name().Return("exec").AnyTimes()
		mockMachine.EXPECT().Identity().Return("ginkgo").AnyTimes()
		mockMachine.EXPECT().InstanceID().Return("1234567890").AnyTimes()
		mockMachine.EXPECT().Version().Return("1.0.0").AnyTimes()
		mockMachine.EXPECT().TimeStampSeconds().Return(now.Unix()).AnyTimes()

		now = time.Unix(1606924953, 0)
		w, err := watcher.NewWatcher("exec", "exec", time.Second, []string{"always"}, mockMachine, "fail", "success")
		Expect(err).ToNot(HaveOccurred())

		watch = &Watcher{
			Watcher: w,
			machine: mockMachine,
			properties: &Properties{
				Environment: []string{},
			},
			previous:        Success,
			previousRunTime: time.Second,
			name:            "ginkgo",
		}
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	Describe("setProperties", func() {
		It("Should parse valid properties", func() {
			prop := map[string]interface{}{
				"command":                   "cmd",
				"timeout":                   "1.5s",
				"environment":               []string{"key1=val1", "key2=val2"},
				"suppress_success_announce": "true",
			}
			Expect(watch.setProperties(prop)).ToNot(HaveOccurred())
			Expect(watch.properties.Command).To(Equal("cmd"))
			Expect(watch.properties.Timeout).To(Equal(1500 * time.Millisecond))
			Expect(watch.properties.Environment).To(Equal([]string{"key1=val1", "key2=val2"}))
			Expect(watch.properties.SuppressSuccessAnnounce).To(BeTrue())
		})

		It("Should handle errors", func() {
			err := watch.setProperties(map[string]interface{}{})
			Expect(err).To(MatchError("command is required"))
		})

		It("Should enforce 1 second intervals", func() {
			err := watch.setProperties(map[string]interface{}{
				"command": "cmd",
				"timeout": "0",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(watch.properties.Command).To(Equal("cmd"))
			Expect(watch.properties.Timeout).To(Equal(time.Second))
		})
	})

	Describe("CurrentState", func() {
		It("Should be a valid state", func() {
			watch.properties.Command = "/bin/sh"

			cs := watch.CurrentState()
			csj, err := cs.(*StateNotification).JSON()
			Expect(err).ToNot(HaveOccurred())

			event := map[string]interface{}{}
			err = json.Unmarshal(csj, &event)
			Expect(err).ToNot(HaveOccurred())
			delete(event, "id")

			Expect(event).To(Equal(map[string]interface{}{
				"time":        "2020-12-02T16:02:33Z",
				"type":        "io.choria.machine.watcher.exec.v1.state",
				"subject":     "ginkgo",
				"specversion": "1.0",
				"source":      "io.choria.machine",
				"data": map[string]interface{}{
					"command":           "/bin/sh",
					"previous_outcome":  "success",
					"previous_run_time": float64(time.Second),
					"id":                "1234567890",
					"identity":          "ginkgo",
					"machine":           "exec",
					"name":              "ginkgo",
					"protocol":          "io.choria.machine.watcher.exec.v1.state",
					"type":              "exec",
					"version":           "1.0.0",
					"timestamp":         float64(now.Unix()),
				},
			}))
		})
	})
})
