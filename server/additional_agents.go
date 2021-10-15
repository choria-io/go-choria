// Copyright (c) 2018-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"sync"

	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/server/agents"
	"github.com/sirupsen/logrus"
)

// AgentInitializer is a function signature for a function that can register an agent
type AgentInitializer func(context.Context, *agents.Manager, inter.AgentConnector, *logrus.Entry) error

var additionalAgents []AgentInitializer
var aamu *sync.Mutex

func init() {
	additionalAgents = []AgentInitializer{}
	aamu = &sync.Mutex{}
}

// RegisterAdditionalAgent adds an agent to a running server
// this should be used for compile time injection of agents
// other than the ones that ship with choria
func RegisterAdditionalAgent(i AgentInitializer) {
	aamu.Lock()
	defer aamu.Unlock()

	additionalAgents = append(additionalAgents, i)
}

func (srv *Instance) setupAdditionalAgents(ctx context.Context) error {
	aamu.Lock()
	agents := make([]AgentInitializer, len(additionalAgents))
	copy(agents, additionalAgents)
	aamu.Unlock()

	for _, initializer := range agents {
		err := initializer(ctx, srv.agents, srv.connector, srv.log)
		if err != nil {
			return err
		}
	}

	return nil
}
