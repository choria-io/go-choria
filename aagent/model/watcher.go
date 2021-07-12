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
