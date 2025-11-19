// Copyright (c) 2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package haswitchwatcher

import (
	"encoding/json"
	"fmt"

	"github.com/choria-io/go-choria/aagent/watchers/event"
	cloudevents "github.com/cloudevents/sdk-go/v2"
)

// StateNotification describes the current state of the watcher
// described by io.choria.machine.watcher.haswitch.v1.state
type StateNotification struct {
	event.Event
	PreviousOutcome string `json:"status"`
	IsOn            bool   `json:"is_on"`
}

// CloudEvent creates a CloudEvent from the state notification
func (s *StateNotification) CloudEvent() cloudevents.Event {
	return s.Event.CloudEvent(s)
}

// JSON creates a JSON representation of the notification
func (s *StateNotification) JSON() ([]byte, error) {
	return json.Marshal(s.CloudEvent())
}

// String is a string representation of the notification suitable for printing
func (s *StateNotification) String() string {
	return fmt.Sprintf("%s %s#%s previous: %v", s.Identity, s.Machine, s.Name, s.PreviousOutcome)
}
