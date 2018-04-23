package server

import (
	"context"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/server/agents"
	"github.com/sirupsen/logrus"
)

type AgentManager interface {
	RegisterAgent(ctx context.Context, name string, agent agents.Agent, conn choria.AgentConnector) error
	Logger() *logrus.Entry
	Choria() *choria.Framework
}
