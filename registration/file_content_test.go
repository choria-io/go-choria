package registration

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/choria-io/go-choria/choria"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
)

func TestFileContent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "FileContent")
}

var _ = Describe("RegistrationData", func() {
	var (
		reg    *FileContent
		c      *choria.Config
		err    error
		logger *log.Entry
	)

	BeforeEach(func() {
		c, err = choria.NewConfig("/dev/null")
		Expect(err).ToNot(HaveOccurred())

		reg = &FileContent{}
		log.SetLevel(log.ErrorLevel)
		logger = log.WithFields(log.Fields{})

	})

	It("Should return nil when the data file is missing", func() {
		c.Choria.FileContentRegistrationData = "/nonexisting"
		reg.Init(c, log.WithFields(log.Fields{}))

		data, err := reg.RegistrationData()
		Expect(err).ToNot(HaveOccurred())
		Expect(data).To(BeNil())
	})

	It("Should return nil when the data file is empty", func() {
		tmpfile, err := ioutil.TempFile("", "file_content_registration")
		Expect(err).ToNot(HaveOccurred())
		tmpfile.Close()
		defer os.Remove(tmpfile.Name())

		c.Choria.FileContentRegistrationData = tmpfile.Name()
		reg.Init(c, log.WithFields(log.Fields{}))

		data, err := reg.RegistrationData()
		Expect(err).ToNot(HaveOccurred())
		Expect(data).To(BeNil())
	})

	It("Should read the file otherwise", func() {
		c.Choria.FileContentRegistrationData = "testdata/sample.json"
		reg.Init(c, log.WithFields(log.Fields{}))

		data, err := reg.RegistrationData()
		Expect(err).ToNot(HaveOccurred())
		Expect(string(*data)).To(Equal(`{"file": true}`))
	})
})
