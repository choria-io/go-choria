package execwatcher

import (
	"encoding/json"
	"fmt"

	cloudevents "github.com/cloudevents/sdk-go"

	"github.com/choria-io/go-choria/aagent/watchers/event"
)

// StateNotification describes the current state of the watcher
// described by io.choria.machine.watcher.exec.v1.state
type StateNotification struct {
	event.Event

	Command         string `json:"command"`
	PreviousOutcome string `json:"previous_outcome"`
	PreviousRunTime int64  `json:"previous_run_time"`
}

// CloudEvent creates a CloudEvent from the state notification
func (s *StateNotification) CloudEvent() cloudevents.Event {
	return s.Event.CloudEvent(s)
}

// JSON creates a JSON representation of the notification
func (s *StateNotification) JSON() ([]byte, error) {
	return json.Marshal(s.CloudEvent())
}

// String is a string representation of the notification suitable for printing
func (s *StateNotification) String() string {
	return fmt.Sprintf("%s %s#%s command: %s, previous: %s ran: %.3fs", s.Identity, s.Machine, s.Name, s.Command, s.PreviousOutcome, float64(s.PreviousRunTime)/1000000000)
}
