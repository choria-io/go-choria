package registration

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"time"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/server/data"

	"github.com/sirupsen/logrus"
)

// FileContent is a fully managed registration plugin for the choria server instance
// it reads a file and publishing it to the collective regularly
type FileContent struct {
	dataFile string
	c        *choria.Config
	log      *logrus.Entry
}

// NewFileContent creates a new fully managed registration plugin instance
func NewFileContent(c *choria.Config, logger *logrus.Entry) (*FileContent, error) {
	if c.Choria.FileContentRegistrationData == "" {
		return nil, fmt.Errorf("File Content Registration is enabled but no source data is configured, please set plugin.choria.registration.file_content.data")
	}

	reg := &FileContent{}
	reg.Init(c, logger)

	return reg, nil
}

// Init sets up the plugin
func (fc *FileContent) Init(c *choria.Config, logger *logrus.Entry) {
	fc.c = c
	fc.dataFile = c.Choria.FileContentRegistrationData
	fc.log = logger.WithFields(logrus.Fields{"registration": "file_content", "source": fc.dataFile})

	fc.log.Infof("Configured JSON Registration", fc.dataFile)
}

// Start stats a publishing loop
func (fc *FileContent) Start(ctx context.Context, wg *sync.WaitGroup, interval int, output chan *data.RegistrationItem) {
	defer wg.Done()

	err := fc.publish(output)
	if err != nil {
		fc.log.Errorf("Could not create registration data: %s", err.Error())
	}

	for {
		select {
		case <-time.Tick(time.Duration(interval) * time.Second):
			err = fc.publish(output)
			if err != nil {
				fc.log.Errorf("Could not create registration data: %s", err.Error())
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
		return fmt.Errorf("Could not find data file %s", fc.dataFile)
	}

	if fstat.Size() == 0 {
		return fmt.Errorf("Data file %s is empty", fc.dataFile)
	}

	dat, err := ioutil.ReadFile(fc.dataFile)
	if err != nil {
		return nil
	}

	item := &data.RegistrationItem{
		Data:        &dat,
		TargetAgent: "registration",
	}

	output <- item

	return nil
}
