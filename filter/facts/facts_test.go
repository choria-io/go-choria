package facts

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func Test(t *testing.T) {
	os.Setenv("MCOLLECTIVE_CERTNAME", "rip.mcollective")
	RegisterFailHandler(Fail)
	RunSpecs(t, "Server/Discovery/Facts")
}

var _ = Describe("Server/Discovery/Facts", func() {
	var (
		t func(fact, op, val string) (bool, error)
		l *logrus.Entry
	)

	BeforeSuite(func() {
		l = logrus.WithFields(logrus.Fields{"test": true})
		l.Logger.Out = ioutil.Discard

		t = func(fact, op, val string) (bool, error) {
			return HasFact(fact, op, val, "testdata/fact.yaml", l)
		}

		// fw.Config.FactSourceFile = "testdata/fact.yaml"
	})

	Describe("JSON", func() {
		It("Should merge multiple fact files", func() {
			j, err := JSON(strings.Join([]string{"testdata/fact.yaml", "testdata/2ndfact.json"}, string(os.PathListSeparator)), l)
			Expect(err).ToNot(HaveOccurred())

			Expect(gjson.GetBytes(j, "ifact").Int()).To(Equal(int64(2)))
		})
	})

	Describe("Match", func() {
		It("Be true if all match", func() {
			filters := [][3]string{}
			filters = append(filters, [3]string{"string", "==", "hello world"})
			filters = append(filters, [3]string{"nested.string", "==", "nested hello world"})

			Expect(MatchFile(filters, "testdata/fact.yaml", l)).To(BeTrue())
		})

		It("Be false if some do not match", func() {
			filters := [][3]string{}
			filters = append(filters, [3]string{"string", "==", "hello world"})
			filters = append(filters, [3]string{"nested.string", "==", "fail"})

			Expect(MatchFile(filters, "testdata/fact.yaml", l)).To(BeFalse())
		})
	})

	Describe("HasFact", func() {
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
