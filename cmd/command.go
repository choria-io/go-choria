// Copyright (c) 2017-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"sync"

	"github.com/choria-io/fisk"
)

type command struct {
	Run   func(wg *sync.WaitGroup) error
	Setup func() error

	cmd *fisk.CmdClause
}

type runableCmd interface {
	Setup() error
	Run(wg *sync.WaitGroup) error
	FullCommand() string
	Cmd() *fisk.CmdClause
	Configure() error
}

func (c *command) FullCommand() string {
	return c.Cmd().FullCommand()
}

func (c *command) Cmd() *fisk.CmdClause {
	return c.cmd
}
