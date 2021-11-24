// Copyright (c) 2018-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"sync"
)

type tPubCommand struct {
	command
}

func (p *tPubCommand) Setup() (err error) {
	if tool, ok := cmdWithFullCommand("tool"); ok {
		p.cmd = tool.Cmd().Command("pub", "Publish to middleware topics").Hidden()
	}

	return nil
}

func (p *tPubCommand) Configure() error {
	return nil
}

func (p *tPubCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	return fmt.Errorf("please use 'choria broker pub'")
}

func init() {
	cli.commands = append(cli.commands, &tPubCommand{})
}
