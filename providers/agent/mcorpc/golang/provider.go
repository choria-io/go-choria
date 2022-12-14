// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package agents

import (
	"context"
	"fmt"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/golang/choriautil"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/golang/discovery"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/golang/rpcutil"
	"github.com/choria-io/go-choria/server"
	"github.com/choria-io/go-choria/server/agents"
	"github.com/sirupsen/logrus"
)

// Provider is a Agent Provider capable of executing compiled mcollective compatible agents written in Go
type Provider struct {
}

// Initialize configures the agent provider
func (p *Provider) Initialize(_ *config.Config, _ *logrus.Entry) {}

// Version reports the version for this provider
func (p *Provider) Version() string {
	return fmt.Sprintf("%s version %s", p.PluginName(), p.PluginVersion())
}

// RegisterAgents registers known ruby agents using a shimm agent
func (p *Provider) RegisterAgents(ctx context.Context, mgr server.AgentManager, connector inter.AgentConnector, _ *logrus.Entry) error {
	var agent agents.Agent

	agent, err := discovery.New(mgr)
	if err != nil {
		return err
	}

	err = mgr.RegisterAgent(ctx, "discovery", agent, connector)
	if err != nil {
		return err
	}

	agent, err = rpcutil.New(mgr)
	if err != nil {
		return err
	}

	err = mgr.RegisterAgent(ctx, "rpcutil", agent, connector)
	if err != nil {
		return err
	}

	agent, err = choriautil.New(mgr)
	if err != nil {
		return err
	}

	err = mgr.RegisterAgent(ctx, "choria_util", agent, connector)
	if err != nil {
		return err
	}

	return nil
}
