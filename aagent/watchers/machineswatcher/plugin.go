// Copyright (c) 2021-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package machines

import (
	"github.com/choria-io/go-choria/aagent/watchers/plugin"
)

func ChoriaPlugin() *plugin.WatcherPlugin {
	return plugin.NewWatcherPlugin(wtype, version, func() any { return &StateNotification{} }, New)
}
