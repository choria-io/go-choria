package submission

import (
	"context"

	"github.com/nats-io/nats.go"
)

type Connector interface {
	PublishRawMsg(msg *nats.Msg) error
	RequestRawMsgWithContext(ctx context.Context, msg *nats.Msg) (*nats.Msg, error)
}
