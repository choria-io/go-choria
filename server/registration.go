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

type RegistrationDataProvider interface {
	RegistrationData() (*[]byte, error)
}

func (self *Instance) startRegistration(ctx context.Context, wg *sync.WaitGroup) error {
	defer wg.Done()

	if self.config.RegistrationCollective == "" {
		self.config.RegistrationCollective = self.config.MainCollective
	}

	var err error

	switch self.config.Registration {
	case "":
		return nil
	case "file_content":
		self.registrator, err = registration.NewFileContent(self.c.Config, self.log)
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

	self.pollAndPublish(self.registrator, self.connector)

	for {
		select {
		case <-time.Tick(time.Duration(self.config.RegisterInterval) * time.Second):
			self.log.Debugf("Starting registration publishing process")
			self.pollAndPublish(self.registrator, self.connector)
		case <-ctx.Done():
			self.log.Infof("Existing on shut down")
			return
		}
	}
}

func (self *Instance) pollAndPublish(provider RegistrationDataProvider, connection choria.PublishingConnector) {
	data, err := provider.RegistrationData()
	if err != nil {
		self.log.Errorf("Could not extract registration data: %s", err.Error())
		return
	}

	if data == nil {
		self.log.Warnf("Received nil data from Registratoin Plugin, skipping")
		return
	}

	if len(*data) == 0 {
		self.log.Warnf("Received empty data from Registratoin Plugin, skipping")
		return
	}

	msg, err := choria.NewMessage(string(*data), "discovery", self.config.RegistrationCollective, "request", nil, self.c)
	if err != nil {
		self.log.Warnf("Could not create Message for registration data: %s", err.Error())
		return
	}

	msg.SetProtocolVersion(protocol.RequestV1)
	msg.SetReplyTo("dev.null")

	self.log.Infof(self.config.MainCollective)
	self.log.Debugf("Publishing %d bytes of registration data to collective %s", len(*data), self.config.RegistrationCollective)

	err = connection.Publish(msg)
	if err != nil {
		self.log.Warnf("Could not publish registration Message: %s", err.Error())
		return
	}
}
