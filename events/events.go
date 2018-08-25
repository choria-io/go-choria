package events

import (
	"encoding/json"
	"time"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/config"
)

// Type is a type of event this system supports
type Type int

const (
	_ = iota

	// Startup is an event components should publish when they start
	Startup Type = iota
)

// EventTypes allow lookup of a event Type by its string representation
var EventTypes map[string]Type

func init() {
	EventTypes = make(map[string]Type)
	EventTypes["startup"] = Startup
}

// EventTypeNames produce a list of valid event type names
func EventTypeNames() []string {
	names := []string{}

	for k := range EventTypes {
		names = append(names, k)
	}

	return names
}

// StartupEvent is a choria:lifecycle:startup:1 event
type StartupEvent struct {
	Protocol  string `json:"protocol"`
	Identity  string `json:"identity"`
	Version   string `json:"version"`
	Timestamp int64  `json:"timestamp"`
	Component string `json:"component"`
}

var mockTime int64

// PublishConnector is a connection to the middleware
type PublishConnector interface {
	PublishRaw(target string, data []byte) error
}

// PublishEvent publishes an event of type event to choria.lifecycle.event
func PublishEvent(event Type, component string, cfg *config.Config, conn PublishConnector) error {
	var body interface{}

	switch event {
	case Startup:
		body = newStartupEvent(cfg.Identity, component)
	}

	if body != nil {
		j, err := json.Marshal(body)
		if err != nil {
			return err
		}

		conn.PublishRaw("choria.lifecycle.event", j)
	}

	return nil
}

func newStartupEvent(identity string, component string) *StartupEvent {
	return &StartupEvent{
		Protocol:  "choria:lifecycle:startup:1",
		Identity:  identity,
		Version:   build.Version,
		Timestamp: timeStamp(),
		Component: component,
	}
}

func timeStamp() int64 {
	if mockTime != 0 {
		return mockTime
	}

	return time.Now().UTC().Unix()
}
