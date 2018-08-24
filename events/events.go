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
	// Startup is an event components should publish when they start
	Startup Type = iota
)

type startupEvent struct {
	Protocol  string `json:"protocol"`
	Identity  string `json:"identity"`
	Version   string `json:"version"`
	Timestamp int64  `json:"timestamp"`
	Component string `json:"component"`
}

var mockTime int64

// RawPublishableConnector is a connection to the middleware
type RawPublishableConnector interface {
	PublishRaw(target string, data []byte) error
}

// PublishEvent publishes an event of type event to choria.lifecycle.event
func PublishEvent(event Type, component string, cfg *config.Config, conn RawPublishableConnector) error {
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

func newStartupEvent(identity string, component string) *startupEvent {
	return &startupEvent{
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
