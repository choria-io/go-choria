package classes

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestFileContent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Classes")
}

var _ = Describe("Classes", func() {
	var log *logrus.Entry

	BeforeEach(func() {
		log = logrus.WithFields(logrus.Fields{"testing": true})
		logrus.SetLevel(logrus.PanicLevel)
	})

	It("Should handle missing classes files", func() {
		Expect(Match([]string{"x"}, "testdata/nonexisting.txt", log)).To(BeFalse())
	})

	It("Should support regex", func() {
		Expect(Match([]string{"/test/"}, "testdata/classes.txt", log)).To(BeTrue())
		Expect(Match([]string{"/nonxisting/"}, "testdata/classes.txt", log)).To(BeFalse())
	})

	It("Should support exact matches", func() {
		Expect(Match([]string{"role::testing"}, "testdata/classes.txt", log)).To(BeTrue())
		Expect(Match([]string{"nonxisting"}, "testdata/classes.txt", log)).To(BeFalse())
	})
})
