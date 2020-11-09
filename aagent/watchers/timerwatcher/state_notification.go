package timerwatcher

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/choria-io/go-choria/choria"
	cloudevents "github.com/cloudevents/sdk-go"
)

// StateNotification describes the current state of the watcher
// described by io.choria.machine.timer.exec.v1.state
type StateNotification struct {
	Protocol  string        `json:"protocol"`
	Identity  string        `json:"identity"`
	ID        string        `json:"id"`
	Version   string        `json:"version"`
	Timestamp int64         `json:"timestamp"`
	Type      string        `json:"type"`
	Machine   string        `json:"machine"`
	Name      string        `json:"name"`
	State     string        `json:"state"`
	Timer     time.Duration `json:"timer"`
}

// JSON creates a JSON representation of the notification
func (s *StateNotification) JSON() ([]byte, error) {
	return json.Marshal(s.CloudEvent())
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

// String is a string representation of the notification suitable for printing
func (s *StateNotification) String() string {
	return fmt.Sprintf("%s %s#%s state: %s", s.Identity, s.Machine, s.Name, s.State)
}

// WatcherType is the type of watcher the notification is for - exec, file etc
func (s *StateNotification) WatcherType() string {
	return s.Type
}

// SenderID is the identity of the event producer
func (s *StateNotification) SenderID() string {
	return s.Identity
}
