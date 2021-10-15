// Copyright (c) 2018-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"fmt"

	"github.com/choria-io/go-choria/config"
)

func registerConfigMutator(name string, plugin Pluggable) error {
	mutator, ok := plugin.PluginInstance().(config.Mutator)
	if !ok {
		return fmt.Errorf("%s is not a valid configuration mutator plugin", plugin.PluginName())
	}

	config.RegisterMutator(plugin.PluginName(), mutator)

	return nil
}
