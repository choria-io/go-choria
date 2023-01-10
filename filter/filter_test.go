// Copyright (c) 2019-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package filter

import (
	"fmt"
	"testing"

	"github.com/choria-io/go-choria/protocol"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestGinkgo(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Filter")
}

var _ = Describe("Filter", func() {
	var (
		pf *protocol.Filter
	)

	BeforeEach(func() {
		pf = protocol.NewFilter()
	})

	Describe("NewFilter", func() {
		It("Should add all the filters", func() {
			pf, err := NewFilter(ClassFilter("klass"), AgentFilter("agent"), IdentityFilter("ident"), FactFilter("country=mt"))
			Expect(err).ToNot(HaveOccurred())
			Expect(pf.ClassFilters()).To(Equal([]string{"klass"}))
			Expect(pf.AgentFilters()).To(Equal([]string{"agent"}))
			Expect(pf.IdentityFilters()).To(Equal([]string{"ident"}))
			Expect(pf.FactFilters()).To(Equal([][3]string{{"country", "==", "mt"}}))
		})

		It("Should handle empty filter lists", func() {
			pf, err := NewFilter()
			Expect(err).ToNot(HaveOccurred())
			Expect(pf.ClassFilters()).To(BeEmpty())
			Expect(pf.AgentFilters()).To(BeEmpty())
			Expect(pf.IdentityFilters()).To(BeEmpty())
			Expect(pf.FactFilters()).To(BeEmpty())
		})
	})

	Describe("FactFilter", func() {
		It("Should add each filter", func() {
			err := FactFilter("country=mt", "country=uk", "")(pf)
			Expect(err).ToNot(HaveOccurred())
			Expect(pf.FactFilters()).To(Equal([][3]string{{"country", "==", "mt"}, {"country", "==", "uk"}}))
		})

		It("Should handle errors", func() {
			err := FactFilter("foo")(pf)
			Expect(err).To(MatchError("could not parse fact foo it does not appear to be in a valid format"))
		})
	})

	Describe("AgentFilter", func() {
		It("Should add each filter", func() {
			AgentFilter("foo", "/bar/", "baz", "")(pf)
			Expect(pf.AgentFilters()).To(Equal([]string{"foo", "/bar/", "baz"}))
		})
	})

	Describe("ClassFilter", func() {
		It("Should add each filter", func() {
			ClassFilter("foo", "/bar/", "baz", "")(pf)
			Expect(pf.ClassFilters()).To(Equal([]string{"foo", "/bar/", "baz"}))
		})
	})

	Describe("IdentityFilter", func() {
		It("Should add each filter", func() {
			IdentityFilter("foo", "/bar/", "baz", "")(pf)
			Expect(pf.IdentityFilters()).To(Equal([]string{"foo", "/bar/", "baz"}))
		})
	})

	// disabled while compound filters are not supported
	// Describe("CompoundFilter", func() {
	// 	It("Should add each filter", func() {
	// 		CompoundFilter("foo", "/bar/", "baz", "")(pf)
	// 		Expect(pf.CompoundFilters()).To(Equal([]string{"foo", "/bar/", "baz"}))
	// 	})
	// })

	Describe("CombinedFilter", func() {
		It("Should add each filter", func() {
			CombinedFilter("foo", "/bar/", "baz", "country=mt", "")(pf)
			Expect(pf.ClassFilters()).To(Equal([]string{"foo", "/bar/", "baz"}))
			Expect(pf.FactFilters()).To(Equal([][3]string{{"country", "==", "mt"}}))
		})
	})

	Describe("ParseFactFilterString", func() {
		var t = func(filter, f, o, v string) {
			pf, err := ParseFactFilterString(filter)
			Expect(err).ToNot(HaveOccurred())
			Expect(pf.Fact).To(Equal(f))
			Expect(pf.Operator).To(Equal(o))
			Expect(pf.Value).To(Equal(v))
		}

		It("Should parse old style regex fact matches", func() {
			t("foo=/bar/", "foo", "=~", "/bar/")
			t("foo = /bar/", "foo", "=~", "/bar/")
		})

		It("Should parse old style equality", func() {
			t("foo=bar", "foo", "==", "bar")
			t("foo = bar", "foo", "==", "bar")
		})

		It("Should parse regex fact matches", func() {
			t("foo=~bar", "foo", "=~", "bar")
			t("foo =~ bar", "foo", "=~", "bar")
		})

		It("Should treat => like >=", func() {
			t("foo=>bar", "foo", ">=", "bar")
			t("foo => bar", "foo", ">=", "bar")
		})

		It("Should treat =< like <=", func() {
			t("foo=<bar", "foo", "<=", "bar")
			t("foo =< bar", "foo", "<=", "bar")
		})

		It("Should parse less than or equal", func() {
			t("foo<=bar", "foo", "<=", "bar")
			t("foo <= bar", "foo", "<=", "bar")
		})

		It("Should parse greater than or equal", func() {
			t("foo>=bar", "foo", ">=", "bar")
			t("foo >= bar", "foo", ">=", "bar")
		})

		It("Should parse less than", func() {
			t("foo<bar", "foo", "<", "bar")
			t("foo < bar", "foo", "<", "bar")
		})

		It("Should parse greater than", func() {
			t("foo>bar", "foo", ">", "bar")
			t("foo > bar", "foo", ">", "bar")
		})

		It("Should parse not equal", func() {
			t("foo!=bar", "foo", "!=", "bar")
			t("foo != bar", "foo", "!=", "bar")
		})

		It("Should parse equal", func() {
			t("foo==bar", "foo", "==", "bar")
			t("foo == bar", "foo", "==", "bar")
		})

		It("Should parse gjson facts", func() {
			t("storage.#(name=\"nvme0n1\").size==64", "storage.#(name=\"nvme0n1\").size", "==", "64")
			t("storage.#(name='nvme0n1').size==64", "storage.#(name='nvme0n1').size", "==", "64")
			t("storage.#(name==\"nvme0n1\").size=64", "storage.#(name==\"nvme0n1\").size", "==", "64")
			t("storage.#(name=\"nvme0n1\").size == 64", "storage.#(name=\"nvme0n1\").size", "==", "64")
			t("storage.#(name<= \"nvme0n1\" ).size==64", "storage.#(name<= \"nvme0n1\" ).size", "==", "64")
			t("storage.#(name=\"foo bar\").size=>baz bar", "storage.#(name=\"foo bar\").size", ">=", "baz bar")
		})

		It("Should fail on invalid fact filters", func() {
			badFilters := []string{"foobarbaz", "=foo(bar=baz)", "foo=", "foo(bar=baz)>", "=><=="}
			for _, filterString := range badFilters {
				_, err := ParseFactFilterString(filterString)
				Expect(err).To(MatchError(fmt.Errorf("could not parse fact %s it does not appear to be in a valid format", filterString)))
			}
		})
	})
})
