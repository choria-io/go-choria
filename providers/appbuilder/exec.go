// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package appbuilder

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/choria-io/go-choria/inter"
	"github.com/kballard/go-shellquote"
	"gopkg.in/alecthomas/kingpin.v2"
)

type ExecCommand struct {
	Command string `json:"command"`

	StandardSubCommands
	StandardCommand
}

type Exec struct {
	Arguments map[string]*string
	Flags     map[string]*string
	cmd       *kingpin.CmdClause
	def       *ExecCommand
	cfg       interface{}
	ctx       context.Context
	b         *AppBuilder
}

func NewExecCommand(b *AppBuilder, j json.RawMessage) (*Exec, error) {
	exec := &Exec{
		def:       &ExecCommand{},
		cfg:       b.cfg,
		ctx:       b.ctx,
		b:         b,
		Arguments: map[string]*string{},
		Flags:     map[string]*string{},
	}

	err := json.Unmarshal(j, exec.def)
	if err != nil {
		return nil, err
	}

	return exec, nil
}

func (r *Exec) SubCommands() []json.RawMessage {
	return r.def.Commands
}

func (r *Exec) CreateCommand(app inter.FlagApp) (*kingpin.CmdClause, error) {
	r.cmd = createStandardCommand(app, r.b, &r.def.StandardCommand, r.Arguments, r.Flags, r.runCommand)

	return r.cmd, nil
}

func (r *Exec) runCommand(_ *kingpin.ParseContext) error {
	cmd, err := parseStateTemplate(r.def.Command, r.Arguments, r.Flags, r.cfg)
	if err != nil {
		return err
	}

	parts, err := shellquote.Split(cmd)
	if err != nil {
		return err
	}
	if len(parts) == 0 {
		return fmt.Errorf("invalid command")
	}

	run := exec.CommandContext(r.ctx, parts[0], parts[1:]...)
	run.Env = os.Environ()
	run.Stdin = os.Stdin
	run.Stdout = os.Stdout
	run.Stderr = os.Stderr

	err = run.Run()
	if err != nil {
		return err
	}

	return nil
}
