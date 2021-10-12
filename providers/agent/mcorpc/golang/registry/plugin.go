// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"github.com/choria-io/go-choria/plugin"
	"github.com/choria-io/go-choria/providers/agent/mcorpc"
)

// ChoriaPlugin creates the choria plugin hooks
func ChoriaPlugin() plugin.Pluggable {
	return mcorpc.NewChoriaAgentPlugin(metadata, New)
}
