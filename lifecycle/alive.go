// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package lifecycle

import (
	"encoding/json"
	"fmt"
)

// AliveEvent is a io.choria.lifecycle.v1.alive event
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
	event := &AliveEvent{basicEvent: newBasicEvent("alive")}

	for _, o := range opts {
		o(event)
	}

	return event
}

func newAliveEventFromJSON(j []byte) (*AliveEvent, error) {
	event := &AliveEvent{basicEvent: newBasicEvent("alive")}

	err := json.Unmarshal(j, event)
	if err != nil {
		return nil, err
	}

	switch event.EventProtocol {
	case "io.choria.lifecycle.v1.alive":
	case "choria:lifecycle:alive:1":
		event.EventProtocol = "io.choria.lifecycle.v1.alive"
	default:
		return nil, fmt.Errorf("invalid protocol '%s'", event.EventProtocol)
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
