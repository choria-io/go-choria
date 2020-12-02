package filewatcher

import (
	"testing"

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
})
