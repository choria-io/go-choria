// Copyright (c) 2020-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"sync"

	"github.com/choria-io/go-choria/client/discovery"
	scoutcmd "github.com/choria-io/go-choria/scout/cmd"
)

type sMaintenanceCommand struct {
	fo *discovery.StandardOptions

	checks  []string
	json    bool
	verbose bool

	command
}

func (s *sMaintenanceCommand) Setup() (err error) {
	if scout, ok := cmdWithFullCommand("scout"); ok {
		s.cmd = scout.Cmd().Command("maintenance", "Place one or more checks into maintenance mode")

		s.fo = discovery.NewStandardOptions()
		s.fo.AddFilterFlags(s.cmd)
		s.fo.AddSelectionFlags(s.cmd)

		s.cmd.Flag("json", "JSON format output").UnNegatableBoolVar(&s.json)
		s.cmd.Flag("verbose", "Show verbose output").Short('v').UnNegatableBoolVar(&s.verbose)
	}

	return nil
}

func (s *sMaintenanceCommand) Configure() error {
	return commonConfigure()
}

func (s *sMaintenanceCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	s.fo.SetDefaultsFromChoria(c)
	trigger, err := scoutcmd.NewMaintenanceCommand(s.fo, c, s.checks, s.json, debug || s.verbose, c.Config.Color, c.Logger("scout"))
	if err != nil {
		return err
	}

	wg.Add(1)
	return trigger.Run(ctx, wg)
}

func init() {
	cli.commands = append(cli.commands, &sMaintenanceCommand{})
}
