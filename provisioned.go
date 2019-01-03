package lifecycle

import (
	"encoding/json"
	"fmt"
)

// ProvisionedEvent is a io.choria.lifecycle.v1.provisioned event
type ProvisionedEvent struct {
	basicEvent
}

func init() {
	eventTypes["provisioned"] = Provisioned

	eventJSONParsers[Provisioned] = func(j []byte) (Event, error) {
		return newProvisionedEventFromJSON(j)
	}

	eventFactories[Provisioned] = func(opts ...Option) Event {
		return newProvisionedEvent(opts...)
	}
}

func newProvisionedEvent(opts ...Option) *ProvisionedEvent {
	event := &ProvisionedEvent{basicEvent: newBasicEvent("provisioned")}

	for _, o := range opts {
		o(event)
	}

	return event
}

func newProvisionedEventFromJSON(j []byte) (*ProvisionedEvent, error) {
	event := &ProvisionedEvent{basicEvent: newBasicEvent("provisioned")}

	err := json.Unmarshal(j, event)
	if err != nil {
		return nil, err
	}

	if event.Protocol == "choria:lifecycle:provisioned:1" {
		event.Protocol = "io.choria.lifecycle.v1.provisioned"
	}

	if event.Protocol != "io.choria.lifecycle.v1.provisioned" {
		return nil, fmt.Errorf("invalid protocol '%s'", event.Protocol)
	}

	return event, nil
}
