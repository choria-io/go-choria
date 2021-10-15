// Copyright (c) 2019-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package aagent

import (
	"sort"
)

// MachineState is the state of a running machine
type MachineState struct {
	Name                 string      `json:"name" yaml:"name"`
	Version              string      `json:"version" yaml:"version"`
	State                string      `json:"state" yaml:"state"`
	Path                 string      `json:"path" yaml:"path"`
	ID                   string      `json:"id" yaml:"id"`
	StartTimeUTC         int64       `json:"start_time" yaml:"start_time"`
	AvailableTransitions []string    `json:"available_transitions" yaml:"available_transitions"`
	Scout                bool        `json:"scout" yaml:"scout"`
	ScoutState           interface{} `json:"current_state,omitempty" yaml:"current_state,omitempty"`
}

// AllMachineStates retrieves a list of machines and their states
func (a *AAgent) AllMachineStates() (states []MachineState, err error) {
	states = []MachineState{}

	a.Lock()
	defer a.Unlock()

	for _, m := range a.machines {
		var (
			cstate interface{}
			scout  = false
		)

		for _, w := range m.machine.WatcherDefs {
			if w.Type == "nagios" {
				scout = true
				cstate, _ = m.machine.WatcherState(w.Name)
			}
		}

		state := MachineState{
			Name:                 m.machine.Name(),
			Version:              m.machine.Version(),
			Path:                 m.machine.Directory(),
			ID:                   m.machine.InstanceID(),
			State:                m.machine.State(),
			StartTimeUTC:         m.machine.StartTime().Unix(),
			AvailableTransitions: m.machine.AvailableTransitions(),
			Scout:                scout,
			ScoutState:           cstate,
		}

		states = append(states, state)
	}

	sort.Slice(states, func(i, j int) bool { return states[i].Name < states[j].Name })

	return states, nil
}
