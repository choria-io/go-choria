package server

import (
	"context"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/server/agents"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
)

type AgentConnector interface {
	ConnectedServer() string
	ConnectionOptions() nats.Options
	ConnectionStats() nats.Statistics
	QueueSubscribe(ctx context.Context, name string, subject string, group string, output chan *choria.ConnectorMessage) error
	Unsubscribe(name string) error
	AgentBroadcastTarget(collective string, agent string) string
	ServiceBroadcastTarget(collective string, agent string) string
}

type AgentManager interface {
	RegisterAgent(ctx context.Context, name string, agent agents.Agent, conn choria.AgentConnector) error
	Logger() *logrus.Entry
	Choria() *choria.Framework
}
