package lifecycle

import (
	"encoding/json"
	"fmt"
)

// ShutdownEvent is a choria:lifecycle:shutdown:1 event
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
	event := &ShutdownEvent{
		basicEvent: basicEvent{
			Protocol:  "choria:lifecycle:shutdown:1",
			Timestamp: timeStamp(),
			etype:     "shutdown",
			dtype:     Shutdown,
		},
	}

	for _, o := range opts {
		o(event)
	}

	return event
}

func newShutdownEventFromJSON(j []byte) (*ShutdownEvent, error) {
	event := &ShutdownEvent{
		basicEvent: basicEvent{
			etype: "shutdown",
			dtype: Shutdown,
		},
	}
	err := json.Unmarshal(j, event)
	if err != nil {
		return nil, err
	}

	if event.Protocol != "choria:lifecycle:shutdown:1" {
		return nil, fmt.Errorf("invalid protocol '%s'", event.Protocol)
	}

	return event, nil
}
