// Copyright (c) 2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package httpswitchwatcher

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
	On
	Off
	// used to indicate that an external event - rpc or other watcher - initiated a transition
	OnNoTransition
	OffNoTransition

	wtype   = "httpswitch"
	version = "v1"
)

var stateNames = map[State]string{
	Unknown: "unknown",
	On:      "on",
	Off:     "off",
}

type properties struct {
	// ShouldOn when the machine is in any of these states the button will be report as on
	ShouldOn []string `mapstructure:"on_when"`
	// ShouldOff when the machine is in any of these states the button will be report as off
	ShouldOff []string `mapstructure:"off_when"`
	// ShouldDisable when the machine is in any of these states the button will stop functioning
	ShouldDisable []string `mapstructure:"disable_when"`
	// Annotations are additional annotations to apply to the watcher
	Annotations map[string]string `mapstructure:"annotations"`
}

type Watcher struct {
	*watcher.Watcher

	name        string
	machine     model.Machine
	properties  *properties
	buttonPress chan State
	previous    State
	httpStarted bool

	mu *sync.Mutex
}

func New(machine model.Machine, name string, states []string, required []model.ForeignMachineState, failEvent string, successEvent string, interval string, ai time.Duration, rawprops map[string]any) (any, error) {
	var err error

	mw := &Watcher{
		name:        name,
		machine:     machine,
		buttonPress: make(chan State, 1),
		mu:          &sync.Mutex{},
	}

	mw.Watcher, err = watcher.NewWatcher(name, wtype, ai, states, required, machine, failEvent, successEvent)
	if err != nil {
		return nil, err
	}

	err = mw.setProperties(rawprops)
	if err != nil {
		return nil, fmt.Errorf("could not set properties: %s", err)
	}

	return mw, nil
}

func (w *Watcher) TurnOn() (bool, error) {
	if w.shouldBeDisabled(w.machine.State()) {
		return false, fmt.Errorf("watcher is disabled")
	}
	if !w.ShouldWatch() {
		return false, fmt.Errorf("watcher is not in an active state")
	}

	w.buttonPress <- On

	return true, nil
}
func (w *Watcher) TurnOff() (bool, error) {
	if w.shouldBeDisabled(w.machine.State()) {
		return false, fmt.Errorf("watcher is disabled")
	}
	if !w.ShouldWatch() {
		return false, fmt.Errorf("switch is not in an active state")
	}

	w.buttonPress <- Off

	return true, nil
}

func (w *Watcher) registerWithHTTP() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.httpStarted {
		return
	}

	ham := w.machine.HttpManager()
	if ham != nil {
		ham.AddSwitchWatcher(w.machine.Name(), w)
		w.httpStarted = true
	}
}

func (w *Watcher) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	if w.ShouldWatch() {
		w.Watcher.StateChangeC() <- struct{}{}
	}

	for {
		w.registerWithHTTP()

		select {
		// button call backs
		case e := <-w.buttonPress:
			err := w.handleStateChange(e)
			if err != nil {
				w.Errorf("Could not handle button %s press: %v", stateNames[e], err)
			}

		// rpc initiated state changes would trigger this
		case <-w.Watcher.StateChangeC():
			mstate := w.machine.State()

			switch {
			case w.shouldBeOn(mstate):
				w.buttonPress <- OnNoTransition

			case w.shouldBeOff(mstate):
				w.buttonPress <- OffNoTransition
			}

		case <-ctx.Done():
			w.Infof("Stopping on context interrupt")
			w.removeFromHTTP()
			return
		}
	}
}

func (w *Watcher) handleStateChange(s State) error {
	if !w.ShouldWatch() {
		return nil
	}

	switch s {
	case On:
		w.setPreviousState(s)
		w.NotifyWatcherState(w.CurrentState())
		return w.SuccessTransition()

	case OnNoTransition:
		w.setPreviousState(On)
		w.NotifyWatcherState(w.CurrentState())
		return nil

	case Off:
		w.setPreviousState(s)
		w.NotifyWatcherState(w.CurrentState())
		return w.FailureTransition()

	case OffNoTransition:
		w.setPreviousState(Off)
		w.machine.NotifyWatcherState(w.name, w.CurrentState())
		return nil
	default:
		return fmt.Errorf("invalid state change event: %s", stateNames[s])
	}
}

func (w *Watcher) setPreviousState(s State) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.previous = s
}

func (w *Watcher) shouldBeOff(s string) bool {
	for _, state := range w.properties.ShouldOff {
		if state == s {
			return true
		}
	}

	return false
}

func (w *Watcher) shouldBeOn(s string) bool {
	for _, state := range w.properties.ShouldOn {
		if state == s {
			return true
		}
	}

	return false
}

func (w *Watcher) shouldBeDisabled(s string) bool {
	for _, state := range w.properties.ShouldDisable {
		if state == s {
			return true
		}
	}

	return false
}

func (w *Watcher) removeFromHTTP() {
	w.mu.Lock()
	defer w.mu.Unlock()

	mgr := w.machine.HttpManager()
	if mgr != nil {
		mgr.RemoveSwitchWatcher(w.machine.Name(), w)
	}
}

func (w *Watcher) Delete() {
	w.removeFromHTTP()
}

func (w *Watcher) setProperties(props map[string]any) error {
	if w.properties == nil {
		w.properties = &properties{
			ShouldDisable: []string{},
			ShouldOff:     []string{},
			ShouldOn:      []string{},
			Annotations:   make(map[string]string),
		}
	}

	err := util.ParseMapStructure(props, w.properties)
	if err != nil {
		return err
	}

	return w.validate()
}

func (w *Watcher) CurrentState() any {
	w.mu.Lock()
	defer w.mu.Unlock()

	s := &StateNotification{
		Event:           event.New(w.name, wtype, version, w.machine),
		PreviousOutcome: stateNames[w.previous],
		IsOn:            w.previous == On,
		Annotations:     w.properties.Annotations,
	}

	if s.Annotations == nil {
		s.Annotations = make(map[string]string)
	}

	return s
}

func (w *Watcher) validate() error {
	return nil
}
