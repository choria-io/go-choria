// Copyright (c) 2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/choria-io/go-choria/client/discovery"
	"github.com/choria-io/go-choria/client/executorclient"
	"os"
	"sync"
)

type executorSignalCommand struct {
	command

	fo *discovery.StandardOptions

	id   string
	sig  uint
	json bool
}

func init() {
	cli.commands = append(cli.commands, &executorSignalCommand{})
}

func (s *executorSignalCommand) Setup() error {
	if exec, ok := cmdWithFullCommand("executor"); ok {
		s.cmd = exec.Cmd().Command("signal", "Retrieves Job Information").Alias("sig").Alias("kill")
		s.cmd.Arg("signal", "The signal to send").Required().UintVar(&s.sig)
		s.cmd.Arg("id", "The Job ID to signal").Required().StringVar(&s.id)
		s.cmd.Flag("json", "Renders result in JSON format").UnNegatableBoolVar(&s.json)

		s.fo = discovery.NewStandardOptions()
		s.fo.AddFilterFlags(s.cmd)
		s.fo.AddFlatFileFlags(s.cmd)
		s.fo.AddSelectionFlags(s.cmd)
	}

	return nil
}

func (s *executorSignalCommand) Configure() (err error) {
	return commonConfigure()
}

func (s *executorSignalCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	s.fo.SetDefaultsFromChoria(c)
	log := c.Logger("executor")

	opts := []executorclient.InitializationOption{
		executorclient.Logger(log), executorclient.Discovery(executorclient.NewMetaNS(s.fo, true)),
	}
	if !s.json {
		opts = append(opts, executorclient.Progress())
	}

	ec, err := executorclient.New(c, opts...)
	if err != nil {
		return err
	}

	res, err := ec.Signal(s.id, int64(s.sig)).Do(ctx)
	if err != nil {
		return err
	}

	format := executorclient.TextFormat
	if s.json {
		format = executorclient.JSONFormat
	}
	return res.RenderResults(os.Stdout, format, executorclient.DisplayDDL, debug, false, true, log)
}
