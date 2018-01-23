package ipv6

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
		Expect(err).To(MatchError("1.2.3.4 is not an IPv6 address"))
		Expect(ok).To(BeFalse())

		ok, err = ValidateString("foo")
		Expect(err).To(MatchError("foo is not an IPv6 address"))
		Expect(ok).To(BeFalse())

		ok, err = ValidateString("2a00:1450:4003:807::200e")
		Expect(err).ToNot(HaveOccurred())
		Expect(ok).To(BeTrue())
	})
})

var _ = Describe("ValidateStructField", func() {
	type t struct {
		IP string `validate:"ipv6"`
	}

	It("Should validate the struct correctly", func() {
		st := t{"1.2.3.4"}

		val := reflect.ValueOf(st)
		valueField := val.FieldByName("IP")
		typeField, _ := val.Type().FieldByName("IP")

		ok, err := ValidateStructField(valueField, typeField.Tag.Get("validate"))
		Expect(err).To(MatchError("1.2.3.4 is not an IPv6 address"))
		Expect(ok).To(BeFalse())

		st.IP = "foo"
		valueField = reflect.ValueOf(st).FieldByName("IP")
		ok, err = ValidateStructField(valueField, typeField.Tag.Get("validate"))
		Expect(err).To(MatchError("foo is not an IPv6 address"))
		Expect(ok).To(BeFalse())

		st.IP = "2a00:1450:4003:807::200e"
		valueField = reflect.ValueOf(st).FieldByName("IP")
		ok, err = ValidateStructField(valueField, typeField.Tag.Get("validate"))
		Expect(err).ToNot(HaveOccurred())
		Expect(ok).To(BeTrue())
	})
})
