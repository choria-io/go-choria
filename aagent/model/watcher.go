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
	CurrentState() interface{}
	AnnounceInterval() time.Duration
	Delete()
}

// WatcherConstructor creates a new watcher plugin
type WatcherConstructor interface {
	New(machine Machine, name string, states []string, failEvent string, successEvent string, interval string, ai time.Duration, properties map[string]interface{}) (interface{}, error)
	Type() string
	EventType() string
	UnmarshalNotification(n []byte) (interface{}, error)
}
