// Copyright (c) 2021-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package inter

// PluginType are types of choria plugin
type PluginType int

const (
	// UnknownPlugin is a unknown plugin type
	UnknownPlugin PluginType = iota

	// AgentProviderPlugin is a plugin that provide types of agents to Choria
	AgentProviderPlugin

	// AgentPlugin is a type of agent
	AgentPlugin

	// ProvisionTargetResolverPlugin is a plugin that helps provisioning mode Choria find its broker
	ProvisionTargetResolverPlugin

	// ConfigMutatorPlugin is a plugin that can dynamically adjust
	// configuration based on local site conditions
	ConfigMutatorPlugin

	// MachineWatcherPlugin is a plugin that adds a Autonomous Agent Watcher
	MachineWatcherPlugin

	// DataPlugin is a plugin that provides data to choria
	DataPlugin

	// MachinePlugin is an autonomous agent
	MachinePlugin
)

// Pluggable is a Choria Plugin
type Pluggable interface {
	// PluginInstance is any structure that implements the plugin, should be right type for the kind of plugin
	PluginInstance() any

	// PluginName is a human friendly name for the plugin
	PluginName() string

	// PluginType is the type of the plugin, to match plugin.Type
	PluginType() PluginType

	// PluginVersion is the version of the plugin
	PluginVersion() string
}
