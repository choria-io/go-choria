package server

import (
	"context"
	"fmt"

	"github.com/choria-io/go-choria/agents/discovery"
)

func (srv *Instance) setupCoreAgents(ctx context.Context) error {
	da, err := discovery.New(srv.log)
	if err != nil {
		return fmt.Errorf("Could not setup initial agents: %s", err.Error())
	}

	srv.agents.RegisterAgent(ctx, "discovery", da, srv.connector)

	return nil
}
