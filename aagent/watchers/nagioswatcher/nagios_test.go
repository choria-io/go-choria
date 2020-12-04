package nagioswatcher

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
	RunSpecs(t, "AAgent/Watchers/NagiosWatcher")
}

var _ = Describe("NagiosWatcher", func() {
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
		mockMachine.EXPECT().Name().Return("nagios").AnyTimes()
		mockMachine.EXPECT().Identity().Return("ginkgo").AnyTimes()
		mockMachine.EXPECT().InstanceID().Return("1234567890").AnyTimes()
		mockMachine.EXPECT().Version().Return("1.0.0").AnyTimes()
		mockMachine.EXPECT().TimeStampSeconds().Return(now.Unix()).AnyTimes()

		watch = &Watcher{
			previousPlugin:   "/bin/sh",
			previous:         OK,
			previousOutput:   "OK: ginkgo",
			machine:          mockMachine,
			previousPerfData: []PerfData{},
			previousRunTime:  500 * time.Millisecond,
			history:          []*Execution{},
			name:             "ginkgo",
			previousCheck:    now,
		}

		watch.setProperties(map[string]interface{}{})
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	Describe("setProperties", func() {
		It("Should parse valid properties", func() {
			err := watch.setProperties(map[string]interface{}{
				"annotations": map[string]string{
					"a1": "v1",
					"a2": "v2",
				},
				"plugin":  "cmd",
				"timeout": "5s",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(watch.properties.Annotations).To(Equal(map[string]string{
				"a1": "v1",
				"a2": "v2",
			}))
			Expect(watch.properties.Plugin).To(Equal("cmd"))
			Expect(watch.properties.Timeout).To(Equal(5 * time.Second))
			Expect(watch.properties.Builtin).To(BeEmpty())
			Expect(watch.properties.Gossfile).To(BeEmpty())
		})

		It("Should handle errors", func() {
			err := watch.setProperties(map[string]interface{}{})
			Expect(err).To(MatchError("plugin or builtin is required"))

			watch.properties = nil
			err = watch.setProperties(map[string]interface{}{
				"plugin":  "cmd",
				"builtin": "goss",
			})
			Expect(err).To(MatchError("cannot set plugin and builtin"))

			watch.properties = nil
			err = watch.setProperties(map[string]interface{}{
				"builtin": "goss",
			})
			Expect(err).To(MatchError("gossfile property is required for the goss builtin check"))
		})

		It("Should handle valid goss setups", func() {
			err := watch.setProperties(map[string]interface{}{
				"builtin":  "goss",
				"gossFile": "/x",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(watch.properties.Builtin).To(Equal("goss"))
			Expect(watch.properties.Gossfile).To(Equal("/x"))
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
				"type":        "io.choria.machine.watcher.nagios.v1.state",
				"subject":     "ginkgo",
				"specversion": "1.0",
				"source":      "io.choria.machine",
				"data": map[string]interface{}{
					"id":          "1234567890",
					"identity":    "ginkgo",
					"machine":     "nagios",
					"name":        "ginkgo",
					"protocol":    "io.choria.machine.watcher.nagios.v1.state",
					"type":        "nagios",
					"version":     "1.0.0",
					"timestamp":   float64(now.Unix()),
					"status_code": float64(0),
					"runtime":     0.5,
					"check_time":  float64(now.Unix()),
					"annotations": map[string]interface{}{},
					"perfdata":    []interface{}{},
					"history":     []interface{}{},
					"status":      "OK",
					"output":      "OK: ginkgo",
					"plugin":      "/bin/sh",
				},
			}))
		})
	})
})
