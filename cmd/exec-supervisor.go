// Copyright (c) 2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"sync"
	"time"

	"github.com/choria-io/go-choria/providers/execution"
	"github.com/choria-io/go-choria/submission"
)

type execSupervisorCommand struct {
	command

	hb     time.Duration
	output bool
	cmdID  string
	env    map[string]string
}

func init() {
	cli.commands = append(cli.commands, &execSupervisorCommand{})
}

func (b *execSupervisorCommand) Setup() (err error) {
	b.env = map[string]string{}

	b.cmd = cli.app.Command("exec-supervisor", "Executes and supervises shell commands").Hidden()
	b.cmd.Flag("config", "Config file to use").PlaceHolder("FILE").ExistingFileVar(&configFile)
	b.cmd.Flag("heartbeat", "Interval to heartbeat about running commands").Default("5m").DurationVar(&b.hb)
	b.cmd.Flag("track-output", "Tracks command output and Submit to Choria").UnNegatableBoolVar(&b.output)
	b.cmd.Flag("process", "Unique ID for this command").Required().StringVar(&b.cmdID)

	return
}

func (b *execSupervisorCommand) Configure() (err error) {
	return commonConfigure()
}

func (b *execSupervisorCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	log := c.Logger("exec-supervisor")
	proc, err := execution.LoadWithChoria(c, b.cmdID)
	if err != nil {
		log.Errorf("Could not start supervisor: %s", err)
		return fmt.Errorf("could not start supervisor: %s", err)
	}

	submit, err := submission.NewFromChoria(c, submission.Directory)
	if err != nil {
		log.Errorf("Could not start supervisor: %s", err)
		return fmt.Errorf("could not start supervisor: %s", err)
	}

	err = proc.StartSupervised(ctx, cfg.Choria.ExecutorSpool, submit, b.hb, b.output, log)
	if err != nil {
		log.Errorf("Could not start supervisor: %s", err)
		return fmt.Errorf("could not start supervisor: %s", err)
	}

	return nil
}
