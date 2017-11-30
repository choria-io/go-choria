package facts

import (
	"testing"

	"github.com/choria-io/go-choria/choria"
	"github.com/sirupsen/logrus"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestFileContent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Server/Discovery/Facts")
}

var _ = Describe("Server/Discovery/Facts", func() {
	var (
		t   func(fact, op, val string) (bool, error)
		fw  *choria.Framework
		err error
		l   *logrus.Entry
	)

	BeforeSuite(func() {
		t = func(fact, op, val string) (bool, error) {
			return HasFact(fact, op, val, "testdata/fact.yaml")
		}

		l = logrus.WithFields(logrus.Fields{"test": true})
		fw, err = choria.New("/dev/null")
		Expect(err).NotTo(HaveOccurred())

		fw.Config.Choria.FactSourceFile = "testdata/fact.yaml"
	})

	var _ = Describe("Match", func() {
		It("Be true if all match", func() {
			filters := [][3]string{}
			filters = append(filters, [3]string{"string", "==", "hello world"})
			filters = append(filters, [3]string{"nested.string", "==", "nested hello world"})

			Expect(Match(filters, fw, l)).To(BeTrue())
		})

		It("Be false if some do not match", func() {
			filters := [][3]string{}
			filters = append(filters, [3]string{"string", "==", "hello world"})
			filters = append(filters, [3]string{"nested.string", "==", "fail"})

			Expect(Match(filters, fw, l)).To(BeFalse())
		})
	})

	var _ = Describe("HasFact", func() {
		It("Should fail on missing data", func() {
			_, err := HasFact("foo", "==", "bar", "testdata/missing.yaml")
			Expect(err).To(MatchError("Cannot do fact discovery the file 'testdata/missing.yaml' does not exist"))
		})

		It("Should match strings", func() {
			Expect(t("string", "==", "hello world")).To(BeTrue())
			Expect(t("string", "!=", "hello world")).To(BeFalse())
			Expect(t("string", "==", "helloworld")).To(BeFalse())
			Expect(t("string", "=~", "/hello/")).To(BeTrue())
			Expect(t("string", "=~", "/bye/")).To(BeFalse())
			Expect(t("string", "<", "zz")).To(BeTrue())
			Expect(t("string", ">", "zz")).To(BeFalse())
			Expect(t("string", "!=", "zz")).To(BeTrue())
			Expect(t("nested.string", "==", "nested hello world")).To(BeTrue())
		})

		It("Should match ints", func() {
			Expect(t("nested.inumber", "==", "2")).To(BeTrue())
			Expect(t("inumber", "==", "2")).To(BeFalse())
			Expect(t("inumber", "!=", "1")).To(BeFalse())
			Expect(t("inumber", ">=", "1")).To(BeTrue())
			Expect(t("inumber", ">", "1")).To(BeFalse())
			Expect(t("inumber", "<", "10")).To(BeTrue())
		})

		It("Should match floats", func() {
			Expect(t("nested.fnumber", "==", "2.2")).To(BeTrue())
			Expect(t("fnumber", "==", "1.2")).To(BeTrue())
			Expect(t("fnumber", "=~", "/1\\.\\d+/")).To(BeTrue())
			Expect(t("fnumber", "!=", "1.2")).To(BeFalse())
			Expect(t("fnumber", "<=", "1.01")).To(BeFalse())
			Expect(t("fnumber", ">=", "1")).To(BeTrue())
			Expect(t("fnumber", "<", "10")).To(BeTrue())
			Expect(t("fnumber", "<", "1.0")).To(BeFalse())
			Expect(t("fnumber", ">", "10")).To(BeFalse())
			Expect(t("fnumber", ">", "1.0")).To(BeTrue())
		})

		It("Should match true booleans", func() {
			Expect(t("tbool", "==", "true")).To(BeTrue())
			Expect(t("tbool", "==", "T")).To(BeTrue())
			Expect(t("tbool", "==", "1")).To(BeTrue())
			Expect(t("tbool", "==", "t")).To(BeTrue())
			Expect(t("tbool", "==", "false")).To(BeFalse())
			Expect(t("tbool", "!=", "bob")).To(BeTrue())
			Expect(t("tbool", "!=", "false")).To(BeTrue())
			Expect(t("tbool", "!=", "T")).To(BeFalse())
		})

		It("Should match false booleans", func() {
			Expect(t("fbool", "==", "false")).To(BeTrue())
			Expect(t("fbool", "==", "F")).To(BeTrue())
			Expect(t("fbool", "==", "0")).To(BeTrue())
			Expect(t("fbool", "==", "f")).To(BeTrue())
			Expect(t("fbool", "==", "true")).To(BeFalse())
			Expect(t("fbool", "!=", "bob")).To(BeTrue())
			Expect(t("fbool", "!=", "true")).To(BeFalse())
			Expect(t("fbool", "!=", "false")).To(BeTrue())
		})
	})
})
