// Copyright (c) 2021-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"sync"

	"github.com/choria-io/go-choria/internal/util"
)

type tGovDeleteCommand struct {
	command
	name  string
	force bool
}

func (g *tGovDeleteCommand) Setup() (err error) {
	if gov, ok := cmdWithFullCommand("governor"); ok {
		g.cmd = gov.Cmd().Command("delete", "Deletes a Governor").Alias("rm")
		g.cmd.Arg("name", "The name for the Governor to managed").Required().StringVar(&g.name)
		g.cmd.Flag("force", "Reset without prompting").Short('f').UnNegatableBoolVar(&g.force)
	}

	return nil
}

func (g *tGovDeleteCommand) Configure() (err error) {
	return commonConfigure()
}

func (g *tGovDeleteCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	gov, _, err := c.NewGovernorManager(ctx, g.name, 0, 0, 1, false, nil)
	if err != nil {
		return err
	}

	entries, err := gov.Active()
	if err != nil {
		return err
	}

	if !g.force {
		ans, err := util.PromptForConfirmation("Delete %s with %d active lease entries?", g.name, entries)
		if err != nil {
			return err
		}
		if !ans {
			return nil
		}
	}

	return gov.Stream().Delete()
}

func init() {
	cli.commands = append(cli.commands, &tGovDeleteCommand{})
}
