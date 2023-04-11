// Copyright (c) 2023, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"sync"
)

type machinePluginsCommand struct {
	command
}

func (p *machinePluginsCommand) Setup() (err error) {
	if machine, ok := cmdWithFullCommand("machine"); ok {
		p.cmd = machine.Cmd().Command("plugins", "Manage specifications for the plugins watcher")
	}
	return nil
}

func (p *machinePluginsCommand) Configure() error {
	return nil
}

func (p *machinePluginsCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	return nil
}

func init() {
	cli.commands = append(cli.commands, &machinePluginsCommand{})
}
