package cmd

import (
	"sync"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

type command struct {
	Run   func(wg *sync.WaitGroup) error
	Setup func() error

	cmd *kingpin.CmdClause
}

type runableCmd interface {
	Setup() error
	Run(wg *sync.WaitGroup) error
	FullCommand() string
	Cmd() *kingpin.CmdClause
}

func (c *command) FullCommand() string {
	return c.Cmd().FullCommand()
}

func (c *command) Cmd() *kingpin.CmdClause {
	return c.cmd
}
