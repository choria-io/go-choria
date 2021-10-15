// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package testutil

import (
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/server"
)

type ChoriaNetwork struct {
	broker   *Broker
	instance *ChoriaServer
	cfg      *config.Config
}

func (cn *ChoriaNetwork) ServerInstance() *server.Instance {
	return cn.instance.Instance
}

func (cn *ChoriaNetwork) ClientURL() string {
	return cn.broker.ClientURL()
}

func (cn *ChoriaNetwork) Start() (err error) {
	cn.broker, err = StartBroker()
	if err != nil {
		return err
	}

	cn.instance, err = StartChoriaServer(cn.broker, cn.cfg)
	return err
}

func (cn *ChoriaNetwork) Stop() {
	cn.instance.Stop()
	cn.broker.Stop()

	cn.instance = nil
	cn.broker = nil
}
