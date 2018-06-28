package client

import (
	"github.com/choria-io/go-protocol/protocol"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client/Filter", func() {
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
			Expect(pf.FactFilters()).To(Equal([][3]string{[3]string{"country", "==", "mt"}}))
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
			Expect(pf.FactFilters()).To(Equal([][3]string{[3]string{"country", "==", "mt"}, [3]string{"country", "==", "uk"}}))
		})

		It("Should handle errors", func() {
			err := FactFilter("foo")(pf)
			Expect(err).To(MatchError("Could not parse fact foo it does not appear to be in a valid format"))
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

	Describe("CompoundFilter", func() {
		It("Should add each filter", func() {
			CompoundFilter("foo", "/bar/", "baz", "")(pf)
			Expect(pf.CompoundFilters()).To(Equal([]string{"foo", "/bar/", "baz"}))
		})
	})

	Describe("CombinedFilter", func() {
		It("Should add each filter", func() {
			CombinedFilter("foo", "/bar/", "baz", "country=mt", "")(pf)
			Expect(pf.ClassFilters()).To(Equal([]string{"foo", "/bar/", "baz"}))
			Expect(pf.FactFilters()).To(Equal([][3]string{[3]string{"country", "==", "mt"}}))
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

		It("Should fail for facts in the wrong format", func() {
			pf, err := ParseFactFilterString("foo")
			Expect(err).To(MatchError("Could not parse fact foo it does not appear to be in a valid format"))
			Expect(pf).To(BeNil())
		})
	})
})
