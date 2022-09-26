// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/choria-io/go-choria/client/discovery"
	scoutcmd "github.com/choria-io/go-choria/scout/cmd"
	"github.com/nats-io/nats.go"
)

type sValidateCommand struct {
	fo          *discovery.StandardOptions
	json        bool
	verbose     bool
	rulesFile   string
	varsFile    string
	rulesOnNode bool
	rulesInKV   bool
	varsOnNode  bool
	varInKV     bool
	display     string
	table       bool
	local       bool

	command
}

func (s *sValidateCommand) Setup() (err error) {
	if scout, ok := cmdWithFullCommand("scout"); ok {
		s.cmd = scout.Cmd().Command("validate", "Execute goss validations")

		s.fo = discovery.NewStandardOptions()
		s.fo.AddFilterFlags(s.cmd)
		s.fo.AddSelectionFlags(s.cmd)

		s.cmd.Arg("rules", "A file holding validation rules to execute").Required().StringVar(&s.rulesFile)
		s.cmd.Arg("variables", "A local file holding template variables to apply").StringVar(&s.varsFile)
		s.cmd.Flag("remote-rules", "Indicates that the rules file path is on the remote nodes").UnNegatableBoolVar(&s.rulesOnNode)
		s.cmd.Flag("kv-rules", "Indicates that the rules is stored in KV in the form Bucket.Key").UnNegatableBoolVar(&s.rulesInKV)
		s.cmd.Flag("remote-variables", "Indicates that the variables file path is on the remote nodes").UnNegatableBoolVar(&s.rulesOnNode)
		s.cmd.Flag("kv-variables", "Indicates that the variables is stored in KV in the form Bucket.Key").UnNegatableBoolVar(&s.rulesInKV)
		s.cmd.Flag("json", "JSON format output").UnNegatableBoolVar(&s.json)
		s.cmd.Flag("table", "Render results in table format").UnNegatableBoolVar(&s.table)
		s.cmd.Flag("verbose", "Show verbose output").Short('v').UnNegatableBoolVar(&s.verbose)
		s.cmd.Flag("display", "Display only a subset of results (ok, failed, all, none)").Default("failed").EnumVar(&s.display, "ok", "failed", "all", "none")
		s.cmd.Flag("local", "Operate locally without connecting to the network").UnNegatableBoolVar(&s.local)
	}

	return nil
}

func (s *sValidateCommand) Configure() error {
	return commonConfigure()
}

func (s *sValidateCommand) parseKv(bk string) (string, string, error) {
	parts := strings.SplitN(bk, ".", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid kv bucket and key")
	}

	return parts[0], parts[1], nil
}

func (s *sValidateCommand) loadKV(bk string) ([]byte, error) {
	b, k, err := s.parseKv(bk)
	if err != nil {
		return nil, err
	}

	bucket, err := c.KV(ctx, nil, b, false)
	if err != nil {
		return nil, err
	}

	v, err := bucket.Get(k)
	if err != nil {
		return nil, err
	}

	if v.Operation() != nats.KeyValuePut {
		return nil, fmt.Errorf("could not find %s > %s", b, k)
	}

	return v.Value(), nil

}

func (s *sValidateCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	s.fo.SetDefaultsFromChoria(c)

	opts := scoutcmd.ValidateCommandOptions{
		Verbose: debug || s.verbose,
		Json:    s.json,
		Display: s.display,
		Table:   s.table,
		Color:   c.Config.Color,
		Local:   s.local,
	}

	switch {
	case s.rulesInKV:
		opts.Rules, err = s.loadKV(s.rulesFile)
		if err != nil {
			return err
		}
	case s.rulesOnNode:
		opts.NodeRulesFile = s.rulesFile
	default:
		opts.Rules, err = os.ReadFile(s.rulesFile)
		if err != nil {
			return err
		}
	}

	if s.varsFile != "" {
		switch {
		case s.varsOnNode:
			opts.NodeVarsFile = s.varsFile
		case s.varInKV:
			opts.Variables, err = s.loadKV(s.varsFile)
			if err != nil {
				return err
			}
		default:
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
