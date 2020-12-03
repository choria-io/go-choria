package event

import (
	"time"

	cloudevents "github.com/cloudevents/sdk-go"

	"github.com/choria-io/go-choria/choria"
)

type Event struct {
	Protocol  string `json:"protocol"`
	Identity  string `json:"identity"`
	ID        string `json:"id"`
	Version   string `json:"version"`
	Timestamp int64  `json:"timestamp"`
	Type      string `json:"type"`
	Machine   string `json:"machine"`
	Name      string `json:"name"`
}

// CloudEvent creates a CloudEvent from the state notification
func (e *Event) CloudEvent(data interface{}) cloudevents.Event {
	event := cloudevents.NewEvent("1.0")

	event.SetType(e.Protocol)
	event.SetSource("io.choria.machine")
	event.SetSubject(e.Identity)
	event.SetID(choria.UniqueID())
	event.SetTime(time.Unix(e.Timestamp, 0))
	event.SetData(data)

	return event
}

// WatcherType is the type of watcher the notification is for - exec, file etc
func (e *Event) WatcherType() string {
	return e.Type
}

// SenderID is the identity of the event producer
func (e *Event) SenderID() string {
	return e.Identity
}
