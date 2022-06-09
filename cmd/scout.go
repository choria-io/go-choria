// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import "sync"

type scoutCommand struct {
	command
}

func (m *scoutCommand) Setup() (err error) {
	m.cmd = cli.app.Command("scout", "Choria Scout management and reporting")
	m.cmd.Flag("config", "Config file to use").PlaceHolder("FILE").StringVar(&configFile)

	return nil
}

func (m *scoutCommand) Configure() error {
	return nil
}

func (m *scoutCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	return nil
}

func init() {
	cli.commands = append(cli.commands, &scoutCommand{})
}
