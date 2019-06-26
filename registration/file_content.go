package registration

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"time"

	"github.com/choria-io/go-choria/server/data"
	"github.com/choria-io/go-config"

	"github.com/sirupsen/logrus"
)

// FileContent is a fully managed registration plugin for the choria server instance
// it reads a file and publishing it to the collective regularly
type FileContent struct {
	dataFile string
	c        *config.Config
	log      *logrus.Entry

	prevMtime int64
}

// FileContentMessage contains message being published
type FileContentMessage struct {
	Mtime    int64   `json:"mtime"`
	File     string  `json:"file"`
	Updated  bool    `json:"updated"`
	Protocol string  `json:"protocol"`
	Content  *[]byte `json:"content,omitempty"`
	ZContent *[]byte `json:"zcontent,omitempty"`
}

// NewFileContent creates a new fully managed registration plugin instance
func NewFileContent(c *config.Config, logger *logrus.Entry) (*FileContent, error) {
	if c.Choria.FileContentRegistrationData == "" {
		return nil, fmt.Errorf("File Content Registration is enabled but no source data is configured, please set plugin.choria.registration.file_content.data")
	}

	reg := &FileContent{}
	reg.Init(c, logger)

	return reg, nil
}

// Init sets up the plugin
func (fc *FileContent) Init(c *config.Config, logger *logrus.Entry) {
	fc.c = c
	fc.dataFile = c.Choria.FileContentRegistrationData
	fc.log = logger.WithFields(logrus.Fields{"registration": "file_content", "source": fc.dataFile})

	fc.log.Infof("Configured File Content Registration with source '%s' and target '%s'", fc.dataFile, c.Choria.FileContentRegistrationTarget)
}

// StartRegistration starts stats a publishing loop
func (fc *FileContent) StartRegistration(ctx context.Context, wg *sync.WaitGroup, interval int, output chan *data.RegistrationItem) {
	defer wg.Done()

	err := fc.publish(output)
	if err != nil {
		fc.log.Errorf("Could not create registration data: %s", err)
	}

	for {
		select {
		case <-time.Tick(time.Duration(interval) * time.Second):
			err = fc.publish(output)
			if err != nil {
				fc.log.Errorf("Could not create registration data: %s", err)
			}

		case <-ctx.Done():
			return
		}
	}
}

func (fc *FileContent) publish(output chan *data.RegistrationItem) error {
	fc.log.Infof("Starting file_content registration poll")

	fstat, err := os.Stat(fc.dataFile)
	if os.IsNotExist(err) {
		return fmt.Errorf("could not find data file %s", fc.dataFile)
	}

	if fstat.Size() == 0 {
		return fmt.Errorf("data file %s is empty", fc.dataFile)
	}

	fstat, err = os.Stat(fc.dataFile)
	if err != nil {
		return fmt.Errorf("could not obtain file times: %s", err)
	}

	dat, err := ioutil.ReadFile(fc.dataFile)
	if err != nil {
		return fmt.Errorf("could not read file registration source %s: %s", fc.dataFile, err)
	}

	msg := &FileContentMessage{
		Protocol: "choria:registration:filecontent:1",
		File:     fc.dataFile,
		Mtime:    fstat.ModTime().Unix(),
	}

	// the first time it starts we just have no idea, so we set it to whatever
	// it is now which would also avoid setting updated=true, we do not want a
	// large fleet restart to mass trigger a needless full site replication
	if fc.prevMtime == 0 {
		fc.prevMtime = msg.Mtime
	}

	if msg.Mtime > fc.prevMtime {
		msg.Updated = true
		fc.prevMtime = msg.Mtime
	}

	if fc.c.Choria.FileContentCompression {
		zdat, err := fc.compress(dat)
		if err != nil {
			fc.log.Warnf("Could not compress file registration data: %s", err)
		} else {
			msg.ZContent = &zdat
		}
	}

	if msg.ZContent == nil {
		msg.Content = &dat
	}

	jdat, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("could not json marshal registration message: %s", err)
	}

	item := &data.RegistrationItem{
		Data:        &jdat,
		Destination: fc.c.Choria.FileContentRegistrationTarget,
	}

	if item.Destination == "" {
		item.TargetAgent = "registration"
	}

	output <- item

	return nil
}

func (fc *FileContent) compress(data []byte) ([]byte, error) {
	var b bytes.Buffer

	gz := gzip.NewWriter(&b)

	_, err := gz.Write(data)
	if err != nil {
		return []byte{}, err
	}

	err = gz.Flush()
	if err != nil {
		return []byte{}, err
	}

	err = gz.Close()
	if err != nil {
		return []byte{}, err
	}

	return b.Bytes(), nil
}
