package validator

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestFileContent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "McoRPC/Validator")
}

var _ = Describe("ValidateStruct", func() {
	type nest struct {
		Nested string `validate:"shellsafe"`
	}

	type vdata struct {
		SS string `validate:"shellsafe"`
		ML string `validate:"maxlength=3"`

		nest
	}

	var (
		s vdata
	)

	BeforeEach(func() {
		s = vdata{}
	})

	It("Should support nested structs", func() {
		s.SS = "safe"
		s.Nested = "safe"

		ok, err := ValidateStruct(s)
		Expect(err).ToNot(HaveOccurred())
		Expect(ok).To(BeTrue())

		s.SS = "safe"
		s.Nested = "un > safe"

		ok, err = ValidateStruct(s)
		Expect(err).To(MatchError("Nested shellsafe validation failed: may not contain '>'"))
		Expect(ok).To(BeFalse())

		s.SS = "un > safe"
		s.Nested = "safe"

		ok, err = ValidateStruct(s)
		Expect(err).To(MatchError("SS shellsafe validation failed: may not contain '>'"))
		Expect(ok).To(BeFalse())
	})

	It("Should support maxlength", func() {
		s.ML = "foo foo foo"
		ok, err := ValidateStruct(s)
		Expect(err).To(MatchError("ML maxlength validation failed: 11 characters, max allowed 3"))
		Expect(ok).To(BeFalse())
	})
})
