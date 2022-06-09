// Copyright (c) 2019-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import "sync"

type machineCommand struct {
	command
}

func (m *machineCommand) Setup() (err error) {
	m.cmd = cli.app.Command("machine", "Autonomous Agent management")
	m.cmd.Flag("config", "Config file to use").PlaceHolder("FILE").StringVar(&configFile)

	return nil
}

func (m *machineCommand) Configure() error {
	return nil
}

func (m *machineCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	return nil
}

func init() {
	cli.commands = append(cli.commands, &machineCommand{})
}
