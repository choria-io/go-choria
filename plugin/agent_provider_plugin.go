// Copyright (c) 2018-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

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
