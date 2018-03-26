package duration

import (
	"reflect"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestFileContent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Validator/Duration")
}

var _ = Describe("ValidateString", func() {
	It("Should match durations correctly", func() {
		ok, err := ValidateString("1s")
		Expect(err).ToNot(HaveOccurred())
		Expect(ok).To(BeTrue())

		ok, err = ValidateString("1h")
		Expect(err).ToNot(HaveOccurred())
		Expect(ok).To(BeTrue())

		ok, err = ValidateString("1w")
		Expect(err).To(MatchError("time: unknown unit w in duration 1w"))
		Expect(ok).To(BeFalse())
	})
})

var _ = Describe("ValidateStructField", func() {
	type t struct {
		Interval string `validate:"duration"`
	}

	It("Should validate the struct correctly", func() {
		st := t{"1h"}

		val := reflect.ValueOf(st)
		valueField := val.FieldByName("Interval")
		typeField, _ := val.Type().FieldByName("Interval")

		ok, err := ValidateStructField(valueField, typeField.Tag.Get("validate"))
		Expect(err).ToNot(HaveOccurred())
		Expect(ok).To(BeTrue())

		st.Interval = "foo"
		valueField = reflect.ValueOf(st).FieldByName("Interval")
		ok, err = ValidateStructField(valueField, typeField.Tag.Get("validate"))
		Expect(err).To(MatchError("time: invalid duration foo"))
		Expect(ok).To(BeFalse())
	})
})
