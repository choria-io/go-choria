package server

import (
	"fmt"

	"github.com/choria-io/go-choria/choria"
	log "github.com/sirupsen/logrus"
)

// Instance is a independant copy of Choria
type Instance struct {
	c           *choria.Framework
	connector   choria.Connector
	config      *choria.Config
	logger      *log.Entry
	servers     []*choria.Server
	registrator Registrator
}

func NewInstance(c *choria.Framework) (i *Instance, err error) {
	i = &Instance{
		c:      c,
		config: c.Config,
	}

	i.logger = log.WithFields(log.Fields{"identity": c.Config.Identity, "component": "server"})
	i.logger.Infof("Choria version %s starting with config %s", "x.x.x", c.Config.ConfigFile)

	if err := i.initialConnect(); err != nil {
		return nil, fmt.Errorf("Initial NATS connection failed: %s", err.Error())
	}

	if err := i.startRegistration(); err != nil {
		return nil, fmt.Errorf("Could not initialize registration: %s", err.Error())
	}

	return i, nil
}

func (self *Instance) initialConnect() error {
	servers := func() ([]choria.Server, error) {
		return self.c.MiddlewareServers()
	}

	_, err := servers()
	if err != nil {
		return fmt.Errorf("Could not find initial NATS servers: %s", err.Error())
	}

	self.connector, err = self.c.NewConnector(servers, self.c.Certname(), self.logger)
	if err != nil {
		return fmt.Errorf("Could not create connector: %s", err.Error())
	}

	return nil
}
