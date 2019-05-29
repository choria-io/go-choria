package cmd

import (
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/aagent/notifiers/console"

	"github.com/choria-io/go-choria/aagent/machine"
	"github.com/choria-io/go-choria/aagent/watchers"
)

type mRunCommand struct {
	command
	sourceDir string
}

func (c *mRunCommand) Setup() (err error) {
	if machine, ok := cmdWithFullCommand("machine"); ok {
		c.cmd = machine.Cmd().Command("run", "Runs an autonomous agent locally")
		c.cmd.Arg("source", "Directory containing the machine definition").Required().ExistingDirVar(&c.sourceDir)
	}

	return nil
}

func (c *mRunCommand) Configure() error {
	return nil
}

func (c *mRunCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	if debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	m, err := machine.FromDir(c.sourceDir, watchers.New())
	if err != nil {
		return err
	}

	m.SetIdentity("cli")
	m.RegisterNotifier(&console.Notifier{})

	<-m.Start(ctx, wg)

	return nil
}

func init() {
	cli.commands = append(cli.commands, &mRunCommand{})
}
