// Copyright (c) 2020-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package mcorpc

import (
	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/server"
	"github.com/choria-io/go-choria/server/agents"
)

// AgentPlugin is a choria plugin
type AgentPlugin struct {
	metadata *agents.Metadata
	creator  func(mgr server.AgentManager) (agents.Agent, error)
}

// NewChoriaAgentPlugin creates a new plugin for an agent that allows it to plug into the Choria Plugin system
func NewChoriaAgentPlugin(metadata *agents.Metadata, creator func(mgr server.AgentManager) (agents.Agent, error)) inter.Pluggable {
	plugin := &AgentPlugin{
		metadata: metadata,
		creator:  creator,
	}

	return plugin
}

// PluginInstance implements plugin.Pluggable
func (p *AgentPlugin) PluginInstance() any {
	return p.creator
}

// PluginVersion implements plugin.Pluggable
func (p *AgentPlugin) PluginVersion() string {
	return p.metadata.Version
}

// PluginName implements plugin.Pluggable
func (p *AgentPlugin) PluginName() string {
	return p.metadata.Description
}

// PluginType implements plugin.Pluggable
func (p *AgentPlugin) PluginType() inter.PluginType {
	return inter.AgentPlugin
}
