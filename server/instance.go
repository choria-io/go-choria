package server

import (
	"context"
	"sync"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/server/agents"
	"github.com/choria-io/go-choria/server/registration"

	log "github.com/sirupsen/logrus"
)

// Instance is an independant copy of Choria
type Instance struct {
	c            *choria.Framework
	connector    choria.Connector
	cfg          *choria.Config
	log          *log.Entry
	servers      []*choria.Server
	registration *registration.Manager
	agents       map[string]*agents.Agent

	agentmu *sync.Mutex
}

func NewInstance(c *choria.Framework) (i *Instance, err error) {
	i = &Instance{
		c:   c,
		cfg: c.Config,
	}

	i.log = log.WithFields(log.Fields{"identity": c.Config.Identity, "component": "server"})

	return i, nil
}

func (srv *Instance) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	if err := srv.initialConnect(ctx); err != nil {
		srv.log.Errorf("Initial NATS connection failed: %s", err.Error())
		return
	}

	srv.registration = registration.New(srv.c, srv.connector, srv.log)

	wg.Add(1)
	if err := srv.registration.Start(ctx, wg); err != nil {
		srv.log.Errorf("Could not initialize registration: %s", err.Error())
		return
	}
}
