package aagent

import "time"

type MachineState struct {
	Name         string
	Version      string
	State        string
	Path         string
	ID           string
	StartTimeUTC time.Time
}

// AllMachineStates retrieves a list of machines and their states
func (a *AAgent) AllMachineStates() (states []MachineState, err error) {
	states = []MachineState{}

	a.Lock()
	defer a.Unlock()

	for _, m := range a.machines {
		state := MachineState{
			Name:         m.machine.Name(),
			Version:      m.machine.Version(),
			Path:         m.machine.Directory(),
			ID:           m.machine.InstanceID(),
			State:        m.machine.State(),
			StartTimeUTC: m.machine.StartTime(),
		}

		states = append(states, state)
	}

	return states, nil
}
