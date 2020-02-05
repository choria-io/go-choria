package regex

import (
	"reflect"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestFileContent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Validator/Regex")
}

var _ = Describe("ValidateString", func() {
	It("Should match strings correctly", func() {
		ok, err := ValidateString("hello world", "world$")
		Expect(err).ToNot(HaveOccurred())
		Expect(ok).To(BeTrue())

		ok, err = ValidateString("hello", "world$")
		Expect(err).To(MatchError("input does not match 'world$'"))
		Expect(ok).To(BeFalse())

		ok, err = ValidateString("hello", "invalid(")
		Expect(err).To(MatchError("invalid regex 'invalid('"))
		Expect(ok).To(BeFalse())
	})
})

var _ = Describe("ValidateStructField", func() {
	type t struct {
		String  string `validate:"regex=world$"`
		Invalid string `validate:"regex"`
	}

	It("Should fail for invalid tags", func() {
		st := t{"1", "foo"}

		val := reflect.ValueOf(st)
		valueField := val.FieldByName("Invalid")
		typeField, _ := val.Type().FieldByName("Invalid")

		ok, err := ValidateStructField(valueField, typeField.Tag.Get("validate"))
		Expect(err).To(MatchError("invalid tag 'regex', must be in the form regex=^hello.+world$"))
		Expect(ok).To(BeFalse())
	})

	It("Should match the regex correctly", func() {
		st := t{"fail", "foo"}

		val := reflect.ValueOf(st)
		valueField := val.FieldByName("String")
		typeField, _ := val.Type().FieldByName("String")

		ok, err := ValidateStructField(valueField, typeField.Tag.Get("validate"))
		Expect(err).To(MatchError("input does not match 'world$'"))
		Expect(ok).To(BeFalse())

		st.String = "hello world"
		val = reflect.ValueOf(st)
		valueField = val.FieldByName("String")

		ok, err = ValidateStructField(valueField, typeField.Tag.Get("validate"))
		Expect(err).ToNot(HaveOccurred())
		Expect(ok).To(BeTrue())
	})
})
