package adapter

import (
	"context"
	"fmt"
	"sync"

	"github.com/choria-io/go-choria/broker/adapter/natsstream"
	"github.com/choria-io/go-choria/choria"
	log "github.com/sirupsen/logrus"
)

func RunAdapters(ctx context.Context, c *choria.Framework, wg *sync.WaitGroup) error {
	for _, a := range c.Config.Choria.Adapters {
		atype := c.Config.Option(fmt.Sprintf("plugin.choria.adapter.%s.type", a), "")
		if atype == "" {
			return fmt.Errorf("Could not determine type for adapter %s, set plugin.choria.adapter.%s.type", a, a)
		}

		switch atype {
		case "nats_stream":
			n, err := natsstream.Create(a, c)
			if err != nil {
				return fmt.Errorf("Could not start nats_stream adapter: %s", err)
			}

			log.Infof("Starting %s Protocol Adapter %s", atype, a)

			err = n.Init(ctx, c)
			if err != nil {
				return fmt.Errorf("Could not initialize adapter %s: %s", a, err)
			}

			wg.Add(1)
			go n.Process(ctx, wg)
		default:
			return fmt.Errorf("Unknown Protocol Adapter type %s", atype)
		}
	}

	return nil
}
