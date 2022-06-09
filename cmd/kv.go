// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"sync"

	"github.com/choria-io/go-choria/internal/fs"
)

type kvCommand struct {
	command
}

func (k *kvCommand) Setup() (err error) {
	k.cmd = cli.app.Command("kv", "Key-Value Store management")
	k.cmd.Flag("config", "Config file to use").PlaceHolder("FILE").StringVar(&configFile)
	k.cmd.CheatFile(fs.FS, "kv", "cheats/kv.md")

	return nil
}

func (k *kvCommand) Configure() error {
	return nil
}

func (k *kvCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	return nil
}

func init() {
	cli.commands = append(cli.commands, &kvCommand{})
}
