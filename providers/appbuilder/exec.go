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
}

func NewExecCommand(ctx context.Context, j json.RawMessage, cfg interface{}) (*Exec, error) {
	exec := &Exec{
		def: &ExecCommand{},
		cfg: cfg,
		ctx: ctx,
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
	r.cmd = app.Command(r.def.Name, r.def.Description).Action(r.runCommand)
	for _, a := range r.def.Aliases {
		r.cmd.Alias(a)
	}

	for _, a := range r.def.Arguments {
		arg := r.cmd.Arg(a.Name, a.Description)
		if a.Required {
			arg.Required()
		}

		r.Arguments[a.Name] = arg.String()
	}

	for _, f := range r.def.Flags {
		flag := r.cmd.Flag(f.Name, f.Description)
		if f.Required {
			flag.Required()
		}
		if f.PlaceHolder != "" {
			flag.PlaceHolder(f.PlaceHolder)
		}
		r.Flags[f.Name] = flag.String()
	}

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
