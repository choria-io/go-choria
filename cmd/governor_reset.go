// Copyright (c) 2021-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"sync"

	"github.com/AlecAivazis/survey/v2"
)

type tGovResetCommand struct {
	command
	name  string
	force bool
}

func (g *tGovResetCommand) Setup() (err error) {
	if gov, ok := cmdWithFullCommand("governor"); ok {
		g.cmd = gov.Cmd().Command("reset", "Evicts all workers")
		g.cmd.Arg("name", "The name for the Governor to managed").Required().StringVar(&g.name)
		g.cmd.Flag("force", "Reset without prompting").Short('f').UnNegatableBoolVar(&g.force)
	}

	return nil
}

func (g *tGovResetCommand) Configure() (err error) {
	return commonConfigure()
}

func (g *tGovResetCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	gov, _, err := c.NewGovernorManager(ctx, g.name, 0, 0, 1, false, nil)
	if err != nil {
		return err
	}

	entries, err := gov.Active()
	if err != nil {
		return err
	}

	if entries == 0 {
		fmt.Println("No lease entries to remove")
		return nil
	}

	if !g.force {
		ans := false
		err = survey.AskOne(&survey.Confirm{
			Message: fmt.Sprintf("Reset %s with %d lease entries?", g.name, entries),
			Default: ans,
		}, &ans)
		if err != nil {
			return err
		}
		if !ans {
			return nil
		}
	}

	return gov.Reset()
}

func init() {
	cli.commands = append(cli.commands, &tGovResetCommand{})
}
