// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package aggregate

import (
	"sort"

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

			results, err := agg.ResultStrings()
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

			results, err := agg.ResultFormattedStrings("")
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(Equal([]string{
				"looooong: 3",
				"       1: 2",
				"     med: 1",
			}))

			results, err = agg.ResultFormattedStrings("%s: %d")
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(Equal([]string{
				"looooong: 3",
				"1: 2",
				"med: 1",
			}))
		})
	})

	Describe("JSONResults", func() {
		It("Should produce correct JSON", func() {
			Expect(agg.ProcessValue("med")).ToNot(HaveOccurred())
			Expect(agg.ProcessValue("looooong")).ToNot(HaveOccurred())
			Expect(agg.ProcessValue("looooong")).ToNot(HaveOccurred())
			Expect(agg.ProcessValue("looooong")).ToNot(HaveOccurred())
			Expect(agg.ProcessValue(1)).ToNot(HaveOccurred())
			Expect(agg.ProcessValue(1)).ToNot(HaveOccurred())

			jresults, err := agg.ResultJSON()
			Expect(err).ToNot(HaveOccurred())
			Expect(jresults).To(MatchJSON("{\"1\":2, \"looooong\":3, \"med\":1}"))
		})
	})

	Describe("Value mapping", func() {
		It("Should format correctly", func() {
			agg, err = NewSummaryAggregator([]interface{}{"item", map[string]interface{}{
				"true":  "enabled",
				"false": "disabled",
				"thing": "other thing",
				"1":     "one",
			}})
			Expect(err).ToNot(HaveOccurred())

			Expect(agg.ProcessValue("true")).ToNot(HaveOccurred())
			Expect(agg.ProcessValue(true)).ToNot(HaveOccurred())
			Expect(agg.ProcessValue("false")).ToNot(HaveOccurred())
			Expect(agg.ProcessValue(false)).ToNot(HaveOccurred())
			Expect(agg.ProcessValue("false")).ToNot(HaveOccurred())
			Expect(agg.ProcessValue("thing")).ToNot(HaveOccurred())
			Expect(agg.ProcessValue(1)).ToNot(HaveOccurred())
			Expect(agg.ProcessValue("1")).ToNot(HaveOccurred())
			Expect(agg.ProcessValue("not mapped")).ToNot(HaveOccurred())
			results, err := agg.ResultFormattedStrings("")
			Expect(err).ToNot(HaveOccurred())
			sort.Strings(results)
			Expect(results).To(Equal([]string{
				"        one: 2",
				"    enabled: 2",
				"   disabled: 3",
				" not mapped: 1",
				"other thing: 1",
			}))
		})
	})

	Describe("CustomFormat", func() {
		It("Should format correctly", func() {
			agg, err = NewSummaryAggregator([]interface{}{"item", map[string]interface{}{"format": "%s=%v"}})
			Expect(err).ToNot(HaveOccurred())

			Expect(agg.ProcessValue("med")).ToNot(HaveOccurred())
			Expect(agg.ProcessValue("looooong")).ToNot(HaveOccurred())
			Expect(agg.ProcessValue("looooong")).ToNot(HaveOccurred())
			Expect(agg.ProcessValue("looooong")).ToNot(HaveOccurred())
			Expect(agg.ProcessValue(1)).ToNot(HaveOccurred())
			Expect(agg.ProcessValue(1)).ToNot(HaveOccurred())
			results, err := agg.ResultFormattedStrings("")
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(Equal([]string{
				"looooong=3",
				"1=2",
				"med=1",
			}))
		})
	})
})
