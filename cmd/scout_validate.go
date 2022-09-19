// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"os"
	"sync"

	"github.com/choria-io/go-choria/client/discovery"
	scoutcmd "github.com/choria-io/go-choria/scout/cmd"
)

type sValidateCommand struct {
	fo          *discovery.StandardOptions
	json        bool
	verbose     bool
	rulesFile   string
	varsFile    string
	rulesOnNode bool
	varsOnNode  bool
	all         bool
	table       bool

	command
}

func (s *sValidateCommand) Setup() (err error) {
	if scout, ok := cmdWithFullCommand("scout"); ok {
		s.cmd = scout.Cmd().Command("validate", "Execute goss validations")

		s.fo = discovery.NewStandardOptions()
		s.fo.AddFilterFlags(s.cmd)
		s.fo.AddSelectionFlags(s.cmd)

		s.cmd.Arg("rules", "A file holding validation rules to execute").PlaceHolder("FILE").Required().StringVar(&s.rulesFile)
		s.cmd.Arg("variables", "A local file holding template variables to apply").PlaceHolder("FILE").StringVar(&s.varsFile)
		s.cmd.Flag("remote-rules", "Indicates that the rules file path is on the remote nodes").UnNegatableBoolVar(&s.rulesOnNode)
		s.cmd.Flag("remote-variables", "Indicates that the variables file path is on the remote nodes").UnNegatableBoolVar(&s.rulesOnNode)
		s.cmd.Flag("json", "JSON format output").UnNegatableBoolVar(&s.json)
		s.cmd.Flag("verbose", "Show verbose output").Short('v').UnNegatableBoolVar(&s.verbose)
		s.cmd.Flag("all", "Show all results").UnNegatableBoolVar(&s.all)
		s.cmd.Flag("table", "Render results in table format").UnNegatableBoolVar(&s.table)
	}

	return nil
}

func (s *sValidateCommand) Configure() error {
	return commonConfigure()
}

func (s *sValidateCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	s.fo.SetDefaultsFromChoria(c)

	opts := scoutcmd.ValidateCommandOptions{
		Verbose: debug || s.verbose,
		Json:    s.json,
		ShowAll: s.all,
		Table:   s.table,
		Color:   c.Config.Color,
	}

	if s.rulesOnNode {
		opts.NodeRulesFile = s.rulesFile
	} else {
		opts.Rules, err = os.ReadFile(s.rulesFile)
		if err != nil {
			return err
		}
	}

	if s.varsFile != "" {
		if s.varsOnNode {
			opts.NodeVarsFile = s.varsFile
		} else {
			opts.Variables, err = os.ReadFile(s.varsFile)
			if err != nil {
				return err
			}
		}
	}

	trigger, err := scoutcmd.NewValidateCommand(s.fo, c, &opts, c.Logger("scout"))
	if err != nil {
		return err
	}

	wg.Add(1)
	return trigger.Run(ctx, wg)
}

func init() {
	cli.commands = append(cli.commands, &sValidateCommand{})
}
