// Copyright (c) 2021-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"sync"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/choria-io/go-choria/providers/governor"
)

type tGovAddCommand struct {
	command
	name     string
	limit    int64
	expire   time.Duration
	replicas int
	force    bool
}

func (g *tGovAddCommand) Setup() (err error) {
	if gov, ok := cmdWithFullCommand("governor"); ok {
		g.cmd = gov.Cmd().Command("add", "Adds or update a Governor")
		g.cmd.Arg("name", "The name for the Governor to managed").Required().StringVar(&g.name)
		g.cmd.Arg("capacity", "How many concurrent lease entries to allow").Required().Int64Var(&g.limit)
		g.cmd.Arg("expire", "Expire entries from the Governor after a period").Required().DurationVar(&g.expire)
		g.cmd.Arg("replicas", "Create a replicated Governor with this many replicas").Default("1").IntVar(&g.replicas)
		g.cmd.Flag("force", "Force operations requiring confirmation").Short('f').UnNegatableBoolVar(&g.force)
	}

	return nil
}

func (g *tGovAddCommand) Configure() (err error) {
	return commonConfigure()
}

func (g *tGovAddCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	gov, _, err := c.NewGovernorManager(ctx, g.name, uint64(g.limit), g.expire, uint(g.replicas), false, nil, governor.WithSubject(c.GovernorSubject(g.name)))
	if err != nil {
		return err
	}

	if gov.MaxAge() != g.expire || gov.Limit() != g.limit || gov.Replicas() != g.replicas {
		fmt.Println("Existing configuration:")
		fmt.Println()
		fmt.Printf("  Capacity: %d desired: %d\n", gov.Limit(), g.limit)
		fmt.Printf("   Expires: %v desired: %v\n", gov.MaxAge(), g.expire)
		fmt.Printf("  Replicas: %d desired: %d\n", gov.Replicas(), g.replicas)

		ans := g.force
		if !g.force {
			fmt.Println()
			err := survey.AskOne(&survey.Confirm{
				Message: fmt.Sprintf("Update configuration for %s?", g.name),
				Default: ans,
			}, &ans)
			if err != nil {
				return err
			}
		}

		if ans {
			err = gov.SetMaxAge(g.expire)
			if err != nil {
				return err
			}

			err = gov.SetLimit(uint64(g.limit))
			if err != nil {
				return err
			}
		}
	}

	fmt.Println("Configuration:")
	fmt.Println()
	fmt.Printf("  Capacity: %d\n", gov.Limit())
	fmt.Printf("   Expires: %v\n", gov.MaxAge())
	fmt.Printf("  Replicas: %d\n", gov.Replicas())
	fmt.Println()

	return nil
}

func init() {
	cli.commands = append(cli.commands, &tGovAddCommand{})
}
