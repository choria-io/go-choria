// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package gossipwatcher

import (
	"github.com/choria-io/go-choria/aagent/watchers/plugin"
)

func ChoriaPlugin() *plugin.WatcherPlugin {
	return plugin.NewWatcherPlugin(wtype, version, func() interface{} { return &StateNotification{} }, New)
}
