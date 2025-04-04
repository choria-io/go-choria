// Copyright (c) 2020-2024, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"sync"
	"time"

	scoutcmd "github.com/choria-io/go-choria/scout/cmd"
)

type sWatchCommand struct {
	identity            string
	check               string
	perf                bool
	ok                  bool
	history             time.Duration
	watch               *scoutcmd.WatchCommand
	ignoreMachineEvents []string

	command
}

func (w *sWatchCommand) Setup() (err error) {
	if scout, ok := cmdWithFullCommand("scout"); ok {
		w.cmd = scout.Cmd().Command("watch", "Watch CloudEvents produced by Scout")
		w.cmd.Flag("identity", "Filters events by identity").StringVar(&w.identity)
		w.cmd.Flag("check", "Filters events by check").StringVar(&w.check)
		w.cmd.Flag("perf", "Show performance data").UnNegatableBoolVar(&w.perf)
		w.cmd.Flag("history", "Retrieve a certain period of history from Choria Streaming Server").DurationVar(&w.history)
		w.cmd.Flag("ok", "Include OK status updates").Default("true").BoolVar(&w.ok)
		w.cmd.Flag("ignore-machine-transition", "Ignore transitions from certain machines").PlaceHolder("MACHINE").StringsVar(&w.ignoreMachineEvents)
	}

	return nil
}

func (w *sWatchCommand) Configure() error {
	return commonConfigure()
}

func (w *sWatchCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	conn, err := c.NewConnector(ctx, c.MiddlewareServers, c.Certname(), c.Logger("scout"))
	if err != nil {
		return fmt.Errorf("cannot connect: %s", err)
	}

	w.watch, err = scoutcmd.NewWatchCommand(w.identity, w.check, w.ignoreMachineEvents, w.perf, !w.ok, w.history, conn, c.Logger("scout"))
	if err != nil {
		return err
	}

	wg.Add(1)
	return w.watch.Run(ctx, wg)
}

func init() {
	cli.commands = append(cli.commands, &sWatchCommand{})
}
