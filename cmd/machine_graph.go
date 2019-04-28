package cmd

import (
	"fmt"
	"sync"

	"github.com/choria-io/go-choria/aagent/machine"
	"github.com/choria-io/go-choria/aagent/watchers"
)

type mGraphCommand struct {
	command
	souceDir string
}

func (c *mGraphCommand) Setup() (err error) {
	if machine, ok := cmdWithFullCommand("machine"); ok {
		c.cmd = machine.Cmd().Command("graph", "Produce a DOT graph of an autonomous agent definition")
		c.cmd.Arg("source", "Directory containing the machine definition").Required().ExistingDirVar(&c.souceDir)
	}

	return nil
}

func (c *mGraphCommand) Configure() error {
	return nil
}

func (c *mGraphCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	m, err := machine.FromDir(c.souceDir, watchers.New())
	if err != nil {
		return err
	}

	fmt.Print(m.Graph())

	return nil
}

func init() {
	cli.commands = append(cli.commands, &mGraphCommand{})
}
