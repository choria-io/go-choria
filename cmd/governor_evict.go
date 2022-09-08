// Copyright (c) 2021-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"sync"

	"github.com/AlecAivazis/survey/v2"
	"github.com/choria-io/go-choria/lifecycle"
)

type tGovEvictCommand struct {
	command
	name  string
	seq   uint64
	force bool
}

func (g *tGovEvictCommand) Setup() (err error) {
	if gov, ok := cmdWithFullCommand("governor"); ok {
		g.cmd = gov.Cmd().Command("evict", "Evicts a specific worker")
		g.cmd.Arg("name", "The name for the Governor to managed").Required().StringVar(&g.name)
		g.cmd.Arg("worker", "The lease ID to remove").Required().Uint64Var(&g.seq)
		g.cmd.Flag("force", "Evict without prompting").Short('f').UnNegatableBoolVar(&g.force)
	}

	return nil
}

func (g *tGovEvictCommand) Configure() (err error) {
	return commonConfigure()
}

func (g *tGovEvictCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	gov, conn, err := c.NewGovernorManager(ctx, g.name, 0, 0, 1, false, nil)
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
			Message: fmt.Sprintf("Evict lease %d from %s?", g.seq, g.name),
			Default: ans,
		}, &ans)
		if err != nil {
			return err
		}
		if !ans {
			return nil
		}
	}

	name, err := gov.Evict(g.seq)
	if err != nil {
		return err
	}

	event, err := lifecycle.New(lifecycle.Governor, lifecycle.Identity(name), lifecycle.Component("CLI"), lifecycle.GovernorSequence(g.seq), lifecycle.GovernorName(g.name), lifecycle.GovernorType(lifecycle.GovernorEvictEvent))
	if err == nil {
		lifecycle.PublishEvent(event, conn)
		conn.Close()
	}

	fmt.Printf("Evicted %q from slot %d\n", name, g.seq)

	return nil
}

func init() {
	cli.commands = append(cli.commands, &tGovEvictCommand{})
}
