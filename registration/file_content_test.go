package registration

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/server/data"
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
		msgs   chan *data.RegistrationItem
	)

	BeforeEach(func() {
		c, err = choria.NewConfig("/dev/null")
		Expect(err).ToNot(HaveOccurred())

		reg = &FileContent{}
		log.SetLevel(log.ErrorLevel)
		logger = log.WithFields(log.Fields{})

		msgs = make(chan *data.RegistrationItem, 1)
	})

	It("Should return err when the data file is missing", func() {
		c.Choria.FileContentRegistrationData = "/nonexisting"
		reg.Init(c, log.WithFields(log.Fields{}))

		err := reg.publish(msgs)
		Expect(err).To(MatchError("Could not find data file /nonexisting"))
	})

	It("Should return err when the data file is empty", func() {
		tmpfile, err := ioutil.TempFile("", "file_content_registration")
		Expect(err).ToNot(HaveOccurred())
		tmpfile.Close()
		defer os.Remove(tmpfile.Name())

		c.Choria.FileContentRegistrationData = tmpfile.Name()
		reg.Init(c, log.WithFields(log.Fields{}))

		err = reg.publish(msgs)
		Expect(err).To(MatchError(fmt.Sprintf("Data file %s is empty", tmpfile.Name())))
	})

	It("Should read the file and publish it to default location", func() {
		c.Choria.FileContentRegistrationData = "testdata/sample.json"
		reg.Init(c, log.WithFields(log.Fields{}))

		err = reg.publish(msgs)
		Expect(err).ToNot(HaveOccurred())

		msg := <-msgs
		Expect(string(*msg.Data)).To(Equal(`{"file": true}`))
		Expect(msg.TargetAgent).To(Equal("registration"))
	})

	It("Should support custom targets", func() {
		c.Choria.FileContentRegistrationData = "testdata/sample.json"
		c.Choria.FileContentRegistrationTarget = "my.cmdb"

		reg.Init(c, log.WithFields(log.Fields{}))

		err = reg.publish(msgs)
		Expect(err).ToNot(HaveOccurred())

		msg := <-msgs
		Expect(string(*msg.Data)).To(Equal(`{"file": true}`))
		Expect(msg.TargetAgent).To(Equal(""))
		Expect(msg.Destination).To(Equal("my.cmdb"))
	})
})
