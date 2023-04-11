// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package machines_manager

import (
	_ "embed"

	"github.com/choria-io/go-choria/aagent/machine"
	"github.com/choria-io/go-choria/aagent/plugin"
	"github.com/ghodss/yaml"
)

var (
	//go:embed machine.yaml
	mdat []byte
)

func ChoriaPlugin() *plugin.MachinePlugin {
	m := &machine.Machine{}
	err := yaml.Unmarshal(mdat, m)
	if err != nil {
		panic(err)
	}

	return plugin.NewMachinePlugin("plugins_manager", m)
}
