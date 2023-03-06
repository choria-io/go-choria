// Copyright (c) 2023, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package lifecycle

import (
	"encoding/json"
	"fmt"
)

type UpgradedEvent struct {
	basicEvent
	Version    string `json:"version"`
	NewVersion string `json:"new_version"`
}

func init() {
	eventTypes["upgraded"] = Upgraded

	eventJSONParsers[Upgraded] = func(j []byte) (Event, error) {
		return newUpgradeEventFromJSON(j)
	}

	eventFactories[Upgraded] = func(opts ...Option) Event {
		return newUpgradeEvent(opts...)
	}
}

func newUpgradeEvent(opts ...Option) *UpgradedEvent {
	event := &UpgradedEvent{basicEvent: newBasicEvent("upgraded")}

	for _, o := range opts {
		o(event)
	}

	return event
}

func newUpgradeEventFromJSON(j []byte) (*UpgradedEvent, error) {
	event := newUpgradeEvent()

	err := json.Unmarshal(j, event)
	if err != nil {
		return nil, err
	}

	switch event.EventProtocol {
	case "io.choria.lifecycle.v1.upgraded":
	case "choria:lifecycle:upgraded:1":
		event.EventProtocol = "io.choria.lifecycle.v1.upgraded"
	default:
		return nil, fmt.Errorf("invalid protocol '%s'", event.EventProtocol)
	}

	return event, nil
}

// String is text suitable to display on the console etc
func (e *UpgradedEvent) String() string {
	return fmt.Sprintf("[upgraded] %s: %s version %s to %s", e.Ident, e.Component(), e.Version, e.NewVersion)
}

// SetVersion sets the version for the event
func (e *UpgradedEvent) SetVersion(v string) {
	e.Version = v
}

// SetNewVersion sets the version for the event
func (e *UpgradedEvent) SetNewVersion(v string) {
	e.NewVersion = v
}
