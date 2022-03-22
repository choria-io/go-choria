// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package testutil

import (
	"context"
	"sync"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/providers/data"
	"github.com/choria-io/go-choria/providers/data/golang/choriadata"
	"github.com/choria-io/go-choria/scout/data/scoutdata"
	"github.com/choria-io/go-choria/server"
	"github.com/sirupsen/logrus"
)

func StartServerInstance(ctx context.Context, wg *sync.WaitGroup, cfgFile string, log *logrus.Logger) (*server.Instance, error) {
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

	data.RegisterPlugin("scout", scoutdata.ChoriaPlugin().Creator)
	data.RegisterPlugin("choria", choriadata.ChoriaPlugin().Creator)

	wg.Add(1)
	err = srv.Run(ctx, wg)
	if err != nil {
		return nil, err
	}

	return srv, nil
}
