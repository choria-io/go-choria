// Copyright (c) 2021-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"fmt"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/providers/data"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type DataPlugin struct {
	Name    string
	Creator *data.Creator
}

func NewDataPlugin(name string, creator func(fw data.Framework) (data.Plugin, error)) *DataPlugin {
	return &DataPlugin{Name: name, Creator: &data.Creator{Name: name, F: creator}}
}

// PluginInstance implements plugin.Pluggable
func (p *DataPlugin) PluginInstance() any {
	return p.Creator
}

// PluginVersion implements plugin.Pluggable
func (p *DataPlugin) PluginVersion() string {
	return build.Version
}

// PluginName implements plugin.Pluggable
func (p *DataPlugin) PluginName() string {
	return fmt.Sprintf("%s Data version %s", cases.Title(language.AmericanEnglish).String(p.Name), build.Version)
}

// PluginType implements plugin.Pluggable
func (p *DataPlugin) PluginType() inter.PluginType {
	return inter.DataPlugin
}
