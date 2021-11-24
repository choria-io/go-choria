// Copyright (c) 2018-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"sync"
)

type tSubCommand struct {
	command
}

func (s *tSubCommand) Setup() (err error) {
	if tool, ok := cmdWithFullCommand("tool"); ok {
		s.cmd = tool.Cmd().Command("sub", "Subscribe to middleware topics").Hidden()
	}

	return nil
}

func (s *tSubCommand) Configure() error {
	return commonConfigure()
}

func (s *tSubCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	return fmt.Errorf("please use 'choria broker sub'")
}

func init() {
	cli.commands = append(cli.commands, &tSubCommand{})
}
