package validator_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"

	"github.com/choria-io/go-validator"
)

func TestFileContent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Validator")
}

type nest struct {
	Nested string `validate:"shellsafe"`
}

type vdata struct {
	SS   string   `validate:"shellsafe"`
	ML   string   `validate:"maxlength=3"`
	Enum []string `validate:"enum=one,two"`
	IPv4 string   `validate:"ipv4"`
	IPv6 string   `validate:"ipv6"`
	IP   string   `validate:"ipaddress"`

	nest
}

var s vdata

var _ = Describe("ValidateStructField", func() {
	BeforeEach(func() {
		s = vdata{
			IPv4: "1.2.3.4",
			IPv6: "2a00:1450:4003:807::200e",
			IP:   "1.2.3.4",
		}
	})

	It("Should validate a specific field", func() {
		s.ML = "too long"
		ok, err := validator.ValidateStructField(s, "ML")
		Expect(err).To(MatchError("ML maxlength validation failed: 8 characters, max allowed 3"))
		Expect(ok).To(BeFalse())
	})

	It("Should handle unknown fields", func() {
		ok, err := validator.ValidateStructField(s, "foo")
		Expect(err).To(MatchError("unknown field foo"))
		Expect(ok).To(BeFalse())
	})
})

var _ = Describe("ValidateStruct", func() {
	BeforeEach(func() {
		s = vdata{
			IPv4: "1.2.3.4",
			IPv6: "2a00:1450:4003:807::200e",
			IP:   "1.2.3.4",
		}
	})

	It("Should support nested structs", func() {
		s.SS = "safe"
		s.Nested = "safe"

		ok, err := validator.ValidateStruct(s)
		Expect(err).ToNot(HaveOccurred())
		Expect(ok).To(BeTrue())

		s.SS = "safe"
		s.Nested = "un > safe"

		ok, err = validator.ValidateStruct(s)
		Expect(err).To(MatchError("Nested shellsafe validation failed: may not contain '>'"))
		Expect(ok).To(BeFalse())

		s.SS = "un > safe"
		s.Nested = "safe"

		ok, err = validator.ValidateStruct(s)
		Expect(err).To(MatchError("SS shellsafe validation failed: may not contain '>'"))
		Expect(ok).To(BeFalse())
	})

	It("Should support maxlength", func() {
		s.ML = "foo foo foo"
		ok, err := validator.ValidateStruct(s)
		Expect(err).To(MatchError("ML maxlength validation failed: 11 characters, max allowed 3"))
		Expect(ok).To(BeFalse())
	})

	It("Should support enum", func() {
		s.Enum = []string{"four"}
		ok, err := validator.ValidateStruct(s)
		Expect(err).To(MatchError("Enum enum validation failed: 'four' is not in the allowed list: one, two"))
		Expect(ok).To(BeFalse())
	})

	It("Should support ipv4", func() {
		s.IPv4 = "2a00:1450:4003:807::200e"
		ok, err := validator.ValidateStruct(s)
		Expect(err).To(MatchError("IPv4 IPv4 validation failed: 2a00:1450:4003:807::200e is not an IPv4 address"))
		Expect(ok).To(BeFalse())
	})

	It("Should support ipv6", func() {
		s.IPv6 = "1.2.3.4"
		ok, err := validator.ValidateStruct(s)
		Expect(err).To(MatchError("IPv6 IPv6 validation failed: 1.2.3.4 is not an IPv6 address"))
		Expect(ok).To(BeFalse())
	})

	It("Should support ipaddress", func() {
		s.IP = "foo"
		ok, err := validator.ValidateStruct(s)
		Expect(err).To(MatchError("IP IP address validation failed: foo is not an IP address"))
		Expect(ok).To(BeFalse())
	})

})
