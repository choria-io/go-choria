package cmd

import (
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/client/discovery"
	scoutcmd "github.com/choria-io/go-choria/scout/cmd"
)

type sTriggerCommand struct {
	fo      *discovery.StandardOptions
	checks  []string
	json    bool
	verbose bool

	command
}

func (s *sTriggerCommand) Setup() (err error) {
	if scout, ok := cmdWithFullCommand("scout"); ok {
		s.cmd = scout.Cmd().Command("trigger", "Trigger immediate check invocations")

		s.fo = discovery.NewStandardOptions()
		s.fo.AddFilterFlags(s.cmd)
		s.fo.AddSelectionFlags(s.cmd)

		s.cmd.Flag("check", "Trigger only specific checks").StringsVar(&s.checks)
		s.cmd.Flag("json", "JSON format output").BoolVar(&s.json)
		s.cmd.Flag("verbose", "Show verbose output").Short('v').BoolVar(&s.verbose)
	}

	return nil
}

func (s *sTriggerCommand) Configure() error {
	return commonConfigure()
}

func (s *sTriggerCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	s.fo.SetDefaults(cfg.MainCollective, cfg.DefaultDiscoveryMethod, cfg.DiscoveryTimeout)
	trigger, err := scoutcmd.NewTriggerCommand(s.fo, s.checks, s.json, configFile, debug || s.verbose, c.Config.Color, logrus.NewEntry(c.Logger("scout").Logger))
	if err != nil {
		return err
	}

	wg.Add(1)
	return trigger.Run(ctx, wg)
}

func init() {
	cli.commands = append(cli.commands, &sTriggerCommand{})
}
