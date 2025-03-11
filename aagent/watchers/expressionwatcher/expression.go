// Copyright (c) 2024-2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package expressionwatcher

import (
	"context"
	"fmt"
	"github.com/choria-io/go-choria/aagent/watchers"
	"sync"
	"time"

	"github.com/choria-io/go-choria/aagent/model"
	"github.com/choria-io/go-choria/aagent/util"
	"github.com/choria-io/go-choria/aagent/watchers/event"
	"github.com/choria-io/go-choria/aagent/watchers/watcher"
	iu "github.com/choria-io/go-choria/internal/util"
)

type State int

const (
	SuccessWhen State = iota
	FailWhen
	NoMatch
	Skipped
	Error

	wtype   = "expression"
	version = "v1"
)

var stateNames = map[State]string{
	SuccessWhen: "success_when",
	FailWhen:    "failed_when",
	NoMatch:     "no_match",
	Skipped:     "skipped",
	Error:       "error",
}

type properties struct {
	FailWhen    string `mapstructure:"fail_when"`
	SuccessWhen string `mapstructure:"success_when"`
}

type Watcher struct {
	*watcher.Watcher
	properties *properties

	interval time.Duration
	name     string
	machine  model.Machine

	previous  State
	terminate chan struct{}
	mu        *sync.Mutex
}

func New(machine model.Machine, name string, states []string, required []watchers.ForeignMachineState, failEvent string, successEvent string, interval string, ai time.Duration, properties map[string]any) (any, error) {
	var err error

	ew := &Watcher{
		name:      name,
		machine:   machine,
		interval:  10 * time.Second,
		terminate: make(chan struct{}),
		previous:  Skipped,
		mu:        &sync.Mutex{},
	}

	ew.Watcher, err = watcher.NewWatcher(name, wtype, ai, states, required, machine, failEvent, successEvent)
	if err != nil {
		return nil, err
	}

	err = ew.setProperties(properties)
	if err != nil {
		return nil, fmt.Errorf("could not set properties: %s", err)
	}

	if interval != "" {
		ew.interval, err = iu.ParseDuration(interval)
		if err != nil {
			return nil, fmt.Errorf("invalid interval: %v", err)
		}

		if ew.interval < 500*time.Millisecond {
			return nil, fmt.Errorf("interval %v is too small", ew.interval)
		}
	}

	return ew, nil
}

func (w *Watcher) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	w.Infof("Expression watcher starting")

	tick := time.NewTicker(w.interval)

	for {
		select {
		case <-tick.C:
			w.Debugf("Performing watch due to ticker")
			w.performWatch()

		case <-w.StateChangeC():
			w.Debugf("Performing watch due to state change")
			w.performWatch()

		case <-w.terminate:
			w.Infof("Handling terminate notification")
			return

		case <-ctx.Done():
			w.Infof("Stopping on context interrupt")
			return
		}
	}
}
func (w *Watcher) performWatch() {
	err := w.handleCheck(w.watch())
	if err != nil {
		w.Errorf("could not handle watcher event: %s", err)
	}
}

func (w *Watcher) handleCheck(state State, err error) error {
	w.mu.Lock()
	previous := w.previous
	w.previous = state
	w.mu.Unlock()

	// shouldn't happen but just a safety here
	if err != nil {
		state = Error
	}

	switch state {
	case SuccessWhen:
		w.NotifyWatcherState(w.CurrentState())

		if previous != SuccessWhen {
			return w.SuccessTransition()
		}

	case FailWhen:
		w.NotifyWatcherState(w.CurrentState())

		if previous != FailWhen {
			return w.FailureTransition()
		}

	case Error:
		if err != nil {
			w.Errorf("Evaluating expressions failed: %v", err)
		}

		w.NotifyWatcherState(w.CurrentState())
	}

	return nil
}

func (w *Watcher) watch() (state State, err error) {
	if !w.ShouldWatch() {
		return Skipped, nil
	}

	if w.properties.SuccessWhen != "" {
		res, err := w.evaluateExpression(w.properties.SuccessWhen)
		if err != nil {
			return Error, err
		}

		if res {
			return SuccessWhen, nil
		}
	}

	if w.properties.FailWhen != "" {
		res, err := w.evaluateExpression(w.properties.FailWhen)
		if err != nil {
			return Error, err
		}

		if res {
			return FailWhen, nil
		}
	}

	return NoMatch, nil
}

func (w *Watcher) CurrentState() any {
	w.mu.Lock()
	defer w.mu.Unlock()

	return &StateNotification{
		Event:           event.New(w.name, wtype, version, w.machine),
		PreviousOutcome: stateNames[w.previous],
	}
}

func (w *Watcher) Delete() {
	close(w.terminate)
}

func (w *Watcher) setProperties(props map[string]any) error {
	if w.properties == nil {
		w.properties = &properties{}
	}

	err := util.ParseMapStructure(props, w.properties)
	if err != nil {
		return err
	}

	return w.validate()
}

func (w *Watcher) validate() error {
	if w.interval < time.Second {
		return fmt.Errorf("interval should be more than 1 second: %v", w.interval)
	}

	if w.properties.FailWhen == "" && w.properties.SuccessWhen == "" {
		return fmt.Errorf("success_when or fail_when is required")
	}

	return nil
}
