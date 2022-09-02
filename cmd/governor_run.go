// Copyright (c) 2021-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/lifecycle"
	"github.com/kballard/go-shellquote"
	"github.com/nats-io/jsm.go"
	"github.com/nats-io/jsm.go/governor" //lint:ignore SA1019 Will vendor
)

type tGovRunCommand struct {
	command
	name     string
	fullCmd  []string
	maxWait  time.Duration
	interval time.Duration
	noLEave  bool
}

func (g *tGovRunCommand) Setup() (err error) {
	if gov, ok := cmdWithFullCommand("governor"); ok {
		g.cmd = gov.Cmd().Command("run", "Runs a command subject to Governor control")
		g.cmd.Arg("name", "The name for the Governor").Required().StringVar(&g.name)
		g.cmd.Arg("command", "Command to execute").Required().StringsVar(&g.fullCmd)
		g.cmd.Flag("max-wait", "Maximum amount of time to wait to obtain a lease").Default("5m").DurationVar(&g.maxWait)
		g.cmd.Flag("interval", "Interval for attempting to get a lease").Default("5s").DurationVar(&g.interval)
		g.cmd.Flag("max-per-period", "Instead of limiting concurrent runs, limit runs per governor period").UnNegatableBoolVar(&g.noLEave)
	}

	return nil
}

func (g *tGovRunCommand) Configure() (err error) {
	// we pretend to be a server so that we use the right certs and identity etc
	return systemConfigureIfRoot(true)
}

func (g *tGovRunCommand) trySendEvent(et lifecycle.GovernorEventType, seq uint64, conn inter.RawNATSConnector) {
	event, err := lifecycle.New(lifecycle.Governor, lifecycle.Component("CLI"), lifecycle.Identity(c.Config.Identity), lifecycle.GovernorType(et), lifecycle.GovernorName(g.name), lifecycle.GovernorSequence(seq))
	if err == nil {
		lifecycle.PublishEvent(event, conn)
	}
}

func (g *tGovRunCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	if g.interval < time.Second {
		return fmt.Errorf("interval should be >=1s")
	}

	ctx, cancel := context.WithTimeout(ctx, g.maxWait)
	defer cancel()

	log := c.Logger("governor").WithField("name", g.name)
	conn, err := c.NewConnector(ctx, c.MiddlewareServers, fmt.Sprintf("governor manager: %s", g.name), log)
	if err != nil {
		return err
	}

	mgr, err := jsm.New(conn.Nats())
	if err != nil {
		return err
	}

	parts, err := shellquote.Split(strings.Join(g.fullCmd, " "))
	if err != nil {
		return fmt.Errorf("can not parse command: %s", err)
	}
	var cmd string
	var args []string

	switch {
	case len(parts) == 0:
		return fmt.Errorf("could not parse command")
	case len(parts) == 1:
		cmd = parts[0]
	default:
		cmd = parts[0]
		args = append(args, parts[1:]...)
	}

	opts := []governor.Option{
		governor.WithSubject(c.GovernorSubject(g.name)),
		governor.WithInterval(g.interval),
		governor.WithLogger(log),
	}

	if g.noLEave {
		opts = append(opts, governor.WithoutLeavingOnCompletion())
	}

	gov := governor.NewJSGovernor(g.name, mgr, opts...)
	finisher, seq, err := gov.Start(ctx, cfg.Identity)
	if err != nil {
		if g.noLEave && err == context.DeadlineExceeded {
			return nil
		}

		g.trySendEvent(lifecycle.GovernorTimeoutEvent, 0, conn)
		return fmt.Errorf("could not get a execution slot: %s", err)
	}

	g.trySendEvent(lifecycle.GovernorEnterEvent, seq, conn)

	finish := func(wg *sync.WaitGroup) {
		defer wg.Done()

		finisher()
		g.trySendEvent(lifecycle.GovernorExitEvent, seq, conn)
		conn.Close()
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	go func() {
		<-sigs
		wg.Add(1)
		finish(wg)
	}()

	osExit := func(c int, format string, a ...any) {
		if format != "" {
			fmt.Println(fmt.Sprintf(format, a...))
		}

		wg.Add(1)
		finish(wg)

		os.Exit(c)
	}

	execution := exec.Command(cmd, args...)
	execution.Stdin = os.Stdin
	execution.Stdout = os.Stdout
	execution.Stderr = os.Stderr
	err = execution.Run()
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if ok {
			osExit(exitErr.ExitCode(), "")
		} else {
			osExit(1, "execution failed: %s", err)
		}
	}

	if execution.ProcessState == nil {
		osExit(1, "Unknown execution state")
	}

	osExit(execution.ProcessState.ExitCode(), "")

	return nil

}

func init() {
	cli.commands = append(cli.commands, &tGovRunCommand{})
}
