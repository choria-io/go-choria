package cmd

import (
	"sync"

	scoutcmd "github.com/choria-io/go-choria/scout/cmd"
	"github.com/sirupsen/logrus"
)

type sChecksCommand struct {
	identity string
	json     bool
	verbose  bool

	command
}

func (w *sChecksCommand) Setup() (err error) {
	if scout, ok := cmdWithFullCommand("scout"); ok {
		w.cmd = scout.Cmd().Command("checks", "Retrieve check statuses from an agent")
		w.cmd.Arg("identity", "Node to retrieve data from").Required().StringVar(&w.identity)
		w.cmd.Flag("json", "JSON format output").BoolVar(&w.json)
		w.cmd.Flag("verbose", "Show verbose output").Short('v').BoolVar(&w.verbose)
	}

	return nil
}

func (w *sChecksCommand) Configure() error {
	return commonConfigure()
}

func (w *sChecksCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	check, err := scoutcmd.NewChecksCommand(w.identity, w.json, debug || w.verbose, configFile, logrus.NewEntry(c.Logger("scout").Logger))
	if err != nil {
		return err
	}

	wg.Add(1)
	return check.Run(ctx, wg)
}

func init() {
	cli.commands = append(cli.commands, &sChecksCommand{})
}
