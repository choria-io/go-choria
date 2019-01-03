package lifecycle

import (
	"encoding/json"
	"fmt"
)

// ShutdownEvent is a io.choria.lifecycle.v1.shutdown event
type ShutdownEvent struct {
	basicEvent
}

func init() {
	eventTypes["shutdown"] = Shutdown

	eventJSONParsers[Shutdown] = func(j []byte) (Event, error) {
		return newShutdownEventFromJSON(j)
	}

	eventFactories[Shutdown] = func(opts ...Option) Event {
		return newShutdownEvent(opts...)
	}
}

func newShutdownEvent(opts ...Option) *ShutdownEvent {
	event := &ShutdownEvent{basicEvent: newBasicEvent("shutdown")}

	for _, o := range opts {
		o(event)
	}

	return event
}

func newShutdownEventFromJSON(j []byte) (*ShutdownEvent, error) {
	event := &ShutdownEvent{basicEvent: newBasicEvent("shutdown")}
	err := json.Unmarshal(j, event)
	if err != nil {
		return nil, err
	}

	if event.Protocol == "choria:lifecycle:shutdown:1" {
		event.Protocol = "io.choria.lifecycle.v1.shutdown"
	}

	if event.Protocol != "io.choria.lifecycle.v1.shutdown" {
		return nil, fmt.Errorf("invalid protocol '%s'", event.Protocol)
	}

	return event, nil
}
