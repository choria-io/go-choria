// Copyright (c) 2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import "sync"

type execCommand struct {
	command
}

func (c *execCommand) Setup() (err error) {
	c.cmd = cli.app.Command("executor", "Long running process executor management").Alias("exec")

	return nil
}

func (c *execCommand) Configure() error {
	return nil
}
func (c *execCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	return nil
}

func init() {
	cli.commands = append(cli.commands, &execCommand{})
}
