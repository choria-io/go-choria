package cmd

import (
	"sync"
)

type toolCommand struct {
	command
}

func (t *toolCommand) Setup() (err error) {
	t.cmd = cli.app.Command("tool", "Various utilities for debugging and verification of Choria Networks")

	return nil
}

func (t *toolCommand) Configure() error {
	return nil
}

func (t *toolCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	return nil
}

func init() {
	cli.commands = append(cli.commands, &toolCommand{})
}
