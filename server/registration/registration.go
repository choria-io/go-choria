// Copyright (c) 2017-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package registration

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/internal/util"
	"github.com/choria-io/go-choria/protocol"
	"github.com/choria-io/go-choria/providers/registration"
	"github.com/choria-io/go-choria/server/data"

	"github.com/sirupsen/logrus"
)

type ChoriaFramework interface {
	NewMessage(payload []byte, agent string, collective string, msgType string, request inter.Message) (msg inter.Message, err error)
	Configuration() *config.Config
}

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

// Connection provides the connection to the middleware
type Connection interface {
	Publish(msg inter.Message) error
	IsConnected() bool
}

// Manager of registration plugins
type Manager struct {
	log       *logrus.Entry
	choria    ChoriaFramework
	cfg       *config.Config
	connector Connection
	datac     chan *data.RegistrationItem
	si        registration.ServerInfoSource
}

// New creates a new instance of the registration subsystem manager
func New(c ChoriaFramework, si registration.ServerInfoSource, conn Connection, logger *logrus.Entry) *Manager {
	r := &Manager{
		log:       logger.WithFields(logrus.Fields{"subsystem": "registration"}),
		choria:    c,
		si:        si,
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
			registrator, err = registration.NewFileContent(reg.cfg, reg.si, reg.log)
			if err != nil {
				return fmt.Errorf("cannot start File Content Registrator: %s", err)
			}

		case "inventory_content":
			registrator, err = registration.NewInventoryContent(reg.cfg, reg.si, reg.log)
			if err != nil {
				return fmt.Errorf("cannot start File Content Registrator: %s", err)
			}

		default:
			return fmt.Errorf("unknown registration plugin: %s", reg.cfg.Registration)
		}

		reg.log.Infof("Starting registration worker for %s", rtype)
		err = reg.RegisterProvider(ctx, wg, registrator)
		if err != nil {
			reg.log.Errorf("Could not register registration worker for %s: %s", rtype, err)
		}
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
		sleepTime := time.Duration(rand.Intn(reg.cfg.RegisterInterval)) * time.Second
		reg.log.Infof("Sleeping %s seconds before first poll due to RegistrationSplay", sleepTime)
		err := util.InterruptibleSleep(ctx, sleepTime)
		if err != nil {
			reg.log.Infof("Registration system exiting on shut down")
			return
		}
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

	if len(rmsg.Data) == 0 {
		reg.log.Warnf("Received empty data from Registratoin Plugin, skipping")
		return
	}

	if rmsg.TargetAgent == "" {
		rmsg.TargetAgent = "registration"
	}

	msg, err := reg.choria.NewMessage(rmsg.Data, rmsg.TargetAgent, reg.cfg.RegistrationCollective, "request", nil)
	if err != nil {
		reg.log.Warnf("Could not create Message for registration data: %s", err)
		return
	}

	msg.SetProtocolVersion(protocol.RequestV1)
	msg.SetReplyTo("dev.null")
	msg.SetCustomTarget(rmsg.Destination)

	if reg.connector.IsConnected() {
		err = reg.connector.Publish(msg)
		if err != nil {
			reg.log.Warnf("Could not publish registration Message: %s", err)
			return
		}
	} else {
		reg.log.Warnf("Skipping registration publish while not connected to the network")
	}
}
