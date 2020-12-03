package filewatcher

import (
	"encoding/json"
	"testing"
	"time"

	gomock "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AAgent/Watchers/ExecWatcher")
}

var _ = Describe("ExecWatcher", func() {
	Describe("setProperties", func() {
		It("Should parse valid properties", func() {
			w := &Watcher{}

			prop := map[string]interface{}{
				"path":                 "cmd",
				"gather_initial_state": "t",
			}
			Expect(w.setProperties(prop)).ToNot(HaveOccurred())
			Expect(w.path).To(Equal("cmd"))
			Expect(w.initial).To(BeTrue())
		})

		It("Should handle errors", func() {
			w := &Watcher{}
			err := w.setProperties(map[string]interface{}{})
			Expect(err).To(MatchError("path is required"))
		})
	})

	Describe("CurrentState", func() {
		var (
			mockctl     *gomock.Controller
			mockMachine *MockMachine
			watcher     *Watcher
			now         time.Time
		)

		BeforeEach(func() {
			mockctl = gomock.NewController(GinkgoT())
			mockMachine = NewMockMachine(mockctl)

			now = time.Unix(1606924953, 0)
			mockMachine.EXPECT().Name().Return("file").AnyTimes()
			mockMachine.EXPECT().Identity().Return("ginkgo").AnyTimes()
			mockMachine.EXPECT().InstanceID().Return("1234567890").AnyTimes()
			mockMachine.EXPECT().Version().Return("1.0.0").AnyTimes()
			mockMachine.EXPECT().TimeStampSeconds().Return(now.Unix()).AnyTimes()

			watcher = &Watcher{path: "/bin/sh", previous: Changed, machine: mockMachine, name: "ginkgo"}
		})

		AfterEach(func() {
			mockctl.Finish()
		})

		It("Should be a valid state", func() {
			cs := watcher.CurrentState()
			csj, err := cs.(*StateNotification).JSON()
			Expect(err).ToNot(HaveOccurred())

			event := map[string]interface{}{}
			err = json.Unmarshal(csj, &event)
			Expect(err).ToNot(HaveOccurred())
			delete(event, "id")

			Expect(event).To(Equal(map[string]interface{}{
				"time":        "2020-12-02T16:02:33Z",
				"type":        "io.choria.machine.watcher.file.v1.state",
				"subject":     "ginkgo",
				"specversion": "1.0",
				"source":      "io.choria.machine",
				"data": map[string]interface{}{
					"id":               "1234567890",
					"identity":         "ginkgo",
					"machine":          "file",
					"name":             "ginkgo",
					"protocol":         "io.choria.machine.watcher.file.v1.state",
					"type":             "file",
					"version":          "1.0.0",
					"timestamp":        float64(now.Unix()),
					"previous_outcome": "changed",
					"path":             "/bin/sh",
				},
			}))
		})
	})
})
