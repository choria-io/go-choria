// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package aagent

import (
	"fmt"
	"sync"

	"github.com/choria-io/go-choria/aagent/model"
	"github.com/choria-io/go-choria/internal/util"
)

var (
	mu      sync.Mutex
	plugins map[string]model.MachineConstructor
)

// RegisterMachinePlugin registers a new compile time constructed machine
func RegisterMachinePlugin(plugin model.MachineConstructor) error {
	mu.Lock()
	defer mu.Unlock()

	if plugins == nil {
		plugins = make(map[string]model.MachineConstructor)
	}

	_, exist := plugins[plugin.Name()]
	if exist {
		return fmt.Errorf("plugin %q already exist", plugin.Name())
	}

	plugins[plugin.Name()] = plugin

	util.BuildInfo().RegisterMachine(plugin.PluginName())

	return nil
}
