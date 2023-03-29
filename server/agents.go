// Copyright (c) 2017-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"

	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/server/agents"
	"github.com/sirupsen/logrus"
)

type AgentManager interface {
	RegisterAgent(ctx context.Context, name string, agent agents.Agent, conn inter.AgentConnector) error
	UnregisterAgent(name string, conn inter.AgentConnector) error
	ReplaceAgent(name string, agent agents.Agent) error
	Logger() *logrus.Entry
	Choria() inter.Framework
}
