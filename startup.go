package lifecycle

import (
	"encoding/json"
	"fmt"
)

// StartupEvent is a choria:lifecycle:startup:1 event
type StartupEvent struct {
	Protocol  string `json:"protocol"`
	Identity  string `json:"identity"`
	Version   string `json:"version"`
	Timestamp int64  `json:"timestamp"`
	Comp      string `json:"component"`
}

func init() {
	eventTypes["startup"] = Startup

	eventJSONParsers[Startup] = func(j []byte) (Event, error) {
		return newStartupEventFromJSON(j)
	}

	eventFactories[Startup] = func(opts ...Option) Event {
		return newStartupEvent(opts...)
	}
}

func newStartupEvent(opts ...Option) *StartupEvent {
	event := &StartupEvent{
		Protocol:  "choria:lifecycle:startup:1",
		Timestamp: timeStamp(),
	}

	for _, o := range opts {
		o(event)
	}

	return event
}

func newStartupEventFromJSON(j []byte) (*StartupEvent, error) {
	event := &StartupEvent{}
	err := json.Unmarshal(j, event)
	if err != nil {
		return nil, err
	}

	if event.Protocol != "choria:lifecycle:startup:1" {
		return nil, fmt.Errorf("invalid protocol '%s'", event.Protocol)
	}

	return event, nil
}

// Component is the component that produced the event
func (e *StartupEvent) Component() string {
	return e.Comp
}

// SetComponent sets the component for the event
func (e *StartupEvent) SetComponent(c string) {
	e.Comp = c
}

// SetVersion sets the version for the event
func (e *StartupEvent) SetVersion(v string) {
	e.Version = v
}

// SetIdentity sets the identity for the event
func (e *StartupEvent) SetIdentity(i string) {
	e.Identity = i
}

// Target is where to publish the event to
func (e *StartupEvent) Target() (string, error) {
	if e.Comp == "" {
		return "", fmt.Errorf("event is not complete, component has not been set")
	}

	return fmt.Sprintf("choria.lifecycle.event.startup.%s", e.Comp), nil
}

// String is text suitable to display on the console etc
func (e *StartupEvent) String() string {
	return fmt.Sprintf("[startup] %s: %s version %s", e.Identity, e.Component(), e.Version)
}

// Type is the type of event
func (e *StartupEvent) Type() Type {
	return Startup
}
