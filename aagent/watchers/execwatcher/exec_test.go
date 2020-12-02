package execwatcher

import (
	"testing"
	"time"

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
				"command":                   "cmd",
				"timeout":                   "1.5s",
				"environment":               []string{"key1=val1", "key2=val2"},
				"suppress_success_announce": "true",
			}
			Expect(w.setProperties(prop)).ToNot(HaveOccurred())
			Expect(w.command).To(Equal("cmd"))
			Expect(w.timeout).To(Equal(1500 * time.Millisecond))
			Expect(w.environment).To(Equal([]string{"key1=val1", "key2=val2"}))
			Expect(w.suppressSuccessAnnounce).To(BeTrue())
		})

		It("Should handle errors", func() {
			w := &Watcher{}
			err := w.setProperties(map[string]interface{}{})
			Expect(err).To(MatchError("command is required"))
		})

		It("Should enforce 1 second intervals", func() {
			w := &Watcher{}
			err := w.setProperties(map[string]interface{}{
				"command": "cmd",
				"timeout": "0",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(w.command).To(Equal("cmd"))
			Expect(w.timeout).To(Equal(time.Second))
		})
	})
})
