package nagioswatcher

import (
	"encoding/json"
	"fmt"
	"time"

	cloudevents "github.com/cloudevents/sdk-go"

	"github.com/choria-io/go-choria/choria"
)

type StateNotification struct {
	Protocol   string  `json:"protocol"`
	Identity   string  `json:"identity"`
	ID         string  `json:"id"`
	Version    string  `json:"version"`
	Timestamp  int64   `json:"timestamp"`
	Type       string  `json:"type"`
	Machine    string  `json:"machine"`
	Name       string  `json:"name"`
	Plugin     string  `json:"plugin"`
	Status     string  `json:"status"`
	StatusCode int     `json:"status_code"`
	Output     string  `json:"output"`
	CheckTime  int64   `json:"check_time"`
	PerfData   string  `json:"perfdata"`
	RunTime    float64 `json:"runtime"`
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
	return fmt.Sprintf("%s %s#%s %s: %s", s.Identity, s.Machine, s.Name, s.Status, s.Output)
}

// WatcherType is the type of watcher the notification is for - exec, file etc
func (s *StateNotification) WatcherType() string {
	return s.Type
}

// SenderID is the identity of the event producer
func (s *StateNotification) SenderID() string {
	return s.Identity
}
