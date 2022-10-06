// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package testutil

import (
	"context"
	"sync"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/golang/choriautil"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/golang/discovery"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/golang/rpcutil"
	"github.com/choria-io/go-choria/providers/data"
	"github.com/choria-io/go-choria/providers/data/golang/choriadata"
	"github.com/choria-io/go-choria/scout/data/scoutdata"
	"github.com/choria-io/go-choria/server"
	"github.com/sirupsen/logrus"
)

type ServerInstanceOptions struct {
	NoDataPlugins   bool
	Discovery       bool
	RPCUtilAgent    bool
	ChoriaUtilAgent bool
}

type ServerInstanceOption func(options *ServerInstanceOptions)

func ServerWithDiscovery() ServerInstanceOption {
	return func(opts *ServerInstanceOptions) {
		opts.Discovery = true
	}
}

func ServerWithRPCUtilAgent() ServerInstanceOption {
	return func(opts *ServerInstanceOptions) {
		opts.RPCUtilAgent = true
	}
}

func ServerWithChoriaUtilAgent() ServerInstanceOption {
	return func(opts *ServerInstanceOptions) {
		opts.ChoriaUtilAgent = true
	}
}

func ServerWithOutDataPlugins() ServerInstanceOption {
	return func(opts *ServerInstanceOptions) {
		opts.NoDataPlugins = true
	}
}

func StartServerInstance(ctx context.Context, wg *sync.WaitGroup, cfgFile string, log *logrus.Logger, opts ...ServerInstanceOption) (*server.Instance, error) {
	siopt := ServerInstanceOptions{}
	for _, opt := range opts {
		opt(&siopt)
	}

	cfg, err := config.NewSystemConfig(cfgFile, true)
	if err != nil {
		return nil, err
	}

	cfg.CustomLogger = log

	fw, err := choria.NewWithConfig(cfg)
	if err != nil {
		return nil, err
	}

	srv, err := server.NewInstance(fw)
	if err != nil {
		return nil, err
	}

	if !siopt.NoDataPlugins {
		data.RegisterPlugin("scout", scoutdata.ChoriaPlugin().Creator)
		data.RegisterPlugin("choria", choriadata.ChoriaPlugin().Creator)
	}

	wg.Add(1)
	err = srv.Run(ctx, wg)
	if err != nil {
		return nil, err
	}

	// wait for connection
	for {
		if srv.Status().ConnectedServer != "" {
			break
		}

		if ctx.Err() != nil {
			break
		}
	}

	if siopt.Discovery {
		da, err := discovery.New(srv.AgentManager())
		if err != nil {
			return nil, err
		}
		err = srv.AgentManager().RegisterAgent(ctx, "discovery", da, srv.Connector())
		if err != nil {
			return nil, err
		}
	}

	if siopt.RPCUtilAgent {
		rpcutilAgent, err := rpcutil.New(srv.AgentManager())
		if err != nil {
			return nil, err
		}
		err = srv.AgentManager().RegisterAgent(ctx, "rpcutil", rpcutilAgent, srv.Connector())
		if err != nil {
			return nil, err
		}
	}

	if siopt.ChoriaUtilAgent {
		cua, err := choriautil.New(srv.AgentManager())
		if err != nil {
			return nil, err
		}
		err = srv.AgentManager().RegisterAgent(ctx, "choria_util", cua, srv.Connector())
		if err != nil {
			return nil, err
		}
	}

	return srv, nil
}
