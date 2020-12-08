package watcher

import (
	"fmt"
	"sync"
	"time"
)

type Watcher struct {
	name             string
	wtype            string
	announceInterval time.Duration
	statechg         chan struct{}
	activeStates     []string
	machine          Machine
	succEvent        string
	failEvent        string

	deleteCb       func()
	currentStateCb func() interface{}

	mu sync.Mutex
}

func (w *Watcher) Machine() Machine {
	return w.machine
}

func (w *Watcher) SuccessEvent() string {
	return w.succEvent
}

func (w *Watcher) FailEvent() string {
	return w.failEvent
}

func (w *Watcher) StateChangeC() chan struct{} {
	return w.statechg
}

func (w *Watcher) SetDeleteFunc(f func()) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.deleteCb = f
}

func (w *Watcher) NotifyWatcherState(state interface{}) {
	w.machine.NotifyWatcherState(w.name, state)
}

func (w *Watcher) SuccessTransition() error {
	if w.succEvent == "" {
		return nil
	}

	return w.machine.Transition(w.succEvent)
}

func (w *Watcher) FailureTransition() error {
	if w.failEvent == "" {
		return nil
	}

	return w.machine.Transition(w.failEvent)
}

func (w *Watcher) Transition(event string) error {
	if event == "" {
		return nil
	}

	return w.machine.Transition(event)
}

func NewWatcher(name string, wtype string, announceInterval time.Duration, activeStates []string, machine Machine, fail string, success string) (*Watcher, error) {
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}

	if wtype == "" {
		return nil, fmt.Errorf("watcher type is required")
	}

	if machine == nil {
		return nil, fmt.Errorf("machine is required")
	}

	return &Watcher{
		name:             name,
		wtype:            wtype,
		announceInterval: announceInterval,
		statechg:         make(chan struct{}, 1),
		failEvent:        fail,
		succEvent:        success,
		machine:          machine,
		activeStates:     activeStates,
	}, nil
}

func (w *Watcher) NotifyStateChance() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if len(w.statechg) < cap(w.statechg) {
		w.statechg <- struct{}{}
	}
}

func (w *Watcher) CurrentState() interface{} {
	if w.currentStateCb != nil {
		return w.currentStateCb()
	}

	return nil
}

func (w *Watcher) AnnounceInterval() time.Duration {
	return w.announceInterval
}

func (w *Watcher) Type() string {
	return w.wtype
}

func (w *Watcher) Name() string {
	return w.name
}

func (w *Watcher) Delete() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.deleteCb != nil {
		w.deleteCb()
	}
}

func (w *Watcher) ShouldWatch() bool {
	if len(w.activeStates) == 0 {
		return true
	}

	for _, e := range w.activeStates {
		if e == w.machine.State() {
			return true
		}
	}

	return false
}

func (w *Watcher) Debugf(format string, args ...interface{}) {
	w.machine.Debugf(w.name, format, args...)
}

func (w *Watcher) Infof(format string, args ...interface{}) {
	w.machine.Infof(w.name, format, args...)
}

func (w *Watcher) Errorf(format string, args ...interface{}) {
	w.machine.Errorf(w.name, format, args...)
}
