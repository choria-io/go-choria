// Copyright (c) 2020-2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package homekitwatcher

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
	RunSpecs(t, "AAgent/Watchers/HomekitWatcher")
}

var _ = Describe("HomekitWatcher", func() {
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
		mockMachine.EXPECT().Name().Return("homekit").AnyTimes()
		mockMachine.EXPECT().Identity().Return("ginkgo").AnyTimes()
		mockMachine.EXPECT().InstanceID().Return("1234567890").AnyTimes()
		mockMachine.EXPECT().Version().Return("1.0.0").AnyTimes()
		mockMachine.EXPECT().TimeStampSeconds().Return(now.Unix()).AnyTimes()
		mockMachine.EXPECT().Directory().Return(".").AnyTimes()

		wi, err := New(mockMachine, "ginkgo", []string{"always"}, nil, "fail", "success", "2m", time.Second, map[string]any{})
		Expect(err).ToNot(HaveOccurred())
		watch = wi.(*Watcher)
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	Describe("setProperties", func() {
		It("Should parse valid properties", func() {
			prop := map[string]any{
				"serial_number": "123456",
				"model":         "Ginkgo",
				"pin":           "12345678",
				"setup_id":      "1234",
				"on_when":       []string{"on"},
				"off_when":      []string{"off"},
				"disable_when":  []string{"disable"},
				"initial":       "true",
			}
			Expect(watch.setProperties(prop)).ToNot(HaveOccurred())
			Expect(watch.properties.SerialNumber).To(Equal("123456"))
			Expect(watch.properties.Model).To(Equal("Ginkgo"))
			Expect(watch.properties.Pin).To(Equal("12345678"))
			Expect(watch.properties.SetupId).To(Equal("1234"))
			Expect(watch.properties.ShouldOn).To(Equal([]string{"on"}))
			Expect(watch.properties.ShouldOff).To(Equal([]string{"off"}))
			Expect(watch.properties.ShouldDisable).To(Equal([]string{"disable"}))
			Expect(watch.properties.InitialState).To(Equal(On))
		})

		It("Should handle initial correctly", func() {
			watch.properties = &properties{Path: "/bin/sh"}
			Expect(watch.setProperties(map[string]any{})).ToNot(HaveOccurred())
			Expect(watch.properties.InitialState).To(Equal(Unknown))

			watch.properties = &properties{Path: "/bin/sh"}
			Expect(watch.setProperties(map[string]any{"initial": "true"})).ToNot(HaveOccurred())
			Expect(watch.properties.InitialState).To(Equal(On))

			watch.properties = &properties{Path: "/bin/sh"}
			Expect(watch.setProperties(map[string]any{"initial": "false"})).ToNot(HaveOccurred())
			Expect(watch.properties.InitialState).To(Equal(Off))
		})

		It("Should handle errors", func() {
			watch.properties = &properties{Path: "/bin/sh"}
			err := watch.setProperties(map[string]any{
				"pin": "1",
			})
			Expect(err).To(MatchError("pin should be 8 characters long"))

			watch.properties = &properties{Path: "/bin/sh"}
			err = watch.setProperties(map[string]any{
				"pin":      "12345678",
				"setup_id": 1,
			})
			Expect(err).To(MatchError("setup_id should be 4 characters long"))

			watch.properties = &properties{Path: "/bin/sh"}
			err = watch.setProperties(map[string]any{
				"pin":      "12345678",
				"setup_id": 1234,
			})
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("CurrentState", func() {
		It("Should be a valid state", func() {
			watch.previous = Off
			cs := watch.CurrentState()
			csj, err := cs.(*StateNotification).JSON()
			Expect(err).ToNot(HaveOccurred())

			event := map[string]any{}
			err = json.Unmarshal(csj, &event)
			Expect(err).ToNot(HaveOccurred())
			delete(event, "id")

			Expect(event).To(Equal(map[string]any{
				"time":            "2020-12-02T16:02:33Z",
				"type":            "io.choria.machine.watcher.homekit.v1.state",
				"subject":         "ginkgo",
				"specversion":     "1.0",
				"source":          "io.choria.machine",
				"datacontenttype": "application/json",
				"data": map[string]any{
					"id":               "1234567890",
					"identity":         "ginkgo",
					"machine":          "homekit",
					"name":             "ginkgo",
					"protocol":         "io.choria.machine.watcher.homekit.v1.state",
					"type":             "homekit",
					"version":          "1.0.0",
					"timestamp":        float64(now.Unix()),
					"previous_outcome": "off",
					"path":             filepath.Join("homekit", "4bb6777bb903cae3166e826932f7fe94"),
				},
			}))
		})
	})
})
