package ipv4

import (
	"reflect"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestFileContent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Validator/IPv4")
}

var _ = Describe("ValidateString", func() {
	It("Should match ipv4 addresses correctly", func() {
		ok, err := ValidateString("1.2.3.4")
		Expect(err).ToNot(HaveOccurred())
		Expect(ok).To(BeTrue())

		ok, err = ValidateString("foo")
		Expect(err).To(MatchError("foo is not an IPv4 address"))
		Expect(ok).To(BeFalse())

		ok, err = ValidateString("2a00:1450:4003:807::200e")
		Expect(err).To(MatchError("2a00:1450:4003:807::200e is not an IPv4 address"))
		Expect(ok).To(BeFalse())
	})
})

var _ = Describe("ValidateStructField", func() {
	type t struct {
		IP string `validate:"ipv4"`
	}

	It("Should validate the struct correctly", func() {
		st := t{"1.2.3.4"}

		val := reflect.ValueOf(st)
		valueField := val.FieldByName("IP")
		typeField, _ := val.Type().FieldByName("IP")

		ok, err := ValidateStructField(valueField, typeField.Tag.Get("validate"))
		Expect(err).ToNot(HaveOccurred())
		Expect(ok).To(BeTrue())

		st.IP = "foo"
		valueField = reflect.ValueOf(st).FieldByName("IP")
		ok, err = ValidateStructField(valueField, typeField.Tag.Get("validate"))
		Expect(err).To(MatchError("foo is not an IPv4 address"))
		Expect(ok).To(BeFalse())

		st.IP = "2a00:1450:4003:807::200e"
		valueField = reflect.ValueOf(st).FieldByName("IP")
		ok, err = ValidateStructField(valueField, typeField.Tag.Get("validate"))
		Expect(err).To(MatchError("2a00:1450:4003:807::200e is not an IPv4 address"))
		Expect(ok).To(BeFalse())
	})
})
