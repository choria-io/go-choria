package adapter

import (
	"context"
	"fmt"
	"sync"

	"github.com/choria-io/go-choria/broker/adapter/jetstream"
	"github.com/choria-io/go-choria/broker/adapter/natsstream"
	"github.com/choria-io/go-choria/choria"
	log "github.com/sirupsen/logrus"
)

type adapter interface {
	Init(ctx context.Context, cm choria.ConnectionManager) (err error)
	Process(ctx context.Context, wg *sync.WaitGroup)
}

func startAdapter(ctx context.Context, a adapter, c *choria.Framework, wg *sync.WaitGroup) error {
	err := a.Init(ctx, c)
	if err != nil {
		return fmt.Errorf("could not initialize adapter %s: %s", a, err)
	}

	wg.Add(1)
	go a.Process(ctx, wg)

	return nil
}

func RunAdapters(ctx context.Context, c *choria.Framework, wg *sync.WaitGroup) error {
	for _, a := range c.Config.Choria.Adapters {
		atype := c.Config.Option(fmt.Sprintf("plugin.choria.adapter.%s.type", a), "")
		if atype == "" {
			return fmt.Errorf("could not determine type for adapter %s, set plugin.choria.adapter.%s.type", a, a)
		}

		switch atype {
		case "jetstream":
			n, err := jetstream.Create(a, c)
			if err != nil {
				return fmt.Errorf("could not start jetstream adapter: %s", err)
			}

			log.Infof("Starting %s Protocol Adapter %s", atype, a)
			err = startAdapter(ctx, n, c, wg)
			if err != nil {
				return fmt.Errorf("could not start jetstream adapter: %s", err)
			}

		case "nats_stream":
			n, err := natsstream.Create(a, c)
			if err != nil {
				return fmt.Errorf("could not start nats_stream adapter: %s", err)
			}

			log.Infof("Starting %s Protocol Adapter %s", atype, a)
			err = startAdapter(ctx, n, c, wg)
			if err != nil {
				return fmt.Errorf("could not start nats_stream adapter: %s", err)
			}

		default:
			return fmt.Errorf("unknown Protocol Adapter type %s", atype)
		}
	}

	return nil
}
