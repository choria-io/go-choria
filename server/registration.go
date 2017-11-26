package server

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/protocol"
	"github.com/choria-io/go-choria/registration"
	log "github.com/sirupsen/logrus"
)

type Registrator interface {
	Init(c *choria.Config, l *log.Entry)
	RegistrationData() (*[]byte, error)
}

func (self *Instance) startRegistration() error {
	var err error

	switch self.config.Registration {
	case "":
		return nil
	case "file_content":
		self.registrator, err = registration.NewFileContent(self.c, self.logger)
		if err != nil {
			return fmt.Errorf("Cannot start File Content Registrator: %s", err.Error())
		}
	default:
		return fmt.Errorf("Unknown registration plugin: %s", self.config.Registration)
	}

	go self.registrationWorker()

	return nil
}

func (self *Instance) registrationWorker() {
	self.logger.Infof("Starting registration %s with interval %d", self.config.Registration, self.config.RegisterInterval)

	if self.config.RegistrationSplay {
		sleepTime := time.Duration(rand.Intn(self.config.RegisterInterval))
		time.Sleep(sleepTime * time.Second)
	}

	cnt := 0

	for {
		if cnt > 0 {
			time.Sleep(time.Duration(self.config.RegisterInterval) * time.Second)
		}

		cnt++

		data, err := self.registrator.RegistrationData()
		if err != nil {
			self.logger.Errorf("Could not extract registration data: %s", err.Error())
			continue
		}

		if data != nil {
			msg, err := choria.NewMessage(string(*data), "discovery", self.config.RegistrationCollective, "request", nil, self.c)
			if err != nil {
				self.logger.Warnf("Could not create Message for registration data: %s", err.Error())
				continue
			}

			msg.SetProtocolVersion(protocol.RequestV1)
			msg.SetReplyTo("dev.null")

			self.logger.Debugf("Publishing %d bytes of registration data to collective %s", len(*data), self.config.RegistrationCollective)

			err = self.connector.Publish(msg)
			if err != nil {
				self.logger.Warnf("Could not publish registration Message: %s", err.Error())
				continue
			}
		}
	}
}
