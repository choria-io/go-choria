package plugin

import (
	"fmt"

	"github.com/choria-io/go-choria/server"
)

var agentProviderHost func(server.AgentProvider)

func init() {
	agentProviderHost = server.RegisterAdditionalAgentProvider
}

func registerAgentProviderPlugin(name string, plugin Pluggable) error {
	instance, ok := plugin.PluginInstance().(server.AgentProvider)
	if !ok {
		return fmt.Errorf("plugin %s is not a valid AgentProvider", plugin.PluginName())
	}

	agentProviderHost(instance)

	return nil
}
