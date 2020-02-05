package aggregate

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("AverageAggregator", func() {
	var (
		err error
		agg *AverageAggregator
	)

	BeforeEach(func() {
		agg, err = NewAverageAggregator([]interface{}{})
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("ProcessValue", func() {
		It("Should process various values", func() {
			Expect(agg.ProcessValue(1)).ToNot(HaveOccurred())
			Expect(agg.ProcessValue(1.1)).ToNot(HaveOccurred())
			Expect(agg.ProcessValue(int64(100))).ToNot(HaveOccurred())
			Expect(agg.ProcessValue("1")).ToNot(HaveOccurred())
			Expect(agg.ProcessValue("a")).To(HaveOccurred())

			results, err := agg.ResultStrings()
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(Equal(map[string]string{
				"Average": "25.775000",
			}))

			fresults, err := agg.ResultFormattedStrings("")
			Expect(err).ToNot(HaveOccurred())
			Expect(fresults).To(Equal([]string{
				"Average: 25.775",
			}))

			jresults, err := agg.ResultJSON()
			Expect(err).ToNot(HaveOccurred())
			Expect(jresults).To(MatchJSON("{\"average\":25.775}"))
		})
	})
})
