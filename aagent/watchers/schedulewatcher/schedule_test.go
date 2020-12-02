package schedulewatcher

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AAgent/Watchers/ScheduleWatcher")
}

var _ = Describe("ScheduleWatcher", func() {
	Describe("setProperties", func() {
		It("Should parse valid properties", func() {
			w := &Watcher{}

			err := w.setProperties(map[string]interface{}{
				"duration":  "1h",
				"schedules": []string{"* * * * *", "1 * * * *"},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(w.duration).To(Equal(time.Hour))
			Expect(w.items).To(HaveLen(2))
			Expect(w.schedules).To(HaveLen(2))
			Expect(w.items[0].spec).To(Equal("* * * * *"))
			Expect(w.items[1].spec).To(Equal("1 * * * *"))
		})

		It("Should handle errors", func() {
			w := &Watcher{}

			err := w.setProperties(map[string]interface{}{})
			Expect(err).To(MatchError("no schedules defined"))

			w = &Watcher{}
			err = w.setProperties(map[string]interface{}{
				"schedules": []string{"* * * * *", "1 * * * *"},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(w.duration).To(Equal(time.Minute))
			Expect(w.items).To(HaveLen(2))
			Expect(w.schedules).To(HaveLen(2))
		})
	})
})
