// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package testbroker

import (
	"context"
	"sync"

	"github.com/choria-io/go-choria/broker/network"
	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/config"
	"github.com/sirupsen/logrus"
)

func StartNetworkBrokerWithConfigFile(ctx context.Context, wg *sync.WaitGroup, file string, log *logrus.Logger) (*network.Server, error) {
	cfg, err := config.NewConfig(file)
	if err != nil {
		return nil, err
	}
	cfg.CustomLogger = log

	fw, err := choria.NewWithConfig(cfg)
	if err != nil {
		return nil, err
	}

	srv, err := network.NewServer(fw, fw.BuildInfo(), fw.Configuration().LogLevel == "debug")
	if err != nil {
		return nil, err
	}

	wg.Add(1)
	go srv.Start(ctx, wg)

	return srv, nil
}
