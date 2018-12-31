package lifecycle

import (
	"encoding/json"
	"fmt"
)

// StartupEvent is a choria:lifecycle:startup:1 event
//
// In addition to the usually required fields it requires a Version()
// specified when producing this type of event
type StartupEvent struct {
	basicEvent
	Version string `json:"version"`
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
		basicEvent: basicEvent{
			Protocol:  "choria:lifecycle:startup:1",
			Timestamp: timeStamp(),
			etype:     "startup",
			dtype:     Startup,
		},
	}

	for _, o := range opts {
		o(event)
	}

	return event
}

func newStartupEventFromJSON(j []byte) (*StartupEvent, error) {
	event := &StartupEvent{
		basicEvent: basicEvent{
			etype: "startup",
			dtype: Startup,
		},
	}
	err := json.Unmarshal(j, event)
	if err != nil {
		return nil, err
	}

	if event.Protocol != "choria:lifecycle:startup:1" {
		return nil, fmt.Errorf("invalid protocol '%s'", event.Protocol)
	}

	return event, nil
}

// String is text suitable to display on the console etc
func (e *StartupEvent) String() string {
	return fmt.Sprintf("[startup] %s: %s version %s", e.Ident, e.Component(), e.Version)
}

// SetVersion sets the version for the event
func (e *StartupEvent) SetVersion(v string) {
	e.Version = v
}
