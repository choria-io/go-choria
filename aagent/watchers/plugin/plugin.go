// Copyright (c) 2020-2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"encoding/json"
	"fmt"
	"github.com/choria-io/go-choria/aagent/watchers"
	"time"

	"github.com/choria-io/go-choria/aagent/model"
	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/inter"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func NewWatcherPlugin(wtype string, version string, notification func() any, new func(machine model.Machine, name string, states []string, requiredStates []watchers.ForeignMachineState, failEvent string, successEvent string, interval string, ai time.Duration, properties map[string]any) (any, error)) *WatcherPlugin {
	return &WatcherPlugin{
		Name: wtype,
		Creator: &watcherCreator{
			wtype:        wtype,
			version:      version,
			notification: notification,
			new:          new,
		},
	}
}

type WatcherPlugin struct {
	Name    string
	Creator any
}

type watcherCreator struct {
	wtype        string
	version      string
	notification func() any
	new          func(machine model.Machine, name string, states []string, requiredStates []watchers.ForeignMachineState, failEvent string, successEvent string, interval string, ai time.Duration, properties map[string]any) (any, error)
}

func (c *watcherCreator) Type() string {
	return c.wtype
}

func (c *watcherCreator) EventType() string {
	return fmt.Sprintf("io.choria.machine.watcher.%s.%s.state", c.wtype, c.version)
}

func (c *watcherCreator) UnmarshalNotification(n []byte) (any, error) {
	state := c.notification()
	err := json.Unmarshal(n, state)

	return state, err
}

func (c *watcherCreator) New(machine model.Machine, name string, states []string, requiredStates []watchers.ForeignMachineState, failEvent string, successEvent string, interval string, ai time.Duration, properties map[string]any) (any, error) {
	return c.new(machine, name, states, requiredStates, failEvent, successEvent, interval, ai, properties)
}

// PluginInstance implements plugin.Pluggable
func (p *WatcherPlugin) PluginInstance() any {
	return p.Creator
}

// PluginVersion implements plugin.Pluggable
func (p *WatcherPlugin) PluginVersion() string {
	return build.Version
}

// PluginName implements plugin.Pluggable
func (p *WatcherPlugin) PluginName() string {
	return fmt.Sprintf("%s Watcher version %s", cases.Title(language.AmericanEnglish).String(p.Name), build.Version)
}

// PluginType implements plugin.Pluggable
func (p *WatcherPlugin) PluginType() inter.PluginType {
	return inter.MachineWatcherPlugin
}
