package aagent

// MachineState is the state of a running machine
type MachineState struct {
	Name                 string   `json:"name" yaml:"name"`
	Version              string   `json:"version" yaml:"version"`
	State                string   `json:"state" yaml:"state"`
	Path                 string   `json:"path" yaml:"path"`
	ID                   string   `json:"id" yaml:"id"`
	StartTimeUTC         int64    `json:"start_time" yaml:"start_time"`
	AvailableTransitions []string `json:"available_transitions" yaml:"available_transitions"`
}

// AllMachineStates retrieves a list of machines and their states
func (a *AAgent) AllMachineStates() (states []MachineState, err error) {
	states = []MachineState{}

	a.Lock()
	defer a.Unlock()

	for _, m := range a.machines {
		state := MachineState{
			Name:                 m.machine.Name(),
			Version:              m.machine.Version(),
			Path:                 m.machine.Directory(),
			ID:                   m.machine.InstanceID(),
			State:                m.machine.State(),
			StartTimeUTC:         m.machine.StartTime().Unix(),
			AvailableTransitions: m.machine.AvailableTransitions(),
		}

		states = append(states, state)
	}

	return states, nil
}
