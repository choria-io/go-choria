package cmd

import (
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

type command struct {
	Run   func() error
	Setup func() error

	cmd *kingpin.CmdClause
}

type runableCmd interface {
	Setup() error
	Run() error
	FullCommand() string
	Cmd() *kingpin.CmdClause
}

func (c *command) FullCommand() string {
	return c.Cmd().FullCommand()
}

func (c *command) Cmd() *kingpin.CmdClause {
	return c.cmd
}
