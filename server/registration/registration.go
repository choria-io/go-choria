package registration

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	framework "github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/protocol"
	"github.com/choria-io/go-choria/registration"
	"github.com/sirupsen/logrus"
)

type Registrator interface {
	Init(cfg *framework.Config, l *logrus.Entry)
	RegistrationData() (*[]byte, error)
}

type RegistrationDataProvider interface {
	RegistrationData() (*[]byte, error)
}

var log *logrus.Entry
var choria *framework.Framework
var config *framework.Config
var connector framework.PublishingConnector
var registrator Registrator

func Start(ctx context.Context, wg *sync.WaitGroup, c *framework.Framework, conn framework.PublishingConnector, logger *logrus.Entry) error {
	setup(c, conn, logger)

	defer wg.Done()

	if config.RegistrationCollective == "" {
		config.RegistrationCollective = config.MainCollective
	}

	var err error

	switch config.Registration {
	case "":
		return nil
	case "file_content":
		registrator, err = registration.NewFileContent(config, log)
		if err != nil {
			return fmt.Errorf("Cannot start File Content Registrator: %s", err.Error())
		}
	default:
		return fmt.Errorf("Unknown registration plugin: %s", config.Registration)
	}

	wg.Add(1)
	go registrationWorker(ctx, wg)

	return nil
}

func setup(c *framework.Framework, conn framework.PublishingConnector, logger *logrus.Entry) {
	log = logger.WithFields(logrus.Fields{"subsystem": "registration"})
	choria = c
	config = c.Config
	connector = conn
}

func registrationWorker(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	log.Infof("Starting registration %s with interval %d", config.Registration, config.RegisterInterval)

	if config.RegistrationSplay {
		sleepTime := time.Duration(rand.Intn(config.RegisterInterval))
		time.Sleep(sleepTime * time.Second)
	}

	pollAndPublish(registrator)

	for {
		select {
		case <-time.Tick(time.Duration(config.RegisterInterval) * time.Second):
			log.Debugf("Starting registration publishing process")
			pollAndPublish(registrator)
		case <-ctx.Done():
			log.Infof("Existing on shut down")
			return
		}
	}
}

func pollAndPublish(provider RegistrationDataProvider) {
	data, err := provider.RegistrationData()
	if err != nil {
		log.Errorf("Could not extract registration data: %s", err.Error())
		return
	}

	if data == nil {
		log.Warnf("Received nil data from Registratoin Plugin, skipping")
		return
	}

	if len(*data) == 0 {
		log.Warnf("Received empty data from Registratoin Plugin, skipping")
		return
	}

	msg, err := framework.NewMessage(string(*data), "discovery", config.RegistrationCollective, "request", nil, choria)
	if err != nil {
		log.Warnf("Could not create Message for registration data: %s", err.Error())
		return
	}

	msg.SetProtocolVersion(protocol.RequestV1)
	msg.SetReplyTo("dev.null")

	log.Debugf("Publishing %d bytes of registration data to collective %s", len(*data), config.RegistrationCollective)

	err = connector.Publish(msg)
	if err != nil {
		log.Warnf("Could not publish registration Message: %s", err.Error())
		return
	}
}
