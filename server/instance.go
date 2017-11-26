package server

import (
	"context"
	"sync"

	"github.com/choria-io/go-choria/choria"
	log "github.com/sirupsen/logrus"
)

// Instance is a independant copy of Choria
type Instance struct {
	c           *choria.Framework
	connector   choria.Connector
	config      *choria.Config
	log         *log.Entry
	servers     []*choria.Server
	registrator Registrator
}

func NewInstance(c *choria.Framework) (i *Instance, err error) {
	i = &Instance{
		c:      c,
		config: c.Config,
	}

	i.log = log.WithFields(log.Fields{"identity": c.Config.Identity, "component": "server"})
	i.log.Infof("Choria version %s starting with config %s", "x.x.x", c.Config.ConfigFile)

	return i, nil
}

func (self *Instance) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	if err := self.initialConnect(ctx); err != nil {
		self.log.Errorf("Initial NATS connection failed: %s", err.Error())
		return
	}

	wg.Add(1)
	if err := self.startRegistration(ctx, wg); err != nil {
		self.log.Errorf("Could not initialize registration: %s", err.Error())
		return
	}
}
