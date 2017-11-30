package server

import (
	"context"
	"fmt"
	"sync"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/server/agents"
	"github.com/choria-io/go-choria/server/registration"

	log "github.com/sirupsen/logrus"
)

// Instance is an independant copy of Choria
type Instance struct {
	fw           *choria.Framework
	connector    choria.Connector
	cfg          *choria.Config
	log          *log.Entry
	servers      []*choria.Server
	registration *registration.Manager
	agents       *agents.Manager

	agentmu *sync.Mutex
}

func NewInstance(fw *choria.Framework) (i *Instance, err error) {
	i = &Instance{
		fw:  fw,
		cfg: fw.Config,
	}

	i.log = log.WithFields(log.Fields{"identity": fw.Config.Identity, "component": "server"})
	i.agents = agents.New(fw, i.log)

	return i, nil
}

func (srv *Instance) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	if err := srv.initialConnect(ctx); err != nil {
		srv.log.Errorf("Initial NATS connection failed: %s", err.Error())
		return
	}

	srv.registration = registration.New(srv.fw, srv.connector, srv.log)

	wg.Add(1)
	if err := srv.registration.Start(ctx, wg); err != nil {
		srv.log.Errorf("Could not initialize registration: %s", err.Error())
		srv.connector.Close()

		return
	}

	if err := srv.setupCoreAgents(); err != nil {
		srv.log.Errorf("Could not initialize initial core agents: %s", err.Error())
		srv.connector.Close()

		return
	}

	if err := srv.subscribeNode(); err != nil {
		srv.log.Errorf("Could not initialize node: %s", err.Error())
		srv.connector.Close()

		return
	}
}

func (srv *Instance) subscribeNode() error {
	var err error

	for _, collective := range srv.cfg.Collectives {
		target := srv.connector.NodeDirectedTarget(collective, srv.cfg.Identity)

		srv.log.Infof("Subscribing node %s to %s", srv.cfg.Identity, target)

		err = srv.connector.Subscribe(fmt.Sprintf("node.%s", collective), target, "")
		if err != nil {
			return fmt.Errorf("Could not subscribe to node directed targets: %s", err.Error())
		}
	}

	return nil
}
