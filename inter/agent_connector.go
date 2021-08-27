package inter

import (
	"context"

	"github.com/nats-io/nats.go"
)

// AgentConnector provides the minimal Connector features for subscribing and unsubscribing agents
type AgentConnector interface {
	ConnectedServer() string
	ConnectionOptions() nats.Options
	ConnectionStats() nats.Statistics
	QueueSubscribe(ctx context.Context, name string, subject string, group string, output chan ConnectorMessage) error
	Unsubscribe(name string) error
	AgentBroadcastTarget(collective string, agent string) string
	ServiceBroadcastTarget(collective string, agent string) string
}
