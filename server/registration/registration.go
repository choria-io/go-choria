package registration

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/registration"
	"github.com/choria-io/go-choria/server/data"
	"github.com/choria-io/go-config"
	"github.com/choria-io/go-protocol/protocol"
	"github.com/sirupsen/logrus"
)

// Registrator is a full managed registration plugin
type Registrator interface {
	Init(cfg *config.Config, l *logrus.Entry)
	StartRegistration(context.Context, *sync.WaitGroup, int, chan *data.RegistrationItem)
}

// RegistrationDataProvider is a provider for data that can be registered
// into a running server instance using AddRegistrationProvider()
type RegistrationDataProvider interface {
	StartRegistration(context.Context, *sync.WaitGroup, int, chan *data.RegistrationItem)
}

// Manager of registration plugins
type Manager struct {
	log         *logrus.Entry
	choria      *choria.Framework
	cfg         *config.Config
	connector   choria.PublishableConnector
	registrator Registrator
	datac       chan *data.RegistrationItem
}

// New creates a new instance of the registration subsystem manager
func New(c *choria.Framework, conn choria.PublishableConnector, logger *logrus.Entry) *Manager {
	r := &Manager{
		log:       logger.WithFields(logrus.Fields{"subsystem": "registration"}),
		choria:    c,
		cfg:       c.Configuration(),
		connector: conn,
		datac:     make(chan *data.RegistrationItem, 1),
	}

	return r
}

// Start initializes the fully managed registration plugins and start publishing
// their data
func (reg *Manager) Start(ctx context.Context, wg *sync.WaitGroup) error {
	defer wg.Done()

	if reg.cfg.RegistrationCollective == "" {
		reg.cfg.RegistrationCollective = reg.cfg.MainCollective
	}

	var err error
	var registrator Registrator

	for _, rtype := range reg.cfg.Registration {
		switch rtype {
		case "":
			return nil
		case "file_content":
			registrator, err = registration.NewFileContent(reg.cfg, reg.log)
			if err != nil {
				return fmt.Errorf("Cannot start File Content Registrator: %s", err)
			}
		default:
			return fmt.Errorf("Unknown registration plugin: %s", reg.cfg.Registration)
		}

		reg.log.Infof("Starting registration worker for %s", rtype)
		reg.RegisterProvider(ctx, wg, registrator)
	}

	return nil
}

// RegisterProvider creates a publisher for a new provider
func (reg *Manager) RegisterProvider(ctx context.Context, wg *sync.WaitGroup, provider RegistrationDataProvider) error {
	wg.Add(1)
	go reg.registrationWorker(ctx, wg, provider)

	return nil
}

func (reg *Manager) registrationWorker(ctx context.Context, wg *sync.WaitGroup, registrator RegistrationDataProvider) {
	defer wg.Done()

	if reg.cfg.RegistrationSplay {
		sleepTime := time.Duration(rand.Intn(reg.cfg.RegisterInterval))
		reg.log.Infof("Sleeping %d seconds before first poll due to RegistrationSplay", sleepTime)
		time.Sleep(sleepTime * time.Second)
	}

	wg.Add(1)
	go registrator.StartRegistration(ctx, wg, reg.cfg.RegisterInterval, reg.datac)

	for {
		select {
		case msg := <-reg.datac:
			reg.publish(msg)
		case <-ctx.Done():
			reg.log.Infof("Registration system exiting on shut down")
			return
		}
	}
}

func (reg *Manager) publish(rmsg *data.RegistrationItem) {
	if rmsg == nil {
		reg.log.Warnf("Received nil data from Registratoin Plugin, skipping")
		return
	}

	if rmsg.Data == nil {
		reg.log.Warnf("Received nil data from Registratoin Plugin, skipping")
		return
	}

	if len(*rmsg.Data) == 0 {
		reg.log.Warnf("Received empty data from Registratoin Plugin, skipping")
		return
	}

	if rmsg.TargetAgent == "" {
		rmsg.TargetAgent = "registration"
	}

	msg, err := choria.NewMessage(string(*rmsg.Data), rmsg.TargetAgent, reg.cfg.RegistrationCollective, "request", nil, reg.choria)
	if err != nil {
		reg.log.Warnf("Could not create Message for registration data: %s", err)
		return
	}

	msg.SetProtocolVersion(protocol.RequestV1)
	msg.SetReplyTo("dev.null")
	msg.CustomTarget = rmsg.Destination

	err = reg.connector.Publish(msg)
	if err != nil {
		reg.log.Warnf("Could not publish registration Message: %s", err)
		return
	}
}
