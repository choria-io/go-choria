package lifecycle

import (
	"encoding/json"
	"fmt"
)

// StartupEvent is a choria:lifecycle:alive:1 event
//
// In addition to the usually required fields it requires a Version()
// specified when producing this type of event
type AliveEvent struct {
	basicEvent
	Version string `json:"version"`
}

func init() {
	eventTypes["alive"] = Alive

	eventJSONParsers[Alive] = func(j []byte) (Event, error) {
		return newAliveEventFromJSON(j)
	}

	eventFactories[Alive] = func(opts ...Option) Event {
		return newAliveEvent(opts...)
	}
}

func newAliveEvent(opts ...Option) *AliveEvent {
	event := &AliveEvent{
		basicEvent: basicEvent{
			Protocol:  "choria:lifecycle:alive:1",
			Timestamp: timeStamp(),
			etype:     "alive",
			dtype:     Alive,
		},
	}

	for _, o := range opts {
		o(event)
	}

	return event
}

func newAliveEventFromJSON(j []byte) (*AliveEvent, error) {
	event := &AliveEvent{
		basicEvent: basicEvent{
			etype: "alive",
			dtype: Alive,
		},
	}
	err := json.Unmarshal(j, event)
	if err != nil {
		return nil, err
	}

	if event.Protocol != "choria:lifecycle:alive:1" {
		return nil, fmt.Errorf("invalid protocol '%s'", event.Protocol)
	}

	return event, nil
}

// String is text suitable to display on the console etc
func (e *AliveEvent) String() string {
	return fmt.Sprintf("[alive] %s: %s version %s", e.Ident, e.Component(), e.Version)
}

// SetVersion sets the version for the event
func (e *AliveEvent) SetVersion(v string) {
	e.Version = v
}
