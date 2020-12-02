package timerwatcher

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AAgent/Watchers/TimerWatcher")
}

var _ = Describe("TimerWatcher", func() {
	Describe("setProperties", func() {
		It("Should parse valid properties", func() {
			w := &Watcher{}

			err := w.setProperties(map[string]interface{}{
				"timer": "1h",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(w.time).To(Equal(time.Hour))
		})

		It("Should handle errors", func() {
			w := &Watcher{}
			err := w.setProperties(map[string]interface{}{})
			Expect(err).ToNot(HaveOccurred())
			Expect(w.time).To(Equal(time.Second))
		})
	})
})
