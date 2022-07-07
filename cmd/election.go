// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"sync"
)

type tElectionCommand struct {
	command
}

func (c *tElectionCommand) Setup() (err error) {
	c.cmd = cli.app.Command("election", "Distributed Leader Election Tools").Alias("elect").Alias("elec")
	c.cmd.Flag("config", "Config file to use").PlaceHolder("FILE").StringVar(&configFile)

	return nil
}

func (c *tElectionCommand) Configure() error {
	return nil
}

func (c *tElectionCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	return nil
}

func init() {
	cli.commands = append(cli.commands, &tElectionCommand{})
}
