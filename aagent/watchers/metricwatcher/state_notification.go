// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package metricwatcher

import (
	"encoding/json"
	"fmt"
	"strings"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	"github.com/choria-io/go-choria/aagent/watchers/event"
)

// StateNotification describes the current state of the watcher
// described by io.choria.machine.watcher.metric.v1.state
type StateNotification struct {
	event.Event

	Metrics Metric `json:"metrics"`
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
	metrics := []string{}

	for k, v := range s.Metrics.Metrics {
		metrics = append(metrics, fmt.Sprintf("%s=%0.3f", k, v))
	}

	return fmt.Sprintf("%s %s#%s metrics: %s", s.Identity, s.Machine, s.Name, strings.Join(metrics, ", "))
}
