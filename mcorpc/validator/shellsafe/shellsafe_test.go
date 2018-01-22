package shellsafe

import (
	"fmt"
	"reflect"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestFileContent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "McoRPC/Validator/ShellSafe")
}

var _ = Describe("Validate", func() {
	It("Should match bad strings", func() {
		badchars := []string{"`", "$", ";", "|", "&&", ">", "<"}

		for _, c := range badchars {
			ok, err := Validate(fmt.Sprintf("thing%sthing", c))
			Expect(err).To(MatchError(fmt.Sprintf("may not contain '%s'", c)))
			Expect(ok).To(BeFalse())
		}
	})

	It("Should allow good things", func() {
		Expect(Validate("ok")).To(BeTrue())
		Expect(Validate("")).To(BeTrue())
		Expect(Validate("ok ok ok")).To(BeTrue())
	})
})

var _ = Describe("ValidateStructField", func() {
	type t struct {
		String string `validate:"shellsafe"`
	}

	It("Should validate string fields", func() {
		st := t{"not > safe"}

		val := reflect.ValueOf(st)
		valueField := val.Field(0)
		typeField := val.Type().Field(0)

		ok, err := ValidateStructField(valueField, typeField.Tag.Get("validate"))
		Expect(err).To(MatchError("may not contain '>'"))
		Expect(ok).To(BeFalse())

		st = t{"safe"}

		val = reflect.ValueOf(st)
		valueField = val.Field(0)

		ok, err = ValidateStructField(valueField, typeField.Tag.Get("validate"))
		Expect(err).ToNot(HaveOccurred())
		Expect(ok).To(BeTrue())
	})
})
