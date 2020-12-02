package nagioswatcher

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AAgent/Watchers/NagiosWatcher")
}

var _ = Describe("NagiosWatcher", func() {
	Describe("setProperties", func() {
		It("Should parse valid properties", func() {
			w := &Watcher{}

			err := w.setProperties(map[string]interface{}{
				"annotations": map[string]string{
					"a1": "v1",
					"a2": "v2",
				},
				"plugin":  "cmd",
				"timeout": "5s",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(w.annotations).To(Equal(map[string]string{
				"a1": "v1",
				"a2": "v2",
			}))
			Expect(w.plugin).To(Equal("cmd"))
			Expect(w.timeout).To(Equal(5 * time.Second))
			Expect(w.builtin).To(BeEmpty())
			Expect(w.gossFile).To(BeEmpty())
		})

		It("Should handle errors", func() {
			w := &Watcher{}
			err := w.setProperties(map[string]interface{}{})
			Expect(err).To(MatchError("plugin or builtin is required"))

			w = &Watcher{}
			err = w.setProperties(map[string]interface{}{
				"plugin":  "cmd",
				"builtin": "goss",
			})
			Expect(err).To(MatchError("cannot set plugin and builtin"))

			w = &Watcher{}
			err = w.setProperties(map[string]interface{}{
				"builtin": "goss",
			})
			Expect(err).To(MatchError("gossfile property is required for the goss builtin check"))
		})

		It("Should handle valid goss setups", func() {
			w := &Watcher{}
			err := w.setProperties(map[string]interface{}{
				"builtin":  "goss",
				"gossFile": "/x",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(w.builtin).To(Equal("goss"))
			Expect(w.gossFile).To(Equal("/x"))
		})
	})
})
