package lifecycle

import (
	"encoding/json"
	"fmt"
)

// ShutdownEvent is a choria:lifecycle:shutdown:1 event
type ShutdownEvent struct {
	Protocol  string `json:"protocol"`
	Identity  string `json:"identity"`
	Comp      string `json:"component"`
	Timestamp int64  `json:"timestamp"`
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
	event := &ShutdownEvent{
		Protocol:  "choria:lifecycle:shutdown:1",
		Timestamp: timeStamp(),
	}

	for _, o := range opts {
		o(event)
	}

	return event
}

func newShutdownEventFromJSON(j []byte) (*ShutdownEvent, error) {
	event := &ShutdownEvent{}
	err := json.Unmarshal(j, event)
	if err != nil {
		return nil, err
	}

	if event.Protocol != "choria:lifecycle:shutdown:1" {
		return nil, fmt.Errorf("invalid protocol '%s'", event.Protocol)
	}

	return event, nil
}

// Component is the component that produced the event
func (e *ShutdownEvent) Component() string {
	return e.Comp
}

// SetComponent sets the component for the event
func (e *ShutdownEvent) SetComponent(c string) {
	e.Comp = c
}

// SetIdentity sets the identity for the event
func (e *ShutdownEvent) SetIdentity(i string) {
	e.Identity = i
}

// Target is where to publish the event to
func (e *ShutdownEvent) Target() (string, error) {
	if e.Comp == "" {
		return "", fmt.Errorf("event is not complete, component has not been set")
	}

	return fmt.Sprintf("choria.lifecycle.event.shutdown.%s", e.Comp), nil
}

// String is text suitable to display on the console etc
func (e *ShutdownEvent) String() string {
	return fmt.Sprintf("[shutdown] %s: %s", e.Identity, e.Component())
}

// Type is the type of event
func (e *ShutdownEvent) Type() Type {
	return Shutdown
}
