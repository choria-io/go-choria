// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/choria-io/appbuilder/commands/exec"
	"github.com/choria-io/appbuilder/commands/parent"
	"github.com/choria-io/go-choria/providers/appbuilder/discover"
	"github.com/choria-io/go-choria/providers/appbuilder/kv"
	"github.com/choria-io/go-choria/providers/appbuilder/rpc"
	"sync"

	"github.com/choria-io/appbuilder/builder"
)

type tBuilderCommand struct {
	command
}

func (c *tBuilderCommand) Setup() error {
	kv.MustRegister()
	rpc.MustRegister()
	discover.MustRegister()
	parent.MustRegister()
	exec.MustRegister()

	if tool, ok := cmdWithFullCommand("tool"); ok {
		c.cmd = tool.Cmd().Command("builder", "Application Builder tools")
		bldr, err := builder.New(ctx, "builder", builderOptions()...)
		if err != nil {
			return err
		}

		// builder cli is Action() based
		ran = true

		bldr.CreateBuilderApp(c.cmd)
	}

	return nil
}

func (c *tBuilderCommand) Configure() error {
	return commonConfigure()
}

func (c *tBuilderCommand) Run(wg *sync.WaitGroup) error {
	wg.Done()
	return nil
}

func init() {
	cli.commands = append(cli.commands, &tBuilderCommand{})
}
