package server

import (
	"context"

	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/server/agents"
	"github.com/sirupsen/logrus"
)

type AgentManager interface {
	RegisterAgent(ctx context.Context, name string, agent agents.Agent, conn inter.AgentConnector) error
	Logger() *logrus.Entry
	Choria() inter.Framework
}
