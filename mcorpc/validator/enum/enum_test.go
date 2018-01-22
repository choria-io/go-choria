package enum

import (
	"reflect"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestFileContent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "McoRPC/Validator/Enum")
}

var _ = Describe("ValidateSlice", func() {
	It("Should match all good", func() {
		ok, err := ValidateSlice([]string{"1", "2", "3"}, []string{"1", "2", "3"})
		Expect(err).ToNot(HaveOccurred())
		Expect(ok).To(BeTrue())
	})

	It("Should catch errors", func() {
		ok, err := ValidateSlice([]string{"1", "2", "3"}, []string{"1", "2"})
		Expect(err).To(MatchError("'3' is not in the allowed list: 1, 2"))
		Expect(ok).To(BeFalse())
	})
})

var _ = Describe("ValidateString", func() {
	type st struct {
		Thing   string `validate:"enum=one,two"`
		Invalid int    `validate:"enum=one,two"`
	}

	It("Should validate string fields", func() {
		ok, err := ValidateString("two", []string{"one", "two"})
		Expect(err).ToNot(HaveOccurred())
		Expect(ok).To(BeTrue())

		ok, err = ValidateString("two", []string{"one", "three"})
		Expect(err).To(MatchError("'two' is not in the allowed list: one, three"))
		Expect(ok).To(BeFalse())
	})
})

var _ = Describe("ValidateStructField", func() {
	type st struct {
		Things  []string `validate:"enum=one,two"`
		Thing   string   `validate:"enum=one,two"`
		Invalid int      `validate:"enum=one,two"`
	}

	It("Should validate the enum field", func() {
		things := st{[]string{"one", "two", "three"}, "three", 1}

		val := reflect.ValueOf(things)
		valueField := val.FieldByName("Things")
		typeField, _ := val.Type().FieldByName("Things")

		ok, err := ValidateStructField(valueField, typeField.Tag.Get("validate"))
		Expect(err).To(MatchError("'three' is not in the allowed list: one, two"))
		Expect(ok).To(BeFalse())

		things = st{[]string{"one", "two"}, "three", 2}
		val = reflect.ValueOf(things)
		valueField = val.FieldByName("Things")

		ok, err = ValidateStructField(valueField, typeField.Tag.Get("validate"))
		Expect(err).ToNot(HaveOccurred())
		Expect(ok).To(BeTrue())

		valueField = val.FieldByName("Thing")

		ok, err = ValidateStructField(valueField, typeField.Tag.Get("validate"))
		Expect(err).To(MatchError("'three' is not in the allowed list: one, two"))
		Expect(ok).To(BeFalse())

	})

	It("Should validate only supported types", func() {
		things := st{[]string{"one", "two", "three"}, "three", 1}

		val := reflect.ValueOf(things)
		valueField := val.FieldByName("Invalid")
		typeField, _ := val.Type().FieldByName("Invalid")

		ok, err := ValidateStructField(valueField, typeField.Tag.Get("validate"))
		Expect(err).To(MatchError("cannot valid data of type int for enums"))
		Expect(ok).To(BeFalse())
	})
})
