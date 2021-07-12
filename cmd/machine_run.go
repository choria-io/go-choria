package cmd

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/aagent/notifiers/console"

	"github.com/choria-io/go-choria/aagent/machine"
	"github.com/choria-io/go-choria/aagent/watchers"
)

type mRunCommand struct {
	command
	sourceDir string
	factsFile string
}

func (c *mRunCommand) Setup() (err error) {
	if machine, ok := cmdWithFullCommand("machine"); ok {
		c.cmd = machine.Cmd().Command("run", "Runs an autonomous agent locally")
		c.cmd.Arg("source", "Directory containing the machine definition").Required().ExistingDirVar(&c.sourceDir)
		c.cmd.Arg("facts", "Facts file to supply to the machine as run time facts").ExistingFileVar(&c.factsFile)
	}

	return nil
}

func (c *mRunCommand) Configure() error {
	return commonConfigure()
}

func (c *mRunCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	if debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	m, err := machine.FromDir(c.sourceDir, watchers.New(ctx))
	if err != nil {
		return err
	}

	m.SetIdentity("cli")
	m.RegisterNotifier(&console.Notifier{})
	m.SetMainCollective(cfg.MainCollective)
	if c.factsFile != "" {
		facts, err := os.ReadFile(c.factsFile)
		if err != nil {
			return err
		}
		m.SetFactSource(func() json.RawMessage { return facts })
	}

	<-m.Start(ctx, wg)

	<-ctx.Done()

	return nil
}

func init() {
	cli.commands = append(cli.commands, &mRunCommand{})
}
