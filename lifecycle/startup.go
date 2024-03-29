// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package lifecycle

import (
	"encoding/json"
	"fmt"
)

// StartupEvent is a io.choria.lifecycle.v1.startup event
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
	event := &StartupEvent{basicEvent: newBasicEvent("startup")}

	for _, o := range opts {
		o(event)
	}

	return event
}

func newStartupEventFromJSON(j []byte) (*StartupEvent, error) {
	event := &StartupEvent{basicEvent: newBasicEvent("startup")}

	err := json.Unmarshal(j, event)
	if err != nil {
		return nil, err
	}

	switch event.EventProtocol {
	case "io.choria.lifecycle.v1.startup":
	case "choria:lifecycle:startup:1":
		event.EventProtocol = "io.choria.lifecycle.v1.startup"
	default:
		return nil, fmt.Errorf("invalid protocol '%s'", event.EventProtocol)
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
