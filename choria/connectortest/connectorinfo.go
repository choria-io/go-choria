package connectortest

import (
	nats "github.com/nats-io/go-nats"
)

type ConnectorInfo struct {
	Server  string
	Options nats.Options
	Stats   nats.Statistics
}

func (i *ConnectorInfo) ConnectedServer() string {
	return i.Server
}

func (i *ConnectorInfo) ConnectionOptions() nats.Options {
	return i.Options
}

func (i *ConnectorInfo) ConnectionStats() nats.Statistics {
	return i.Stats
}
