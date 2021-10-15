// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package registration

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/tidwall/gjson"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/server/data"
)

func Test(t *testing.T) {
	os.Setenv("MCOLLECTIVE_CERTNAME", "rip.mcollective")
	RegisterFailHandler(Fail)
	RunSpecs(t, "Registration")
}

var _ = Describe("RegistrationData", func() {
	var (
		reg  *FileContent
		c    *config.Config
		err  error
		msgs chan *data.RegistrationItem
	)

	BeforeEach(func() {
		c = config.NewConfigForTests()

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
		tmpfile, err := os.CreateTemp("", "file_content_registration")
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
		Expect(string(msg.Data)).To(Equal(`{"mtime":1511865541,"file":"testdata/sample.json","updated":false,"protocol":"choria:registration:filecontent:1","zcontent":"H4sIAAAAAAAA/6pWSsvMSVWyUigpKk2tBQAAAP//AQAA//9QwpuPDgAAAA=="}`))
		Expect(msg.TargetAgent).To(Equal("registration"))
	})

	It("Should support custom targets", func() {
		c.Choria.FileContentRegistrationData = "testdata/sample.json"
		c.Choria.FileContentRegistrationTarget = "my.cmdb"

		reg.Init(c, log.WithFields(log.Fields{}))

		err = reg.publish(msgs)
		Expect(err).ToNot(HaveOccurred())

		msg := <-msgs
		Expect(string(msg.Data)).To(Equal(`{"mtime":1511865541,"file":"testdata/sample.json","updated":false,"protocol":"choria:registration:filecontent:1","zcontent":"H4sIAAAAAAAA/6pWSsvMSVWyUigpKk2tBQAAAP//AQAA//9QwpuPDgAAAA=="}`))
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
		Expect(string(msg.Data)).To(Equal(`{"mtime":1511865541,"file":"testdata/sample.json","updated":false,"protocol":"choria:registration:filecontent:1","content":"eyJmaWxlIjogdHJ1ZX0="}`))
		Expect(msg.TargetAgent).To(Equal("registration"))
	})

	It("Should detect file updates", func() {
		c.Choria.FileContentRegistrationData = "testdata/sample.json"
		c.Choria.FileContentCompression = false
		reg.Init(c, log.WithFields(log.Fields{}))

		err = reg.publish(msgs)
		Expect(err).ToNot(HaveOccurred())

		msg := <-msgs
		Expect(gjson.GetBytes(msg.Data, "updated").Bool()).To(BeFalse())

		err = os.Chtimes("testdata/sample.json", time.Now(), time.Now())
		Expect(err).ToNot(HaveOccurred())

		err = reg.publish(msgs)
		Expect(err).ToNot(HaveOccurred())

		msg = <-msgs

		Expect(gjson.GetBytes(msg.Data, "updated").Bool()).To(BeTrue())
	})
})
