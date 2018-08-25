package events

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/srvcache"
	"github.com/tidwall/gjson"
)

// SubscribeConnector is a connection to the middleware
type SubscribeConnector interface {
	QueueSubscribe(ctx context.Context, name string, subject string, group string, output chan *choria.ConnectorMessage) error
	ConnectedServer() string
}

// ViewOptions configure the view command
type ViewOptions struct {
	Choria          *choria.Framework
	TypeFilter      string
	ComponentFilter string
	Output          io.Writer
	Connector       SubscribeConnector
	Debug           bool
}

// View connects and stream events to Output
func View(ctx context.Context, opt *ViewOptions) error {
	var err error

	log := opt.Choria.Logger("event_viewer")
	opt.Connector, err = opt.Choria.NewConnector(ctx, brokerUrls(opt.Choria), opt.Choria.Certname(), log)
	if err != nil {
		return fmt.Errorf("cannot connect: %s", err)
	}

	if opt.Output == nil {
		opt.Output = os.Stdout
	}

	fmt.Fprintf(opt.Output, "Waiting for events from topic choria.lifecycle.event on %s\n", opt.Connector.ConnectedServer())

	return WriteEvents(ctx, opt)
}

func brokerUrls(fw *choria.Framework) func() ([]srvcache.Server, error) {
	return func() ([]srvcache.Server, error) { return fw.MiddlewareServers() }
}

// WriteEvents views the event stream to the output
func WriteEvents(ctx context.Context, opt *ViewOptions) error {
	events := make(chan *choria.ConnectorMessage, 100)

	err := opt.Connector.QueueSubscribe(ctx, choria.NewRequestID(), "choria.lifecycle.event", "", events)
	if err != nil {
		return fmt.Errorf("could not subscribe to event source: %s", err)
	}

	for {
		select {
		case e := <-events:
			etype, _ := typeForEventJSON(e.Data)
			if opt.TypeFilter != "" && etype != EventTypes[opt.TypeFilter] {
				continue
			}

			if opt.ComponentFilter != "" {
				if c, err := componentForEvent(e.Data); err == nil && c != opt.ComponentFilter {
					continue
				}
			}

			if opt.Debug {
				fmt.Fprintf(opt.Output, "%s\n", string(e.Data))
				continue
			}

			fmt.Fprintf(opt.Output, "%s %s\n", time.Now().Format("15:04:05"), typeBytesToString(e.Data, etype))

		case <-ctx.Done():
			return nil
		}
	}
}

func typeBytesToString(e []byte, t Type) string {
	format := "unknown event"
	var fields []string

	switch t {
	case Startup:
		format = "[startup] %s: %s component version %s"
		fields = []string{"identity", "component", "version"}
	default:
		return "Unknown event"
	}

	args := []interface{}{}

	for _, f := range fields {
		jval := gjson.GetBytes(e, f)
		val := "unknown"

		if jval.Exists() {
			if jval.Type == gjson.String {
				val = jval.String()
			} else {
				val = jval.Raw
			}
		}

		args = append(args, val)
	}

	return fmt.Sprintf(format, args...)
}

func typeForEventJSON(e []byte) (Type, error) {
	protocol := gjson.GetBytes(e, "protocol")
	if !protocol.Exists() {
		return 0, fmt.Errorf("no protocol field present")
	}

	switch protocol.String() {
	case "choria:lifecycle:startup:1":
		return Startup, nil
	default:
		return 0, fmt.Errorf("cannot determine event type")
	}
}

func componentForEvent(e []byte) (string, error) {
	component := gjson.GetBytes(e, "component")
	if !component.Exists() {
		return "", fmt.Errorf("no component field present")
	}

	return component.String(), nil
}
