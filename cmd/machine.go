package cmd

import "sync"

type machineCommand struct {
	command
}

func (m *machineCommand) Setup() (err error) {
	m.cmd = cli.app.Command("machine", "Manages autonomous agents")

	return nil
}

func (m *machineCommand) Configure() error {
	return nil
}

func (m *machineCommand) Run(wg *sync.WaitGroup) (err error) {
	return nil
}

func init() {
	cli.commands = append(cli.commands, &machineCommand{})
}
