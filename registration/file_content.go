package registration

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/choria-io/go-choria/choria"
	log "github.com/sirupsen/logrus"
)

type FileContent struct {
	dataFile string
	interval int
	c        *choria.Config
	l        *log.Entry
}

func NewFileContent(c *choria.Framework, logger *log.Entry) (*FileContent, error) {
	if c.Config.Choria.FileContentRegistrationData == "" {
		return nil, fmt.Errorf("File Content Registration is enabled but no source data is configured, please set plugin.choria.registration.file_content.data")
	}

	reg := &FileContent{}
	reg.Init(c.Config, logger)

	return reg, nil
}

func (self *FileContent) Init(c *choria.Config, logger *log.Entry) {
	self.c = c
	self.interval = c.RegisterInterval
	self.dataFile = c.Choria.FileContentRegistrationData
	self.l = logger.WithFields(log.Fields{"registration": "file_content"})

	self.l.Infof("Configured JSON Registration with source file %s", self.dataFile)
}

func (self *FileContent) RegistrationData() (*[]byte, error) {
	if _, err := os.Stat(self.dataFile); os.IsNotExist(err) {
		self.l.Infof("Could not find data file %s for registration, skipping", self.dataFile)
		return nil, nil
	}

	dat, err := ioutil.ReadFile(self.dataFile)
	if err != nil {
		return nil, err
	}

	return &dat, nil
}
