// Copyright (c) 2017-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package adapter

import (
	"context"
	"fmt"
	"sync"

	"github.com/choria-io/go-choria/broker/adapter/streams"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/protocol"
	"github.com/choria-io/go-choria/srvcache"
	log "github.com/sirupsen/logrus"
)

type ChoriaFramework interface {
	Configuration() *config.Config
	MiddlewareServers() (servers srvcache.Servers, err error)
	NewConnector(ctx context.Context, servers func() (srvcache.Servers, error), name string, logger *log.Entry) (conn inter.Connector, err error)
	NewRequestFromTransportJSON(payload []byte, skipvalidate bool) (msg protocol.Request, err error)
	NewReplyFromTransportJSON(payload []byte, skipvalidate bool) (msg protocol.Reply, err error)
}

type adapter interface {
	Init(ctx context.Context, cm inter.ConnectionManager) (err error)
	Process(ctx context.Context, wg *sync.WaitGroup)
}

func startAdapter(ctx context.Context, a adapter, c inter.ConnectionManager, wg *sync.WaitGroup) error {
	err := a.Init(ctx, c)
	if err != nil {
		return fmt.Errorf("could not initialize adapter %s: %s", a, err)
	}

	wg.Add(1)
	go a.Process(ctx, wg)

	return nil
}

func RunAdapters(ctx context.Context, c ChoriaFramework, wg *sync.WaitGroup) error {
	for _, a := range c.Configuration().Choria.Adapters {
		atype := c.Configuration().Option(fmt.Sprintf("plugin.choria.adapter.%s.type", a), "")
		if atype == "" {
			return fmt.Errorf("could not determine type for adapter %s, set plugin.choria.adapter.%s.type", a, a)
		}

		switch atype {
		case "jetstream", "choria_streams":
			n, err := streams.Create(a, c)
			if err != nil {
				return fmt.Errorf("could not start choria_streams adapter: %s", err)
			}

			log.Infof("Starting %s Protocol Adapter %s", atype, a)
			err = startAdapter(ctx, n, c, wg)
			if err != nil {
				return fmt.Errorf("could not start choria_streams adapter: %s", err)
			}

		case "nats_stream":
			return fmt.Errorf("the NATS Streaming Server adapter has been deprecated")

		default:
			return fmt.Errorf("unknown Protocol Adapter type %s", atype)
		}
	}

	return nil
}
