// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import "sync"

type pGenerateCommand struct {
	command
}

func (g *pGenerateCommand) Setup() (err error) {
	if tool, ok := cmdWithFullCommand("plugin"); ok {
		g.cmd = tool.Cmd().Command("generate", "Generates choria related data")
	}

	return nil
}

func (g *pGenerateCommand) Configure() error {
	return nil
}

func (g *pGenerateCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	return nil
}

func init() {
	cli.commands = append(cli.commands, &pGenerateCommand{})
}
