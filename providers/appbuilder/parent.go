// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package appbuilder

import (
	"encoding/json"

	"github.com/choria-io/go-choria/inter"
	"gopkg.in/alecthomas/kingpin.v2"
)

type ParentCommand struct {
	StandardSubCommands
	StandardCommand
}

type Parent struct {
	cmd *kingpin.CmdClause
	def *ParentCommand
}

func NewParentCommand(j json.RawMessage, _ interface{}) (*Parent, error) {
	parent := &Parent{
		def: &ParentCommand{},
	}

	err := json.Unmarshal(j, parent.def)
	if err != nil {
		return nil, err
	}

	return parent, nil
}

func (p *Parent) SubCommands() []json.RawMessage {
	return p.def.Commands
}

func (p *Parent) CreateCommand(app inter.FlagApp) (*kingpin.CmdClause, error) {
	p.cmd = app.Command(p.def.Name, p.def.Description)
	for _, a := range p.def.Aliases {
		p.cmd.Alias(a)
	}

	return p.cmd, nil
}
