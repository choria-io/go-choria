// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"sync"
)

type pluginCommand struct {
	command
}

func (t *pluginCommand) Setup() (err error) {
	t.cmd = cli.app.Command("plugin", "Plugin Inspection, Generation and Documentation")
	t.cmd.Flag("config", "Config file to use").PlaceHolder("FILE").StringVar(&configFile)

	return nil
}

func (t *pluginCommand) Configure() error {
	return nil
}

func (t *pluginCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	return nil
}

func init() {
	cli.commands = append(cli.commands, &pluginCommand{})
}
