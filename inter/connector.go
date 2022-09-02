// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package inter

import (
	"context"

	"github.com/nats-io/nats.go"
)

// Connector is the interface a connector must implement to be valid be it NATS, Stomp, Testing etc
type Connector interface {
	AgentBroadcastTarget(collective string, agent string) string
	ChanQueueSubscribe(name string, subject string, group string, capacity int) (chan ConnectorMessage, error)
	Close()
	Connect(ctx context.Context) (err error)
	ConnectedServer() string
	ConnectionOptions() nats.Options
	ConnectionStats() nats.Statistics
	InboxPrefix() string
	IsConnected() bool
	Nats() *nats.Conn
	NodeDirectedTarget(collective string, identity string) string
	Publish(msg Message) error
	PublishRaw(target string, data []byte) error
	PublishRawMsg(msg *nats.Msg) error
	QueueSubscribe(ctx context.Context, name string, subject string, group string, output chan ConnectorMessage) error
	ReplyTarget(msg Message) (string, error)
	RequestRawMsgWithContext(ctx context.Context, msg *nats.Msg) (*nats.Msg, error)
	ServiceBroadcastTarget(collective string, agent string) string
	Unsubscribe(name string) error
}
