package metricwatcher

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
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
		td          string
		err         error
	)

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())
		mockMachine = watcher.NewMockMachine(mockctl)

		td, err = ioutil.TempDir("", "")
		Expect(err).ToNot(HaveOccurred())

		now = time.Unix(1606924953, 0)
		mockMachine.EXPECT().Name().Return("metric").AnyTimes()
		mockMachine.EXPECT().Identity().Return("ginkgo").AnyTimes()
		mockMachine.EXPECT().InstanceID().Return("1234567890").AnyTimes()
		mockMachine.EXPECT().Version().Return("1.0.0").AnyTimes()
		mockMachine.EXPECT().TimeStampSeconds().Return(now.Unix()).AnyTimes()
		mockMachine.EXPECT().Infof("ginkgo", gomock.Any(), gomock.Any()).AnyTimes()
		mockMachine.EXPECT().Debugf("ginkgo", gomock.Any(), gomock.Any()).AnyTimes()
		mockMachine.EXPECT().Directory().Return(".").AnyTimes()
		mockMachine.EXPECT().TextFileDirectory().Return(td).AnyTimes()
		mockMachine.EXPECT().State().Return("run").AnyTimes()

		watch, err = New(mockMachine, "ginkgo", []string{"run"}, "fail", "success", time.Second, map[string]interface{}{
			"command": "metric.sh",
		})
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		mockctl.Finish()
		os.Remove(td)
	})

	Describe("performWatch", func() {
		It("Should run the script and correctly parse the metrics", func() {
			if runtime.GOOS == "windows" {
				Skip("not tested on windows yet")
			}

			handled := false
			mockMachine.EXPECT().NotifyWatcherState("ginkgo", gomock.Any()).Do(func(_ string, m *StateNotification) {
				Expect(m.Metrics.Labels).To(Equal(map[string]string{"dupe": "w", "unique": "u"}))
				Expect(m.Metrics.Metrics["v1"]).To(Equal(float64(1)))
				Expect(m.Metrics.Metrics["v2"]).To(Equal(1.1))
				Expect(m.Metrics.Metrics["choria_runtime_seconds"]).To(BeNumerically(">", 0))
				handled = true
			})

			watch, err = New(mockMachine, "ginkgo", []string{"run"}, "fail", "success", time.Second, map[string]interface{}{
				"command": filepath.Join("testdata", "metric.sh"),
				"labels":  map[string]string{"dupe": "w"},
			})
			Expect(err).ToNot(HaveOccurred())

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			watch.performWatch(ctx)
			Expect(handled).To(BeTrue())
		})
	})

	Describe("setProperties", func() {
		It("Should parse valid properties", func() {
			err := watch.setProperties(map[string]interface{}{
				"command":  "cmd",
				"interval": "1s",
				"labels": map[string]string{
					"test": "label",
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(watch.properties.Command).To(Equal("cmd"))
			Expect(watch.properties.Interval).To(Equal(time.Second))
			Expect(watch.properties.Labels).To(Equal(map[string]string{"test": "label"}))
		})

		It("Should handle errors", func() {
			watch.properties = nil
			err = watch.setProperties(map[string]interface{}{
				"interval": "500ms",
			})
			Expect(err).To(MatchError("command is required"))
		})

		It("Should enforce 1 second intervals", func() {
			err := watch.setProperties(map[string]interface{}{
				"command":  "cmd",
				"interval": "500ms",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(watch.properties.Command).To(Equal("cmd"))
			Expect(watch.properties.Interval).To(Equal(time.Second))
		})
	})

	Describe("CurrentState", func() {
		It("Should be a valid state", func() {
			watch.previousRunTime = 500 * time.Millisecond
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
