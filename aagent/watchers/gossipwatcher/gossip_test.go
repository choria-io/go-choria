// Copyright (c) 2022-2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package gossipwatcher

import (
	"encoding/json"
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

		mockMachine.EXPECT().Name().Return("gossip").AnyTimes()
		mockMachine.EXPECT().Identity().Return("ginkgo").AnyTimes()
		mockMachine.EXPECT().InstanceID().Return("1234567890").AnyTimes()
		mockMachine.EXPECT().Version().Return("1.0.0").AnyTimes()
		mockMachine.EXPECT().TimeStampSeconds().Return(now.Unix()).AnyTimes()
		mockMachine.EXPECT().Debugf(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		mockMachine.EXPECT().Infof(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

		now = time.Unix(1606924953, 0)

		wi, err := New(mockMachine, "ginkgo", []string{"always"}, nil, "fail", "success", "10s", time.Second, map[string]any{
			"subject": "foo.bar",
			"payload": "msg.msg",
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
				"subject": "foo.bar",
				"payload": "pay.load",
			}
			Expect(watch.setProperties(prop)).To(Succeed())
			Expect(watch.properties.Subject).To(Equal("foo.bar"))
			Expect(watch.properties.Payload).To(Equal("pay.load"))
			Expect(watch.properties.Registration).To(BeNil())

			prop = map[string]any{
				"registration": map[string]any{
					"cluster":  "lon",
					"service":  "ginkgo",
					"protocol": "http",
					"ip":       "192.168.1.1",
					"port":     8080,
					"priority": 1,
					"annotations": map[string]string{
						"test": "annotation",
					},
				},
			}

			watch.properties = nil
			Expect(watch.setProperties(prop)).To(Succeed())
			Expect(watch.properties.Registration).To(Equal(&Registration{
				Cluster:  "lon",
				Service:  "ginkgo",
				Protocol: "http",
				IP:       "192.168.1.1",
				Port:     8080,
				Priority: 1,
				Annotations: map[string]string{
					"test": "annotation",
				},
			}))

			rj, err := json.Marshal(watch.properties.Registration)
			Expect(err).ToNot(HaveOccurred())

			Expect(watch.machine.InstanceID()).To(Equal("1234567890"))
			Expect(watch.properties.Subject).To(Equal("$KV.CHORIA_SERVICES.lon.http.ginkgo.1234567890"))
			Expect(watch.properties.Payload).To(Equal(string(rj)))
		})

		It("Should handle errors", func() {
			watch.properties = nil
			err := watch.setProperties(map[string]any{})
			Expect(err).To(MatchError("subject is required"))

			watch.properties = nil
			err = watch.setProperties(map[string]any{
				"subject": "foo.bar",
			})
			Expect(err).To(MatchError("payload is required"))
		})
	})

	Describe("CurrentState", func() {
		It("Should be a valid state", func() {
			watch.lastSubject = "x.y"
			watch.lastPayload = "a.b"
			watch.lastGossip = now

			cs := watch.CurrentState()
			csj, err := cs.(*StateNotification).JSON()
			Expect(err).ToNot(HaveOccurred())

			event := map[string]any{}
			err = json.Unmarshal(csj, &event)
			Expect(err).ToNot(HaveOccurred())
			delete(event, "id")

			Expect(event).To(Equal(map[string]any{
				"time":            "2020-12-02T16:02:33Z",
				"type":            "io.choria.machine.watcher.gossip.v1.state",
				"subject":         "ginkgo",
				"specversion":     "1.0",
				"source":          "io.choria.machine",
				"datacontenttype": "application/json",
				"data": map[string]any{
					"previous_subject": "x.y",
					"previous_payload": "a.b",
					"previous_gossip":  float64(now.Unix()),
					"id":               "1234567890",
					"identity":         "ginkgo",
					"machine":          "gossip",
					"name":             "ginkgo",
					"protocol":         "io.choria.machine.watcher.gossip.v1.state",
					"type":             "gossip",
					"version":          "1.0.0",
					"timestamp":        float64(now.Unix()),
				},
			}))
		})
	})
})
