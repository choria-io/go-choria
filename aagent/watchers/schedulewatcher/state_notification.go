package schedulewatcher

import (
	"encoding/json"
	"fmt"
)

// StateNotification describes the current state of the watcher
// described by io.choria.machine.watcher.exec.v1.state
type StateNotification struct {
	Protocol  string `json:"protocol"`
	Identity  string `json:"identity"`
	ID        string `json:"id"`
	Version   string `json:"version"`
	Timestamp int64  `json:"timestamp"`
	Type      string `json:"type"`
	Machine   string `json:"machine"`
	Name      string `json:"name"`
	State     string `json:"state"`
}

// JSON creates a JSON representation of the notification
func (s *StateNotification) JSON() ([]byte, error) {
	return json.Marshal(s)
}

// String is a string representation of the notification suitable for printing
func (s *StateNotification) String() string {
	return fmt.Sprintf("%s %s#%s state: %s", s.Identity, s.Machine, s.Name, s.State)
}

// WatcherType is the type of watcher the notification is for - exec, file etc
func (s *StateNotification) WatcherType() string {
	return s.Type
}
