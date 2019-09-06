package aggregate

import (
	"encoding/json"

	"github.com/guptarohit/asciigraph"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ChartAggregator", func() {
	var (
		err error
		agg *ChartAggregator
	)

	BeforeEach(func() {
		agg, err = NewChartAggregator([]interface{}{})
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("ProcessValue", func() {
		It("Should process various values", func() {
			Expect(agg.ProcessValue(1)).ToNot(HaveOccurred())
			Expect(agg.ProcessValue(7)).ToNot(HaveOccurred())
			Expect(agg.ProcessValue("50")).ToNot(HaveOccurred())

			Expect(agg.items).To(Equal([]float64{1, 7, 50}))
		})
	})

	Describe("Results", func() {
		var expected string

		BeforeEach(func() {
			Expect(agg.ProcessValue(1)).ToNot(HaveOccurred())
			Expect(agg.ProcessValue(7)).ToNot(HaveOccurred())
			Expect(agg.ProcessValue("50")).ToNot(HaveOccurred())

			Expect(agg.items).To(Equal([]float64{1, 7, 50}))
			expected = asciigraph.Plot([]float64{1, 7, 50}, asciigraph.Height(15), asciigraph.Width(60), asciigraph.Offset(5))
		})

		Describe("ResultStrings", func() {
			It("Should produce the correct result", func() {
				results, err := agg.ResultStrings()
				Expect(err).ToNot(HaveOccurred())
				Expect(results["Chart"]).To(Equal(expected))
			})
		})

		Describe("ResultJSON", func() {
			It("Should produce the correct result", func() {
				results, err := agg.ResultJSON()
				Expect(err).ToNot(HaveOccurred())

				jexpected, err := json.Marshal(map[string]string{
					"chart": expected,
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(results).To(Equal(jexpected))
			})
		})

		Describe("ResultFormattedStrings", func() {
			It("Should produce the correct result", func() {
				results, err := agg.ResultFormattedStrings("")
				Expect(err).ToNot(HaveOccurred())
				Expect(results[0]).To(Equal(expected))
			})
		})
	})
})
