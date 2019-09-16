package ruby

import (
	"github.com/choria-io/go-choria/plugin"
)

// ChoriaPlugin produces the plugin for choria
func ChoriaPlugin() plugin.Pluggable {
	return &Provider{}
}

// PluginInstance implements plugin.Pluggable
func (p *Provider) PluginInstance() interface{} {
	return p
}

// PluginVersion implements plugin.Pluggable
func (p *Provider) PluginVersion() string {
	return "0.8.0"
}

// PluginName implements plugin.Pluggable
func (p *Provider) PluginName() string {
	return "Ruby MCollective Agent Compatibility"
}

// PluginType implements plugin.Pluggable
func (p *Provider) PluginType() plugin.Type {
	return plugin.AgentProviderPlugin
}
