package server

import (
	"context"
	"fmt"

	"github.com/choria-io/go-choria/agents/choriautil"
	"github.com/choria-io/go-choria/agents/discovery"
	"github.com/choria-io/go-choria/agents/provision"
	"github.com/choria-io/go-choria/agents/rpcutil"
	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/server/agents"
	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/build"
)

type AgentManager interface {
	RegisterAgent(ctx context.Context, name string, agent agents.Agent, conn choria.AgentConnector) error
	Logger() *logrus.Entry
	Choria() *choria.Framework
}

func (srv *Instance) setupCoreAgents(ctx context.Context) error {
	da, err := discovery.New(srv.agents)
	if err != nil {
		return fmt.Errorf("Could not setup initial agents: %s", err)
	}

	srv.agents.RegisterAgent(ctx, "discovery", da, srv.connector)

	cu, err := choriautil.New(srv.agents)
	if err != nil {
		return fmt.Errorf("Could not setup choria_util agent: %s", err)
	}

	srv.agents.RegisterAgent(ctx, "choria_util", cu, srv.connector)

	rpcu, err := rpcutil.New(srv.agents)
	if err != nil {
		return fmt.Errorf("Could not setup rpcutil agent: %s", err)
	}

	srv.agents.RegisterAgent(ctx, "rpcutil", rpcu, srv.connector)

	if build.ProvisionBrokerURLs != "" && build.ProvisionAgent == "true" {
		pa, err := provision.New(srv.agents)
		if err != nil {
			return fmt.Errorf("Could not setup choria_provision agent: %s", err)
		}

		srv.agents.RegisterAgent(ctx, "choria_provision", pa, srv.connector)
	}

	return nil
}
