package execwatcher

import (
	"encoding/json"
	"fmt"

	"github.com/choria-io/go-choria/choria"
	cloudevents "github.com/cloudevents/sdk-go"
	"time"
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

// CloudEvent creates a CloudEvent from the state notification
func (s *StateNotification) CloudEvent() cloudevents.Event {
	event := cloudevents.NewEvent("1.0")

	event.SetType(s.Protocol)
	event.SetSource("io.choria.machine")
	event.SetSubject(s.Identity)
	event.SetID(choria.UniqueID())
	event.SetTime(time.Unix(s.Timestamp, 0))
	event.SetData(s)

	return event
}

// JSON creates a JSON representation of the notification
func (s *StateNotification) JSON() ([]byte, error) {
	return json.Marshal(s.CloudEvent())
}

// String is a string representation of the notification suitable for printing
func (s *StateNotification) String() string {
	return fmt.Sprintf("%s %s#%s command: %s, previous: %s ran: %.3fs", s.Identity, s.Machine, s.Name, s.Command, s.PreviousOutcome, float64(s.PreviousRunTime)/1000000000)
}

// WatcherType is the type of watcher the notification is for - exec, file etc
func (s *StateNotification) WatcherType() string {
	return s.Type
}
