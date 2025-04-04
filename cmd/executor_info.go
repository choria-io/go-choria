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

type executorInfoCommand struct {
	command

	fo *discovery.StandardOptions

	id   string
	json bool
}

func init() {
	cli.commands = append(cli.commands, &executorInfoCommand{})
}

func (n *executorInfoCommand) Setup() error {
	if exec, ok := cmdWithFullCommand("executor"); ok {
		n.cmd = exec.Cmd().Command("info", "Retrieves Job Information")
		n.cmd.Arg("id", "The Job ID to retrieves information for").Required().StringVar(&n.id)
		n.cmd.Flag("json", "Renders result in JSON format").UnNegatableBoolVar(&n.json)

		n.fo = discovery.NewStandardOptions()
		n.fo.AddFilterFlags(n.cmd)
		n.fo.AddFlatFileFlags(n.cmd)
		n.fo.AddSelectionFlags(n.cmd)
	}

	return nil
}

func (n *executorInfoCommand) Configure() (err error) {
	return commonConfigure()
}

func (n *executorInfoCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	n.fo.SetDefaultsFromChoria(c)
	log := c.Logger("executor")

	opts := []executorclient.InitializationOption{
		executorclient.Logger(log), executorclient.Discovery(executorclient.NewMetaNS(n.fo, true)),
	}
	if !n.json {
		opts = append(opts, executorclient.Progress())
	}

	ec, err := executorclient.New(c, opts...)
	if err != nil {
		return err
	}

	res, err := ec.Status(n.id).Do(ctx)
	if err != nil {
		return err
	}

	format := executorclient.TextFormat
	if n.json {
		format = executorclient.JSONFormat
	}
	return res.RenderResults(os.Stdout, format, executorclient.DisplayDDL, debug, false, true, log)
}
