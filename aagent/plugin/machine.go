// Copyright (c) 2021-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"fmt"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/inter"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func NewMachinePlugin(name string, machine any) *MachinePlugin {
	return &MachinePlugin{name: name, machine: machine}
}

type MachinePlugin struct {
	name    string
	machine any
}

func (p *MachinePlugin) Name() string {
	return p.name
}

func (p *MachinePlugin) Machine() any {
	return p.machine
}

func (p *MachinePlugin) PluginInstance() any {
	return p
}

func (p *MachinePlugin) PluginVersion() string {
	return build.Version
}

func (p *MachinePlugin) PluginName() string {
	return fmt.Sprintf("%s Autonomous Agent version %s", cases.Title(language.AmericanEnglish).String(p.name), build.Version)
}

func (p *MachinePlugin) PluginType() inter.PluginType {
	return inter.MachinePlugin
}
