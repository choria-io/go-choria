package server

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
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

func (self *Instance) startRegistration(ctx context.Context, wg *sync.WaitGroup) error {
	defer wg.Done()

	var err error

	switch self.config.Registration {
	case "":
		return nil
	case "file_content":
		self.registrator, err = registration.NewFileContent(self.c, self.log)
		if err != nil {
			return fmt.Errorf("Cannot start File Content Registrator: %s", err.Error())
		}
	default:
		return fmt.Errorf("Unknown registration plugin: %s", self.config.Registration)
	}

	wg.Add(1)
	go self.registrationWorker(ctx, wg)

	return nil
}

func (self *Instance) registrationWorker(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	self.log.Infof("Starting registration %s with interval %d", self.config.Registration, self.config.RegisterInterval)

	if self.config.RegistrationSplay {
		sleepTime := time.Duration(rand.Intn(self.config.RegisterInterval))
		time.Sleep(sleepTime * time.Second)
	}

	self.pollAndPublish()

	for {
		select {
		case <-time.Tick(time.Duration(self.config.RegisterInterval) * time.Second):
			self.log.Infof("Starting registration publishing process")
			self.pollAndPublish()
		case <-ctx.Done():
			self.log.Infof("Existing on shut down")
			return
		}
	}
}

func (self *Instance) pollAndPublish() {
	data, err := self.registrator.RegistrationData()
	if err != nil {
		self.log.Errorf("Could not extract registration data: %s", err.Error())
		return
	}

	if data != nil {
		msg, err := choria.NewMessage(string(*data), "discovery", self.config.RegistrationCollective, "request", nil, self.c)
		if err != nil {
			self.log.Warnf("Could not create Message for registration data: %s", err.Error())
			return
		}

		msg.SetProtocolVersion(protocol.RequestV1)
		msg.SetReplyTo("dev.null")

		self.log.Debugf("Publishing %d bytes of registration data to collective %s", len(*data), self.config.RegistrationCollective)

		err = self.connector.Publish(msg)
		if err != nil {
			self.log.Warnf("Could not publish registration Message: %s", err.Error())
			return
		}
	}
}
