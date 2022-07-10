// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package gossipwatcher

import (
	"encoding/json"
	"fmt"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	"github.com/choria-io/go-choria/aagent/watchers/event"
)

// StateNotification describes the current state of the watcher
// described by io.choria.machine.watcher.gossip.v1.state
type StateNotification struct {
	event.Event

	Subject   string `json:"previous_subject"`
	Payload   string `json:"previous_payload"`
	Published int64  `json:"previous_gossip"`
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
	return fmt.Sprintf("%s %s#%s subject: %s, previous: %v", s.Identity, s.Machine, s.Name, s.Subject, s.Published)
}
