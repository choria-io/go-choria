// Copyright (c) 2018-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"fmt"

	"github.com/choria-io/go-choria/providers/provtarget"
)

var provTargetResolverHost func(provtarget.TargetResolver) error

func init() {
	provTargetResolverHost = provtarget.RegisterTargetResolver
}

func registerProvisionTargetResolverPlugin(name string, plugin Pluggable) error {
	instance, ok := plugin.PluginInstance().(provtarget.TargetResolver)
	if !ok {
		return fmt.Errorf("plugin %s is not a valid ProvisionTargetResolver", plugin.PluginName())
	}

	provTargetResolverHost(instance)

	return nil
}
