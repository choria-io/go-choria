// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package kvwatcher

import (
	"encoding/json"
	"fmt"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	"github.com/choria-io/go-choria/aagent/watchers/event"
)

// StateNotification describes the current state of the watcher
// described by io.choria.machine.timer.exec.v1.state
type StateNotification struct {
	event.Event

	State  string `json:"state"`
	Bucket string `json:"bucket"`
	Key    string `json:"key,omitempty"`
	Mode   string `json:"mode"`
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
	if s.Key != "" {
		return fmt.Sprintf("%s key-value %s#%s %sing bucket: %s key: %s state: %s", s.Identity, s.Machine, s.Name, s.Mode, s.Bucket, s.Key, s.State)
	} else {
		return fmt.Sprintf("%s key-value %s#%s %sing bucket: %s state: %s", s.Identity, s.Machine, s.Name, s.Mode, s.Bucket, s.State)
	}
}
