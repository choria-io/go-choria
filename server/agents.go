package server

import (
	"fmt"

	"github.com/choria-io/go-choria/agents/discovery"
)

func (srv *Instance) setupCoreAgents() error {
	da, err := discovery.New(srv.log)
	if err != nil {
		return fmt.Errorf("Could not setup initial agents: %s", err.Error())
	}

	srv.agents.RegisterAgent("discovery", da, srv.connector)

	return nil
}
