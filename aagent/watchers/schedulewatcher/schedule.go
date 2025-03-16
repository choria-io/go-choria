// Copyright (c) 2019-2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package schedulewatcher

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/choria-io/go-choria/aagent/model"
	"github.com/choria-io/go-choria/aagent/util"
	"github.com/choria-io/go-choria/aagent/watchers/event"
	"github.com/choria-io/go-choria/aagent/watchers/watcher"
)

type State int

const (
	Unknown State = iota
	Off
	On
	Skipped

	wtype   = "schedule"
	version = "v1"
)

var stateNames = map[State]string{
	Unknown: "unknown",
	Off:     "off",
	On:      "on",
	Skipped: "skipped",
}

type properties struct {
	Duration             time.Duration
	StartSplay           time.Duration `mapstructure:"start_splay"`
	SkipTriggerOnReenter bool          `mapstructure:"skip_trigger_on_reenter"`
	Schedules            []string
}

type Watcher struct {
	*watcher.Watcher
	properties *properties
	name       string
	machine    model.Machine
	items      []*scheduleItem

	// each item sends a 1 or -1 into this to increment or decrement the counter
	// when the ctr is > 0 the switch should be on, this handles multiple schedules
	// overlapping and keeping it on for longer than a single schedule would be
	ctrq chan int
	ctr  int

	triggered bool

	state         State
	previousState State

	mu *sync.Mutex
}

func New(machine model.Machine, name string, states []string, required []model.ForeignMachineState, failEvent string, successEvent string, interval string, ai time.Duration, properties map[string]any) (any, error) {
	var err error

	sw := &Watcher{
		name:    name,
		machine: machine,
		ctrq:    make(chan int, 1),
		ctr:     0,
		mu:      &sync.Mutex{},
	}

	sw.Watcher, err = watcher.NewWatcher(name, wtype, ai, states, required, machine, failEvent, successEvent)
	if err != nil {
		return nil, err
	}

	err = sw.setProperties(properties)
	if err != nil {
		return nil, fmt.Errorf("could not set properties: %s", err)
	}

	return sw, nil
}

func (w *Watcher) watchSchedule(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case i := <-w.ctrq:
			w.Debugf("Handling state change counter %v while ctr=%v", i, w.ctr)
			w.mu.Lock()

			w.ctr = w.ctr + i

			// shouldn't happen but lets handle it
			if w.ctr < 0 {
				w.ctr = 0
			}

			if w.ctr == 0 {
				w.Debugf("State going off due to ctr change to 0")
				w.state = Off
			} else {
				w.Debugf("State going on due to ctr change of %v", i)
				w.state = On
			}

			w.mu.Unlock()

		case <-ctx.Done():
			return
		}
	}
}

func (w *Watcher) setPreviousState(s State) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.previousState = s
}

func (w *Watcher) watch() (err error) {
	if !w.ShouldWatch() {
		w.setPreviousState(Skipped)

		return nil
	}

	// nothing changed
	if w.previousState == w.state {
		return nil
	}

	w.setPreviousState(w.state)

	switch w.state {
	case Off, Unknown:
		w.setTriggered(false)
		w.NotifyWatcherState(w.CurrentState())
		return w.FailureTransition()

	case On:
		if w.properties.SkipTriggerOnReenter && w.didTrigger() {
			w.Debugf("Skipping success transition that's already fired in this schedule due to skip_trigger_on_reenter")
			return nil
		}

		w.setTriggered(true)
		w.setPreviousState(w.state)
		w.NotifyWatcherState(w.CurrentState())
		return w.SuccessTransition()

	case Skipped:
		// not doing anything when we aren't eligible, regular announces happen

	}

	return nil
}

func (w *Watcher) setTriggered(s bool) {
	w.mu.Lock()
	w.triggered = s
	w.mu.Unlock()
}

func (w *Watcher) didTrigger() bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.triggered
}

func (w *Watcher) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	w.Infof("schedule watcher starting with %d items", len(w.items))

	wg.Add(1)
	go w.watchSchedule(ctx, wg)

	for _, item := range w.items {
		wg.Add(1)
		go item.start(ctx, wg)
	}

	tick := time.NewTicker(500 * time.Millisecond)

	for {
		select {
		case <-tick.C:
			err := w.watch()
			if err != nil {
				w.Errorf("Could not handle current scheduler state: %s", err)
			}

		case <-w.StateChangeC():
			err := w.watch()
			if err != nil {
				w.Errorf("Could not handle current scheduler state: %s", err)
			}

		case <-ctx.Done():
			tick.Stop()
			w.Infof("Stopping on context interrupt")
			return
		}
	}
}

func (w *Watcher) validate() error {
	if w.properties.Duration < time.Second {
		w.properties.Duration = time.Minute
	}

	if len(w.properties.Schedules) == 0 {
		return fmt.Errorf("no schedules defined")
	}

	return nil
}

func (w *Watcher) setProperties(props map[string]any) error {
	if w.properties == nil {
		w.properties = &properties{
			Schedules: []string{},
		}
	}

	err := util.ParseMapStructure(props, w.properties)
	if err != nil {
		return err
	}

	for _, spec := range w.properties.Schedules {
		item, err := newSchedItem(spec, w)
		if err != nil {
			return fmt.Errorf("could not parse '%s': %s", spec, err)
		}

		w.items = append(w.items, item)
	}

	if w.properties.StartSplay > w.properties.Duration/2 {
		return fmt.Errorf("start splay %v is bigger than half the duration %v", w.properties.StartSplay, w.properties.Duration)
	}

	return w.validate()
}

func (w *Watcher) CurrentState() any {
	w.mu.Lock()
	defer w.mu.Unlock()

	s := &StateNotification{
		Event: event.New(w.name, wtype, version, w.machine),
		State: stateNames[w.state],
	}

	return s
}
