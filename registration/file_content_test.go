package registration

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/server/data"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
)

func TestFileContent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Registration")
}

var _ = Describe("RegistrationData", func() {
	var (
		reg  *FileContent
		c    *choria.Config
		err  error
		msgs chan *data.RegistrationItem
	)

	BeforeEach(func() {
		c, err = choria.NewDefaultConfig()
		Expect(err).ToNot(HaveOccurred())

		reg = &FileContent{}
		log.SetLevel(log.ErrorLevel)

		msgs = make(chan *data.RegistrationItem, 1)

		os.Chtimes("testdata/sample.json", time.Unix(1511865541, 0), time.Unix(1511865541, 0))
	})

	It("Should return err when the data file is missing", func() {
		c.Choria.FileContentRegistrationData = "/nonexisting"
		reg.Init(c, log.WithFields(log.Fields{}))

		err := reg.publish(msgs)
		Expect(err).To(MatchError("could not find data file /nonexisting"))
	})

	It("Should return err when the data file is empty", func() {
		tmpfile, err := ioutil.TempFile("", "file_content_registration")
		Expect(err).ToNot(HaveOccurred())
		tmpfile.Close()
		defer os.Remove(tmpfile.Name())

		c.Choria.FileContentRegistrationData = tmpfile.Name()
		reg.Init(c, log.WithFields(log.Fields{}))

		err = reg.publish(msgs)
		Expect(err).To(MatchError(fmt.Sprintf("data file %s is empty", tmpfile.Name())))
	})

	It("Should read the file and publish it to default location", func() {
		c.Choria.FileContentRegistrationData = "testdata/sample.json"
		reg.Init(c, log.WithFields(log.Fields{}))

		err = reg.publish(msgs)
		Expect(err).ToNot(HaveOccurred())

		msg := <-msgs
		Expect(string(*msg.Data)).To(Equal(`{"mtime":1511865541,"file":"testdata/sample.json","protocol":"choria:registration:filecontent:1","zcontent":"H4sIAAAAAAAA/6pWSsvMSVWyUigpKk2tBQAAAP//AQAA//9QwpuPDgAAAA=="}`))
		Expect(msg.TargetAgent).To(Equal("registration"))
	})

	It("Should support custom targets", func() {
		c.Choria.FileContentRegistrationData = "testdata/sample.json"
		c.Choria.FileContentRegistrationTarget = "my.cmdb"

		reg.Init(c, log.WithFields(log.Fields{}))

		err = reg.publish(msgs)
		Expect(err).ToNot(HaveOccurred())

		msg := <-msgs
		Expect(string(*msg.Data)).To(Equal(`{"mtime":1511865541,"file":"testdata/sample.json","protocol":"choria:registration:filecontent:1","zcontent":"H4sIAAAAAAAA/6pWSsvMSVWyUigpKk2tBQAAAP//AQAA//9QwpuPDgAAAA=="}`))
		Expect(msg.TargetAgent).To(Equal(""))
		Expect(msg.Destination).To(Equal("my.cmdb"))
	})

	It("Should support disabling compression", func() {
		c.Choria.FileContentRegistrationData = "testdata/sample.json"
		c.Choria.FileContentCompression = false
		reg.Init(c, log.WithFields(log.Fields{}))

		err = reg.publish(msgs)
		Expect(err).ToNot(HaveOccurred())

		msg := <-msgs
		Expect(string(*msg.Data)).To(Equal(`{"mtime":1511865541,"file":"testdata/sample.json","protocol":"choria:registration:filecontent:1","content":"eyJmaWxlIjogdHJ1ZX0="}`))
		Expect(msg.TargetAgent).To(Equal("registration"))
	})
})
