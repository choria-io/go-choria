// Copyright (c) 2020-2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package filewatcher

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/choria-io/go-choria/aagent/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AAgent/Watchers/ExecWatcher")
}

var _ = Describe("ExecWatcher", func() {
	var (
		mockctl     *gomock.Controller
		mockMachine *model.MockMachine
		watch       *Watcher
		now         time.Time
	)

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())
		mockMachine = model.NewMockMachine(mockctl)

		mockMachine.EXPECT().Name().Return("file").AnyTimes()
		mockMachine.EXPECT().Identity().Return("ginkgo").AnyTimes()
		mockMachine.EXPECT().InstanceID().Return("1234567890").AnyTimes()
		mockMachine.EXPECT().Version().Return("1.0.0").AnyTimes()
		mockMachine.EXPECT().TimeStampSeconds().Return(now.Unix()).AnyTimes()
		mockMachine.EXPECT().Directory().Return(".").AnyTimes()

		now = time.Unix(1606924953, 0)

		wi, err := New(mockMachine, "ginkgo", []string{"always"}, nil, "fail", "success", "2m", time.Second, map[string]any{
			"path": filepath.Join("bin", "sh"),
		})
		Expect(err).ToNot(HaveOccurred())
		watch = wi.(*Watcher)
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	Describe("setProperties", func() {
		It("Should parse valid properties", func() {
			prop := map[string]any{
				"path":                 "cmd",
				"gather_initial_state": "t",
			}
			Expect(watch.setProperties(prop)).ToNot(HaveOccurred())
			Expect(watch.properties.Path).To(Equal("cmd"))
			Expect(watch.properties.Initial).To(BeTrue())
		})

		It("Should handle errors", func() {
			watch.properties = &Properties{}
			err := watch.setProperties(map[string]any{})
			Expect(err).To(MatchError("path is required"))
		})
	})

	Describe("CurrentState", func() {
		It("Should be a valid state", func() {
			watch.previous = Changed
			cs := watch.CurrentState()
			csj, err := cs.(*StateNotification).JSON()
			Expect(err).ToNot(HaveOccurred())

			event := map[string]any{}
			err = json.Unmarshal(csj, &event)
			Expect(err).ToNot(HaveOccurred())
			delete(event, "id")

			Expect(event).To(Equal(map[string]any{
				"time":            "2020-12-02T16:02:33Z",
				"type":            "io.choria.machine.watcher.file.v1.state",
				"subject":         "ginkgo",
				"specversion":     "1.0",
				"source":          "io.choria.machine",
				"datacontenttype": "application/json",
				"data": map[string]any{
					"id":               "1234567890",
					"identity":         "ginkgo",
					"machine":          "file",
					"name":             "ginkgo",
					"protocol":         "io.choria.machine.watcher.file.v1.state",
					"type":             "file",
					"version":          "1.0.0",
					"timestamp":        float64(now.Unix()),
					"previous_outcome": "changed",
					"path":             filepath.Join("bin", "sh"),
				},
			}))
		})
	})
})
