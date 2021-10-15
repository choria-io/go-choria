// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"fmt"

	"github.com/choria-io/go-choria/aagent"
	"github.com/choria-io/go-choria/aagent/model"
)

func registerMachinePlugin(plugin Pluggable) error {
	instance, ok := plugin.PluginInstance().(model.MachineConstructor)
	if !ok {
		return fmt.Errorf("%s is not a valid machine plugin", plugin.PluginName())
	}

	return aagent.RegisterMachinePlugin(instance)
}
