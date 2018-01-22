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
		nest
	}

	var (
		s = vdata{}
	)

	It("Should support nested structs", func() {
		s.SS = "safe"
		s.Nested = "safe"

		ok, err := ValidateStruct(s)
		Expect(err).ToNot(HaveOccurred())
		Expect(ok).To(BeTrue())

		s.SS = "safe"
		s.Nested = "un > safe"

		ok, err = ValidateStruct(s)
		Expect(err).To(MatchError("Nested is not shellsafe"))
		Expect(ok).To(BeFalse())

		s.SS = "un > safe"
		s.Nested = "safe"

		ok, err = ValidateStruct(s)
		Expect(err).To(MatchError("SS is not shellsafe"))
		Expect(ok).To(BeFalse())
	})
})
