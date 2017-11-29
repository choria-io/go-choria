package facts

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestFileContent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Facts")
}

var _ = Describe("HasFact", func() {
	BeforeEach(func() {
		Setup("testdata/fact.yaml", logrus.WithFields(logrus.Fields{}))
	})

	It("Should match strings", func() {
		Expect(HasFact("string", "==", "hello world")).To(BeTrue())
		Expect(HasFact("string", "!=", "hello world")).To(BeFalse())
		Expect(HasFact("string", "==", "helloworld")).To(BeFalse())
		Expect(HasFact("string", "=~", "/hello/")).To(BeTrue())
		Expect(HasFact("string", "=~", "/bye/")).To(BeFalse())
		Expect(HasFact("string", "<", "zz")).To(BeTrue())
		Expect(HasFact("string", ">", "zz")).To(BeFalse())
		Expect(HasFact("string", "!=", "zz")).To(BeTrue())
		Expect(HasFact("nested.string", "==", "nested hello world")).To(BeTrue())
	})

	It("Should match ints", func() {
		Expect(HasFact("nested.inumber", "==", "2")).To(BeTrue())
		Expect(HasFact("inumber", "==", "2")).To(BeFalse())
		Expect(HasFact("inumber", "!=", "1")).To(BeFalse())
		Expect(HasFact("inumber", ">=", "1")).To(BeTrue())
		Expect(HasFact("inumber", ">", "1")).To(BeFalse())
		Expect(HasFact("inumber", "<", "10")).To(BeTrue())
	})

	It("Should match floats", func() {
		Expect(HasFact("nested.fnumber", "==", "2.2")).To(BeTrue())
		Expect(HasFact("fnumber", "==", "1.2")).To(BeTrue())
		Expect(HasFact("fnumber", "=~", "/1\\.\\d+/")).To(BeTrue())
		Expect(HasFact("fnumber", "!=", "1.2")).To(BeFalse())
		Expect(HasFact("fnumber", "<=", "1.01")).To(BeFalse())
		Expect(HasFact("fnumber", ">=", "1")).To(BeTrue())
		Expect(HasFact("fnumber", "<", "10")).To(BeTrue())
		Expect(HasFact("fnumber", "<", "1.0")).To(BeFalse())
		Expect(HasFact("fnumber", ">", "10")).To(BeFalse())
		Expect(HasFact("fnumber", ">", "1.0")).To(BeTrue())
	})

	It("Should match true booleans", func() {
		Expect(HasFact("tbool", "==", "true")).To(BeTrue())
		Expect(HasFact("tbool", "==", "T")).To(BeTrue())
		Expect(HasFact("tbool", "==", "1")).To(BeTrue())
		Expect(HasFact("tbool", "==", "t")).To(BeTrue())
		Expect(HasFact("tbool", "==", "false")).To(BeFalse())
		Expect(HasFact("tbool", "!=", "bob")).To(BeTrue())
		Expect(HasFact("tbool", "!=", "false")).To(BeTrue())
		Expect(HasFact("tbool", "!=", "T")).To(BeFalse())
	})

	It("Should match false booleans", func() {
		Expect(HasFact("fbool", "==", "false")).To(BeTrue())
		Expect(HasFact("fbool", "==", "F")).To(BeTrue())
		Expect(HasFact("fbool", "==", "0")).To(BeTrue())
		Expect(HasFact("fbool", "==", "f")).To(BeTrue())
		Expect(HasFact("fbool", "==", "true")).To(BeFalse())
		Expect(HasFact("fbool", "!=", "bob")).To(BeTrue())
		Expect(HasFact("fbool", "!=", "true")).To(BeFalse())
		Expect(HasFact("fbool", "!=", "false")).To(BeTrue())
	})
})
