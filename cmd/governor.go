// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"sync"

	"github.com/choria-io/go-choria/internal/fs"
)

type tGovCommand struct {
	command
}

func (g *tGovCommand) Setup() (err error) {
	g.cmd = cli.app.Command("governor", "Distributed Concurrency Control System management").Alias("gov")
	g.cmd.Flag("config", "Config file to use").PlaceHolder("FILE").StringVar(&configFile)
	g.cmd.CheatFile(fs.FS, "governor", "cheats/governor.md")

	return nil
}

func (g *tGovCommand) Configure() error {
	return nil
}

func (g *tGovCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	return nil
}

func init() {
	cli.commands = append(cli.commands, &tGovCommand{})
}
