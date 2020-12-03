package schedulewatcher

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

	State string `json:"state"`
}

// JSON creates a JSON representation of the notification
func (s *StateNotification) JSON() ([]byte, error) {
	return json.Marshal(s.CloudEvent())
}

// CloudEvent creates a CloudEvent from the state notification
func (s *StateNotification) CloudEvent() cloudevents.Event {
	return s.Event.CloudEvent(s)
}

// String is a string representation of the notification suitable for printing
func (s *StateNotification) String() string {
	return fmt.Sprintf("%s %s#%s state: %s", s.Identity, s.Machine, s.Name, s.State)
}
