// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package machinedata

import (
	"context"
	"fmt"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/ddl/common"
	"github.com/choria-io/go-choria/providers/data"
	"github.com/choria-io/go-choria/providers/data/ddl"
	"github.com/choria-io/go-choria/providers/data/plugin"
	"github.com/choria-io/go-choria/server/agents"
)

type MachineData struct{}

func ChoriaPlugin() *plugin.DataPlugin {
	return plugin.NewDataPlugin("machine_state", New)
}

func New(_ data.Framework) (data.Plugin, error) {
	return &MachineData{}, nil
}

func (s *MachineData) Run(_ context.Context, q data.Query, si agents.ServerInfoSource) (map[string]data.OutputItem, error) {
	machines, err := si.MachinesStatus()
	if err != nil {
		return nil, err
	}

	query := q.(string)
	for _, m := range machines {
		if m.ID == query || m.Path == query || m.Name == query {
			response := map[string]data.OutputItem{
				"name":                  m.Name,
				"version":               m.Version,
				"state":                 m.State,
				"path":                  m.Path,
				"id":                    m.ID,
				"start_time":            m.StartTimeUTC,
				"available_transitions": m.AvailableTransitions,
				"scout":                 m.Scout,
			}

			if m.Scout {
				response["current_state"] = m.ScoutState
			}

			return response, nil
		}
	}

	return nil, fmt.Errorf("no machine matching %q found", query)
}

func (s *MachineData) DLL() (*ddl.DDL, error) {
	sddl := &ddl.DDL{
		Metadata: ddl.Metadata{
			License:     "Apache-2.0",
			Author:      "R.I.Pienaar <rip@devco.net>",
			Timeout:     1,
			Name:        "machine_state",
			Version:     build.Version,
			URL:         "https://choria.io",
			Description: "Data about a Choria Autonomous Agent",
			Provider:    "golang",
		},
		Query: &common.InputItem{
			Prompt:      "Machine",
			Description: "The machine name, path or ID to match",
			Type:        common.InputTypeString,
			Default:     "",
			Optional:    false,
			Validation:  ".+",
			MaxLength:   256,
		},

		Output: map[string]*common.OutputItem{
			"name": {
				Description: "The machine name",
				DisplayAs:   "Name",
				Type:        common.OutputTypeString,
			},

			"version": {
				Description: "The machine version",
				DisplayAs:   "Version",
				Type:        common.OutputTypeString,
			},

			"state": {
				Description: "The state the machine is in currently",
				DisplayAs:   "Current State",
				Type:        common.OutputTypeString,
			},

			"path": {
				Description: "The path the machine is stored in on disk",
				DisplayAs:   "Path",
				Type:        common.OutputTypeString,
			},

			"id": {
				Description: "The unique instance id",
				DisplayAs:   "ID",
				Type:        common.OutputTypeString,
			},

			"start_time": {
				Description: "The time this machine started, seconds since 1970",
				DisplayAs:   "Path",
				Type:        common.OutputTypeInteger,
			},

			"available_transitions": {
				Description: "The names of transition events that's valid for this machine",
				DisplayAs:   "Available Transitions",
				Type:        common.OutputTypeArray,
			},

			"scout": {
				Description: "Indicates if this machine is a Scout check",
				DisplayAs:   "Scout",
				Type:        common.OutputTypeBoolean,
			},

			"current_state": {
				Description: "For Scout checks, this is the extended Scout state",
				DisplayAs:   "Scout State",
				Type:        common.OutputTypeHash,
			},
		},
	}

	return sddl, nil
}
