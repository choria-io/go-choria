package inter

import (
	"context"

	"github.com/nats-io/nats.go"
)

// RawNATSConnector sends NATS messages directly
type RawNATSConnector interface {
	PublishRaw(target string, data []byte) error
	PublishRawMsg(msg *nats.Msg) error
	RequestRawMsgWithContext(ctx context.Context, msg *nats.Msg) (*nats.Msg, error)
}
