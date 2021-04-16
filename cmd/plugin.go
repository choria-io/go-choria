package cmd

import (
	"sync"
)

type pluginCommand struct {
	command
}

func (t *pluginCommand) Setup() (err error) {
	t.cmd = cli.app.Command("plugin", "Plugin inspection and generation")

	return nil
}

func (t *pluginCommand) Configure() error {
	return nil
}

func (t *pluginCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	return nil
}

func init() {
	cli.commands = append(cli.commands, &pluginCommand{})
}
