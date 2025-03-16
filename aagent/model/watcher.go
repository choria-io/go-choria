// Copyright (c) 2021-2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"context"
	"sync"
	"time"
)

// Watcher is anything that can be used to watch the system for events
type Watcher interface {
	Name() string
	Type() string
	Run(context.Context, *sync.WaitGroup)
	NotifyStateChance()
	CurrentState() any
	AnnounceInterval() time.Duration
	Delete()
}

// WatcherConstructor creates a new watcher plugin
type WatcherConstructor interface {
	New(machine Machine, name string, states []string, requiredState []ForeignMachineState, failEvent string, successEvent string, interval string, ai time.Duration, properties map[string]any) (any, error)
	Type() string
	EventType() string
	UnmarshalNotification(n []byte) (any, error)
}

// ForeignMachineState describe a requirement on a foreign machine state
type ForeignMachineState struct {
	MachineName  string `json:"machine_name" yaml:"machine_name"`
	MachineState string `json:"state" yaml:"state"`
}
