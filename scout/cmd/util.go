// Copyright (c) 2020-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package scoutcmd

import (
	"github.com/choria-io/go-choria/inter"
	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/client/discovery"
	"github.com/choria-io/go-choria/client/scoutclient"
)

func scoutClient(fw inter.Framework, opt *discovery.StandardOptions, log *logrus.Entry) (*scoutclient.ScoutClient, error) {
	co := []scoutclient.InitializationOption{
		scoutclient.Logger(log),
		scoutclient.Progress(),
		scoutclient.Discovery(&scoutclient.MetaNS{
			Options:               opt,
			Agent:                 "scout",
			DisablePipedDiscovery: false,
		}),
	}

	sc, err := scoutclient.New(fw, co...)
	if err != nil {
		return nil, err
	}

	return sc, nil
}
