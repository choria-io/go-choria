package cmd

import "sync"

type kvCommand struct {
	command
}

func (k *kvCommand) Setup() (err error) {
	k.cmd = cli.app.Command("kv", "Key-Value store for Choria Streams")

	return nil
}

func (k *kvCommand) Configure() error {
	return nil
}

func (k *kvCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	return nil
}

func init() {
	cli.commands = append(cli.commands, &kvCommand{})
}
