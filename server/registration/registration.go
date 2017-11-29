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

type Manager struct {
	log         *logrus.Entry
	choria      *framework.Framework
	cfg         *framework.Config
	connector   framework.PublishingConnector
	registrator Registrator
}

func New(c *framework.Framework, conn framework.PublishingConnector, logger *logrus.Entry) *Manager {
	r := &Manager{
		log:       logger.WithFields(logrus.Fields{"subsystem": "registration"}),
		choria:    c,
		cfg:       c.Config,
		connector: conn,
	}

	return r
}

func (reg *Manager) Start(ctx context.Context, wg *sync.WaitGroup) error {
	defer wg.Done()

	if reg.cfg.RegistrationCollective == "" {
		reg.cfg.RegistrationCollective = reg.cfg.MainCollective
	}

	var err error

	switch reg.cfg.Registration {
	case "":
		return nil
	case "file_content":
		reg.registrator, err = registration.NewFileContent(reg.cfg, reg.log)
		if err != nil {
			return fmt.Errorf("Cannot start File Content Registrator: %s", err.Error())
		}
	default:
		return fmt.Errorf("Unknown registration plugin: %s", reg.cfg.Registration)
	}

	wg.Add(1)
	go reg.registrationWorker(ctx, wg)

	return nil
}

func (reg *Manager) registrationWorker(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	reg.log.Infof("Starting registration %s with interval %d", reg.cfg.Registration, reg.cfg.RegisterInterval)

	if reg.cfg.RegistrationSplay {
		sleepTime := time.Duration(rand.Intn(reg.cfg.RegisterInterval))
		time.Sleep(sleepTime * time.Second)
	}

	reg.pollAndPublish(reg.registrator)

	for {
		select {
		case <-time.Tick(time.Duration(reg.cfg.RegisterInterval) * time.Second):
			reg.log.Debugf("Starting registration publishing process")
			reg.pollAndPublish(reg.registrator)
		case <-ctx.Done():
			reg.log.Infof("Existing on shut down")
			return
		}
	}
}

func (reg *Manager) pollAndPublish(provider RegistrationDataProvider) {
	data, err := provider.RegistrationData()
	if err != nil {
		reg.log.Errorf("Could not extract registration data: %s", err.Error())
		return
	}

	if data == nil {
		reg.log.Warnf("Received nil data from Registratoin Plugin, skipping")
		return
	}

	if len(*data) == 0 {
		reg.log.Warnf("Received empty data from Registratoin Plugin, skipping")
		return
	}

	msg, err := framework.NewMessage(string(*data), "discovery", reg.cfg.RegistrationCollective, "request", nil, reg.choria)
	if err != nil {
		reg.log.Warnf("Could not create Message for registration data: %s", err.Error())
		return
	}

	msg.SetProtocolVersion(protocol.RequestV1)
	msg.SetReplyTo("dev.null")

	reg.log.Debugf("Publishing %d bytes of registration data to collective %s", len(*data), reg.cfg.RegistrationCollective)

	err = reg.connector.Publish(msg)
	if err != nil {
		reg.log.Warnf("Could not publish registration Message: %s", err.Error())
		return
	}
}
