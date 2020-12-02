package homekitwatcher

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AAgent/Watchers/HomekitWatcher")
}

var _ = Describe("HomekitWatcher", func() {
	Describe("setProperties", func() {
		It("Should parse valid properties", func() {
			w := &Watcher{}

			prop := map[string]interface{}{
				"serial_number": "123456",
				"model":         "Ginkgo",
				"pin":           "12345678",
				"setup_id":      "1234",
				"on_when":       []string{"on"},
				"off_when":      []string{"off"},
				"disable_when":  []string{"disable"},
				"initial":       "true",
			}
			Expect(w.setProperties(prop)).ToNot(HaveOccurred())
			Expect(w.serialNumber).To(Equal("123456"))
			Expect(w.model).To(Equal("Ginkgo"))
			Expect(w.pin).To(Equal("12345678"))
			Expect(w.setupID).To(Equal("1234"))
			Expect(w.shouldOn).To(Equal([]string{"on"}))
			Expect(w.shouldOff).To(Equal([]string{"off"}))
			Expect(w.shouldDisable).To(Equal([]string{"disable"}))
			Expect(w.initial).To(Equal(On))
		})

		It("Should handle initial correctly", func() {
			w := &Watcher{}
			Expect(w.setProperties(map[string]interface{}{})).ToNot(HaveOccurred())
			Expect(w.initial).To(Equal(Off))

			w = &Watcher{}
			Expect(w.setProperties(map[string]interface{}{"initial": "true"})).ToNot(HaveOccurred())
			Expect(w.initial).To(Equal(On))

			w = &Watcher{}
			Expect(w.setProperties(map[string]interface{}{"initial": "false"})).ToNot(HaveOccurred())
			Expect(w.initial).To(Equal(Off))
		})

		It("Should handle errors", func() {
			w := &Watcher{}
			err := w.setProperties(map[string]interface{}{
				"pin": "1",
			})
			Expect(err).To(MatchError("pin should be 8 characters long"))

			err = w.setProperties(map[string]interface{}{
				"pin":      "12345678",
				"setup_id": 1,
			})
			Expect(err).To(MatchError("setup_id should be 4 characters long"))

			err = w.setProperties(map[string]interface{}{
				"pin":      "12345678",
				"setup_id": 1234,
			})
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
