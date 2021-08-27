package inter

import (
	"github.com/nats-io/nats.go"
)

// ConnectorInfo provides information about the active connection without giving access to the connection
type ConnectorInfo interface {
	ConnectedServer() string
	ConnectionOptions() nats.Options
	ConnectionStats() nats.Statistics
}
