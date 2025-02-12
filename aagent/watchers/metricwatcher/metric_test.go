// Copyright (c) 2020-2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package metricwatcher

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/choria-io/go-choria/aagent/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AAgent/Watchers/MetricWatcher")
}

var _ = Describe("MetricWatcher", func() {
	var (
		mockctl     *gomock.Controller
		mockMachine *model.MockMachine
		watch       *Watcher
		now         time.Time
		td          string
		err         error
	)

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())
		mockMachine = model.NewMockMachine(mockctl)

		td, err = os.MkdirTemp("", "")
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
		mockMachine.EXPECT().Facts().Return([]byte(`{"fqdn":"ginkgo.example.net"}`)).AnyTimes()
		mockMachine.EXPECT().Data().Return(map[string]any{}).AnyTimes()

		wi, err := New(mockMachine, "ginkgo", []string{"run"}, "fail", "success", "", time.Second, map[string]any{
			"command": "metric.sh",
		})
		Expect(err).ToNot(HaveOccurred())
		watch = wi.(*Watcher)
	})

	AfterEach(func() {
		mockctl.Finish()
		os.Remove(td)
	})

	Describe("performWatch", func() {
		It("Should run the script and correctly parse nagios style metrics", func(ctx context.Context) {
			if runtime.GOOS == "windows" {
				Skip("not tested on windows yet")
			}

			handled := false
			mockMachine.EXPECT().NotifyWatcherState("ginkgo", gomock.Any()).Do(func(_ string, m *StateNotification) {
				Expect(m.Metrics.Labels).To(Equal(map[string]string{
					"dupe":   "w",
					"format": "nagios",
				}))
				Expect(m.Metrics.Metrics["failed_events"]).To(Equal(0.0))
				Expect(m.Metrics.Metrics["failed_resources"]).To(Equal(0.0))
				Expect(m.Metrics.Metrics["last_run_duration"]).To(Equal(59.67))
				Expect(m.Metrics.Metrics["time_since_last_run"]).To(Equal(237.0))
				Expect(m.Metrics.Metrics["choria_runtime_seconds"]).To(BeNumerically(">", 0))
				handled = true
			})

			wi, err := New(mockMachine, "ginkgo", []string{"run"}, "fail", "success", "", time.Second, map[string]any{
				"command": filepath.Join("testdata", "nagios.sh"),
				"labels":  map[string]string{"dupe": "w"},
			})
			Expect(err).ToNot(HaveOccurred())
			watch = wi.(*Watcher)

			watch.performWatch(ctx)
			Expect(handled).To(BeTrue())
		})

		It("Should run the script and correctly parse choria style metrics", func(ctx context.Context) {
			if runtime.GOOS == "windows" {
				Skip("not tested on windows yet")
			}

			handled := false
			mockMachine.EXPECT().NotifyWatcherState("ginkgo", gomock.Any()).Do(func(_ string, m *StateNotification) {
				Expect(m.Metrics.Labels).To(Equal(map[string]string{
					"dupe":   "w",
					"unique": "u",
					"format": "choria",
				}))
				Expect(m.Metrics.Metrics["v1"]).To(Equal(float64(1)))
				Expect(m.Metrics.Metrics["v2"]).To(Equal(1.1))
				Expect(m.Metrics.Metrics["choria_runtime_seconds"]).To(BeNumerically(">", 0))
				handled = true
			})

			wi, err := New(mockMachine, "ginkgo", []string{"run"}, "fail", "success", "", time.Second, map[string]any{
				"command": filepath.Join("testdata", "metric.sh"),
				"labels":  map[string]string{"dupe": "w"},
			})
			Expect(err).ToNot(HaveOccurred())
			watch = wi.(*Watcher)

			watch.performWatch(ctx)
			Expect(handled).To(BeTrue())
		})
	})

	Describe("setProperties", func() {
		It("Should parse valid properties", func() {
			err := watch.setProperties(map[string]any{
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
			err = watch.setProperties(map[string]any{
				"interval": "500ms",
			})
			Expect(err).To(MatchError("command is required"))
		})

		It("Should enforce 1 second intervals", func() {
			err := watch.setProperties(map[string]any{
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

			event := map[string]any{}
			err = json.Unmarshal(csj, &event)
			Expect(err).ToNot(HaveOccurred())
			delete(event, "id")

			Expect(event).To(Equal(map[string]any{
				"time":            "2020-12-02T16:02:33Z",
				"type":            "io.choria.machine.watcher.metric.v1.state",
				"subject":         "ginkgo",
				"specversion":     "1.0",
				"source":          "io.choria.machine",
				"datacontenttype": "application/json",
				"data": map[string]any{
					"id":        "1234567890",
					"identity":  "ginkgo",
					"machine":   "metric",
					"name":      "ginkgo",
					"protocol":  "io.choria.machine.watcher.metric.v1.state",
					"type":      "metric",
					"version":   "1.0.0",
					"timestamp": float64(now.Unix()),
					"metrics": map[string]any{
						"labels": map[string]any{},
						"metrics": map[string]any{
							"choria_runtime_seconds": 0.5,
						},
						"time": float64(0),
					},
				},
			}))
		})
	})
})
