package lifecycle

import (
	"encoding/json"
	"fmt"
)

// GovernorEvent is a io.choria.lifecycle.v1.governor event
//
// In addition to the usual required fields it requires a GovernorName(), GovernorSequence() and GovernorType() specified when producing this kind of event
type GovernorEvent struct {
	basicEvent
	Governor  string            `json:"governor"`
	Sequence  uint64            `json:"sequence"`
	EventType GovernorEventType `json:"event_type"`
}

type GovernorEventType string

const (
	// GovernorEnterEvent is when a Governor slot is obtained through active campaigning
	GovernorEnterEvent GovernorEventType = "enter"
	// GovernorExitEvent is when a slot is left, but not expired or evicted
	GovernorExitEvent GovernorEventType = "exit"
	// GovernorTimeoutEvent is when a slot could not be obtained after some time
	GovernorTimeoutEvent GovernorEventType = "timeouts"
	// GovernorEvictEvent is when a slot is evicted using a admin API
	GovernorEvictEvent GovernorEventType = "eviction"
)

func init() {
	eventTypes["governor"] = Governor

	eventJSONParsers[Governor] = func(j []byte) (Event, error) {
		return newGovernorEnterEventFromJSON(j)
	}

	eventFactories[Governor] = func(opts ...Option) Event {
		return newGovernorEvent(opts...)
	}
}

func newGovernorEvent(opts ...Option) *GovernorEvent {
	event := &GovernorEvent{basicEvent: newBasicEvent("governor")}

	for _, o := range opts {
		o(event)
	}

	return event
}

func (g *GovernorEvent) SetEventType(stage GovernorEventType) error {
	switch stage {
	case GovernorEnterEvent, GovernorExitEvent, GovernorTimeoutEvent, GovernorEvictEvent:
		g.EventType = stage
	default:
		return fmt.Errorf("invalid stage")
	}

	return nil
}

func (g *GovernorEvent) SetSequence(seq uint64) {
	g.Sequence = seq
}

func (g *GovernorEvent) SetGovernor(name string) {
	g.Governor = name
}

func (g *GovernorEvent) String() string {
	switch g.EventType {
	case GovernorExitEvent:
		if g.Sequence > 0 {
			return fmt.Sprintf("[governor] %s: vacated slot %d on %s", g.Ident, g.Sequence, g.Governor)
		} else {
			return fmt.Sprintf("[governor] %s: vacated %s", g.Ident, g.Governor)
		}

	case GovernorEnterEvent:
		if g.Sequence > 0 {
			return fmt.Sprintf("[governor] %s: obtained slot %d on %s", g.Ident, g.Sequence, g.Governor)
		} else {
			return fmt.Sprintf("[governor] %s: obtained slot on %s", g.Ident, g.Governor)
		}

	case GovernorTimeoutEvent:
		return fmt.Sprintf("[governor] %s: failed to obtain a slot on %s", g.Ident, g.Governor)

	case GovernorEvictEvent:
		if g.Sequence > 0 {
			return fmt.Sprintf("[governor] %s: evicted from slot %d on %s", g.Ident, g.Sequence, g.Governor)
		} else {
			return fmt.Sprintf("[governor] %s: evicted from %s", g.Ident, g.Governor)
		}

	default:
		return fmt.Sprintf("[governor] %s: unknown stage on Governor %s", g.Ident, g.Governor)
	}
}

func newGovernorEnterEventFromJSON(j []byte) (*GovernorEvent, error) {
	event := &GovernorEvent{basicEvent: newBasicEvent("governor")}

	err := json.Unmarshal(j, event)
	if err != nil {
		return nil, err
	}

	switch event.EventProtocol {
	case "io.choria.lifecycle.v1.governor":
	case "choria:lifecycle:governor:1":
		event.EventProtocol = "io.choria.lifecycle.v1.governor"
	default:
		return nil, fmt.Errorf("invalid protocol '%s'", event.EventProtocol)
	}

	if event.Governor == "" {
		return nil, fmt.Errorf("governor name is not set")
	}

	return event, nil
}
