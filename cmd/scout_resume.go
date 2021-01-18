package cmd

import (
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/client/discovery"
	scoutcmd "github.com/choria-io/go-choria/scout/cmd"
)

type sResumeCommand struct {
	fo      *discovery.StandardOptions
	checks  []string
	json    bool
	verbose bool

	command
}

func (s *sResumeCommand) Setup() (err error) {
	if scout, ok := cmdWithFullCommand("scout"); ok {
		s.cmd = scout.Cmd().Command("resume", "Resume normal checks of checks in maintenance mode")

		s.fo = discovery.NewStandardOptions()
		s.fo.AddFilterFlags(s.cmd)
		s.fo.AddSelectionFlags(s.cmd)

		s.cmd.Flag("check", "Affect only specific checks").StringsVar(&s.checks)
		s.cmd.Flag("json", "JSON format output").BoolVar(&s.json)
		s.cmd.Flag("verbose", "Show verbose output").Short('v').BoolVar(&s.verbose)
	}

	return nil
}

func (s *sResumeCommand) Configure() error {
	return commonConfigure()
}

func (s *sResumeCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	s.fo.SetDefaultsFromChoria(c)
	trigger, err := scoutcmd.NewResumeCommand(s.fo, s.checks, s.json, configFile, debug || s.verbose, c.Config.Color, logrus.NewEntry(c.Logger("scout").Logger))
	if err != nil {
		return err
	}

	wg.Add(1)
	return trigger.Run(ctx, wg)
}

func init() {
	cli.commands = append(cli.commands, &sResumeCommand{})
}
