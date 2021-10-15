// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"fmt"

	"github.com/choria-io/go-choria/aagent/model"
	"github.com/choria-io/go-choria/aagent/watchers"
)

func registerWatcherPlugin(_ string, plugin Pluggable) error {
	instance, ok := plugin.PluginInstance().(model.WatcherConstructor)
	if !ok {
		return fmt.Errorf("%s is not a valid watcher plugin", plugin.PluginName())
	}

	return watchers.RegisterWatcherPlugin(plugin.PluginName(), instance)
}
