package nagioswatcher

import (
	"encoding/json"
	"fmt"

	cloudevents "github.com/cloudevents/sdk-go"

	"github.com/choria-io/go-choria/aagent/util"
	"github.com/choria-io/go-choria/aagent/watchers/event"
)

type StateNotification struct {
	event.Event

	Plugin      string            `json:"plugin"`
	Status      string            `json:"status"`
	StatusCode  int               `json:"status_code"`
	Output      string            `json:"output"`
	CheckTime   int64             `json:"check_time"`
	PerfData    []util.PerfData   `json:"perfdata"`
	RunTime     float64           `json:"runtime"`
	History     []*Execution      `json:"history"`
	Annotations map[string]string `json:"annotations"`
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
	return fmt.Sprintf("%s %s#%s %s: %s", s.Identity, s.Machine, s.Name, s.Status, s.Output)
}
