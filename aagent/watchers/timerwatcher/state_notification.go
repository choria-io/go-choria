// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package timerwatcher

import (
	"encoding/json"
	"fmt"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	"github.com/choria-io/go-choria/aagent/watchers/event"
)

// StateNotification describes the current state of the watcher
// described by io.choria.machine.timer.exec.v1.state
type StateNotification struct {
	event.Event

	State string        `json:"state"`
	Timer time.Duration `json:"timer"`
}

// JSON creates a JSON representation of the notification
func (s *StateNotification) JSON() ([]byte, error) {
	return json.Marshal(s.CloudEvent())
}

// CloudEvent creates a CloudEvent from the state notification
func (s *StateNotification) CloudEvent() cloudevents.Event {
	return s.Event.CloudEvent(s)
}

// String is a string representation of the notification suitable for printing
func (s *StateNotification) String() string {
	return fmt.Sprintf("%s timer %s#%s state: %s with %v timer", s.Identity, s.Machine, s.Name, s.State, s.Timer)
}
