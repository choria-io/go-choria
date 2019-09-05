package aggregate

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SummaryAggregator", func() {
	var (
		err error
		agg *SummaryAggregator
	)

	BeforeEach(func() {
		agg, err = NewSummaryAggregator([]interface{}{})
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("ProcessValue", func() {
		It("Should process various values", func() {
			Expect(agg.ProcessValue(1)).ToNot(HaveOccurred())
			Expect(agg.ProcessValue(1)).ToNot(HaveOccurred())
			Expect(agg.ProcessValue(1)).ToNot(HaveOccurred())
			Expect(agg.ProcessValue("a")).ToNot(HaveOccurred())

			results, err := agg.StringResults()
			Expect(err).ToNot(HaveOccurred())
			Expect(results["1"]).To(Equal("3"))
			Expect(results["a"]).To(Equal("1"))
		})
	})

	Describe("FormattedStrings", func() {
		It("Should calculate a correct width format", func() {
			Expect(agg.ProcessValue("med")).ToNot(HaveOccurred())
			Expect(agg.ProcessValue("looooong")).ToNot(HaveOccurred())
			Expect(agg.ProcessValue("looooong")).ToNot(HaveOccurred())
			Expect(agg.ProcessValue("looooong")).ToNot(HaveOccurred())
			Expect(agg.ProcessValue(1)).ToNot(HaveOccurred())
			Expect(agg.ProcessValue(1)).ToNot(HaveOccurred())

			results, err := agg.FormattedStrings("")
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(Equal([]string{
				"looooong: 3",
				"       1: 2",
				"     med: 1",
			}))

			results, err = agg.FormattedStrings("%s: %d")
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(Equal([]string{
				"looooong: 3",
				"1: 2",
				"med: 1",
			}))
		})
	})
})
