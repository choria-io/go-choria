package nagioswatcher

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/choria-io/go-choria/aagent/model"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/choria-io/go-choria/aagent/util"
	"github.com/choria-io/go-choria/statistics"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AAgent/Watchers/NagiosWatcher")
}

var _ = Describe("NagiosWatcher", func() {
	var (
		mockctl     *gomock.Controller
		mockMachine *model.MockMachine
		watch       *Watcher
		now         time.Time
		err         error
		td          string
	)

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())
		mockMachine = model.NewMockMachine(mockctl)

		td, err = ioutil.TempDir("", "")
		Expect(err).ToNot(HaveOccurred())

		now = time.Unix(1606924953, 0)
		mockMachine.EXPECT().Name().Return("nagios").AnyTimes()
		mockMachine.EXPECT().Identity().Return("ginkgo").AnyTimes()
		mockMachine.EXPECT().InstanceID().Return("1234567890").AnyTimes()
		mockMachine.EXPECT().Version().Return("1.0.0").AnyTimes()
		mockMachine.EXPECT().TimeStampSeconds().Return(now.Unix()).AnyTimes()
		mockMachine.EXPECT().TextFileDirectory().Return(td).AnyTimes()

		wi, err := New(mockMachine, "ginkgo", []string{"always"}, "fail", "success", "1s", time.Second, map[string]interface{}{
			"plugin": "/bin/sh",
		})
		Expect(err).ToNot(HaveOccurred())

		watch = wi.(*Watcher)
		watch.previousCheck = now
		watch.previousOutput = "OK: ginkgo"
		watch.previousPerfData = []util.PerfData{}
		watch.previousRunTime = 500 * time.Millisecond
		watch.previous = OK
		watch.previousPlugin = "/bin/sh"
	})

	AfterEach(func() {
		mockctl.Finish()
		os.RemoveAll(td)
	})

	Describe("setProperties", func() {
		It("Should parse valid properties", func() {
			watch.properties = nil
			err = watch.setProperties(map[string]interface{}{
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
			watch.properties = nil
			err = watch.setProperties(map[string]interface{}{})
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

			watch.properties = nil
			err = watch.setProperties(map[string]interface{}{
				"builtin": "choria_status",
			})
			Expect(err).To(MatchError("last_message property is required for the choria_status builtin check"))
		})

		It("Should handle valid goss setups", func() {
			watch.properties = nil
			err = watch.setProperties(map[string]interface{}{
				"builtin":  "goss",
				"gossFile": "/x",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(watch.properties.Builtin).To(Equal("goss"))
			Expect(watch.properties.Gossfile).To(Equal("/x"))
		})

		It("Should handle valid choria_status setups", func() {
			watch.properties = nil
			err = watch.setProperties(map[string]interface{}{
				"builtin":      "choria_status",
				"last_message": "1h",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(watch.properties.Builtin).To(Equal("choria_status"))
			Expect(watch.properties.LastMessage).To(Equal(time.Hour))

			sf := filepath.Join(td, "status.json")
			mockMachine.EXPECT().ChoriaStatusFile().Return(sf, 60*60).AnyTimes()
			state, _, err := watch.watchUsingChoria()
			Expect(state).To(Equal(CRITICAL))
			Expect(err).ToNot(HaveOccurred())

			now := time.Now().UTC()
			status := statistics.InstanceStatus{
				Identity:        "ginkgo.example.net",
				Uptime:          1000,
				ConnectedServer: "broker.example.net",
				LastMessage:     now.Unix(),
				Provisioning:    false,
				Stats: &statistics.ServerStats{
					Total:      4,
					Valid:      1,
					Invalid:    1,
					Passed:     1,
					Filtered:   1,
					Replies:    2,
					TTLExpired: 1,
				},
				FileName: sf,
				ModTime:  now,
			}
			sj, _ := json.Marshal(status)
			ioutil.WriteFile(sf, sj, 0644)

			state, output, err := watch.watchUsingChoria()
			Expect(output).To(Equal(fmt.Sprintf("OK: %s|uptime=1000;; filtered_msgs=1;; invalid_msgs=1;; passed_msgs=1;; replies_msgs=2;; total_msgs=4;; ttlexpired_msgs=1;; last_msg=%d;;", sf, now.Unix())))
			Expect(state).To(Equal(OK))
			Expect(err).ToNot(HaveOccurred())

			lm := now.Add(-1 * 70 * 70 * time.Second)
			status.LastMessage = lm.Unix()
			sj, _ = json.Marshal(status)
			ioutil.WriteFile(sf, sj, 0644)
			state, output, err = watch.watchUsingChoria()
			Expect(state).To(Equal(CRITICAL))
			Expect(output).To(Equal(fmt.Sprintf("CRITICAL: last message at %s|uptime=1000;; filtered_msgs=1;; invalid_msgs=1;; passed_msgs=1;; replies_msgs=2;; total_msgs=4;; ttlexpired_msgs=1;; last_msg=%d;;", time.Unix(status.LastMessage, 0).UTC(), status.LastMessage)))
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
				"type":            "io.choria.machine.watcher.nagios.v1.state",
				"subject":         "ginkgo",
				"specversion":     "1.0",
				"source":          "io.choria.machine",
				"datacontenttype": "application/json",
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
