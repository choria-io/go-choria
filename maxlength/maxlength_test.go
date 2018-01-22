package maxlength

import (
	"reflect"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestFileContent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "McoRPC/Validator/Maxlength")
}

var _ = Describe("ValidateString", func() {
	It("Should allow short strings", func() {
		ok, err := ValidateString("foo", 4)
		Expect(err).ToNot(HaveOccurred())
		Expect(ok).To(BeTrue())
	})

	It("Should fail on long strings", func() {
		ok, err := ValidateString("foo", 1)
		Expect(err).To(MatchError("3 characters, max allowed 1"))
		Expect(ok).To(BeFalse())
	})
})

var _ = Describe("ValidateStructField", func() {
	type t struct {
		Broken     string   `validate:"maxlength"`
		String     string   `validate:"maxlength=3"`
		Slice      []string `validate:"maxlength=3"`
		Untestable bool     `validate:"maxlength=3"`
	}

	var (
		st t
	)

	BeforeEach(func() {
		st = t{}
	})

	It("Should fail for invalid tags", func() {
		val := reflect.ValueOf(st)
		valueField := val.Field(0)
		typeField := val.Type().Field(0)

		ok, err := ValidateStructField(valueField, typeField.Tag.Get("validate"))
		Expect(err).To(MatchError("invalid tag 'maxlength', must be maxlength=n"))
		Expect(ok).To(BeFalse())
	})

	It("Should validate strings", func() {
		st.String = "foo"

		val := reflect.ValueOf(st)
		valueField := val.FieldByName("String")
		typeField, _ := val.Type().FieldByName("String")

		ok, err := ValidateStructField(valueField, typeField.Tag.Get("validate"))
		Expect(err).ToNot(HaveOccurred())
		Expect(ok).To(BeTrue())

		st.String = "foo foo foo"
		val = reflect.ValueOf(st)
		valueField = val.Field(1)

		ok, err = ValidateStructField(valueField, typeField.Tag.Get("validate"))
		Expect(err).To(MatchError("11 characters, max allowed 3"))
		Expect(ok).To(BeFalse())
	})

	It("Should validate slices", func() {
		st.Slice = []string{"one", "two", "three", "four"}

		val := reflect.ValueOf(st)
		valueField := val.FieldByName("Slice")
		typeField, _ := val.Type().FieldByName("Slice")

		ok, err := ValidateStructField(valueField, typeField.Tag.Get("validate"))
		Expect(err).To(MatchError("4 values, max allowed 3"))
		Expect(ok).To(BeFalse())

		st.Slice = []string{"one", "two", "three"}

		val = reflect.ValueOf(st)
		valueField = val.FieldByName("Slice")

		ok, err = ValidateStructField(valueField, typeField.Tag.Get("validate"))
		Expect(err).ToNot(HaveOccurred())
		Expect(ok).To(BeTrue())

	})

	It("Should fail for invalid data", func() {
		val := reflect.ValueOf(st)
		valueField := val.FieldByName("Untestable")
		typeField, _ := val.Type().FieldByName("Untestable")

		ok, err := ValidateStructField(valueField, typeField.Tag.Get("validate"))
		Expect(err).To(MatchError("cannot check length of bool type"))
		Expect(ok).To(BeFalse())
	})
})
