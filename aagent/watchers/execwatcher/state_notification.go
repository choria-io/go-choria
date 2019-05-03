package execwatcher

import (
	"encoding/json"
	"fmt"
)

// StateNotification describes the current state of the watcher
// described by io.choria.machine.watcher.exec.v1.state
type StateNotification struct {
	Protocol        string `json:"protocol"`
	Identity        string `json:"identity"`
	ID              string `json:"id"`
	Version         string `json:"version"`
	Timestamp       int64  `json:"timestamp"`
	Type            string `json:"type"`
	Machine         string `json:"machine"`
	Name            string `json:"name"`
	Command         string `json:"command"`
	PreviousOutcome string `json:"previous_outcome"`
	PreviousRunTime int64  `json:"previous_run_time"`
}

// JSON creates a JSON representation of the notification
func (s *StateNotification) JSON() ([]byte, error) {
	return json.Marshal(s)
}

// String is a string representation of the notification suitable for printing
func (s *StateNotification) String() string {
	return fmt.Sprintf("%s %s#%s command: %s, previous: %s ran: %.3fs", s.Identity, s.Machine, s.Name, s.Command, s.PreviousOutcome, float64(s.PreviousRunTime)/1000000000)
}

// WatcherType is the type of watcher the notification is for - exec, file etc
func (s *StateNotification) WatcherType() string {
	return s.Type
}
