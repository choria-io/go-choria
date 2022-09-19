// Copyright (c) 2020-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"sync"

	scoutcmd "github.com/choria-io/go-choria/scout/cmd"
)

type sStatusCommand struct {
	identity string
	json     bool
	verbose  bool

	command
}

func (s *sStatusCommand) Setup() (err error) {
	if scout, ok := cmdWithFullCommand("scout"); ok {
		s.cmd = scout.Cmd().Command("status", "Retrieve check statuses from an agent")
		s.cmd.Arg("identity", "Node to retrieve data from").Required().StringVar(&s.identity)
		s.cmd.Flag("json", "JSON format output").UnNegatableBoolVar(&s.json)
		s.cmd.Flag("verbose", "Show verbose output").Short('v').UnNegatableBoolVar(&s.verbose)
	}

	return nil
}

func (s *sStatusCommand) Configure() error {
	return commonConfigure()
}

func (s *sStatusCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	status, err := scoutcmd.NewStatusCommand(c, s.identity, s.json, debug || s.verbose, c.Config.Color, c.Logger("scout"))
	if err != nil {
		return err
	}

	wg.Add(1)
	return status.Run(ctx, wg)
}

func init() {
	cli.commands = append(cli.commands, &sStatusCommand{})
}
