// Copyright (c) 2021-2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package machines_manager

import (
	_ "embed"

	"github.com/goccy/go-yaml"

	"github.com/choria-io/go-choria/aagent/machine"
	"github.com/choria-io/go-choria/aagent/plugin"
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

	return plugin.NewMachinePlugin("machines_manager", m)
}
