// Copyright (c) 2018-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"sync"
)

type toolCommand struct {
	command
}

func (t *toolCommand) Setup() (err error) {
	t.cmd = cli.app.Command("tool", "Various utilities for debugging and verification of Choria Networks").Alias("t")

	return nil
}

func (t *toolCommand) Configure() error {
	return nil
}

func (t *toolCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	return nil
}

func init() {
	cli.commands = append(cli.commands, &toolCommand{})
}
