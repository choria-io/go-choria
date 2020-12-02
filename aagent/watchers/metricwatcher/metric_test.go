package metricwatcher

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AAgent/Watchers/MetricWatcher")
}

var _ = Describe("MetricWatcher", func() {
	Describe("setProperties", func() {
		It("Should parse valid properties", func() {
			w := &Watcher{}

			err := w.setProperties(map[string]interface{}{
				"command":  "cmd",
				"interval": "1s",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(w.command).To(Equal("cmd"))
			Expect(w.checkInterval).To(Equal(time.Second))
		})

		It("Should handle errors", func() {
			w := &Watcher{}
			err := w.setProperties(map[string]interface{}{
				"interval": "500ms",
			})
			Expect(err).To(MatchError("command is required"))
		})

		It("Should enforce 1 second intervals", func() {
			w := &Watcher{}
			err := w.setProperties(map[string]interface{}{
				"command":  "cmd",
				"interval": "500ms",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(w.command).To(Equal("cmd"))
			Expect(w.checkInterval).To(Equal(time.Second))
		})
	})
})
