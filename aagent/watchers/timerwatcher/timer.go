package timerwatcher

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type State int

const (
	Stopped State = iota
	Running
)

var stateNames = map[State]string{
	Running: "running",
	Stopped: "stopped",
}

type Machine interface {
	State() string
	Name() string
	Identity() string
	InstanceID() string
	Version() string
	TimeStampSeconds() int64
	NotifyWatcherState(string, interface{})
	Transition(t string, args ...interface{}) error
	Debugf(name string, format string, args ...interface{})
	Infof(name string, format string, args ...interface{})
	Errorf(name string, format string, args ...interface{})
}

type Watcher struct {
	name             string
	states           []string
	startEvent       string
	stopEvent        string
	machine          Machine
	state            State
	announceInterval time.Duration

	terminate   chan struct{}
	statechg    chan struct{}
	cancelTimer func()

	time time.Duration

	sync.Mutex
}

func New(machine Machine, name string, states []string, failEvent string, successEvent string, ai time.Duration, properties map[string]interface{}) (watcher *Watcher, err error) {
	w := &Watcher{
		name:             name,
		states:           states,
		startEvent:       successEvent,
		stopEvent:        failEvent,
		machine:          machine,
		state:            0,
		terminate:        make(chan struct{}),
		statechg:         make(chan struct{}, 1),
		announceInterval: ai,
	}

	err = w.setProperties(properties)
	if err != nil {
		return nil, fmt.Errorf("could not set properties: %s", err)
	}

	return w, nil
}

func (w *Watcher) Delete() {
	close(w.terminate)
}

func (w *Watcher) forceTimerStop() {
	w.Lock()
	cancel := w.cancelTimer
	w.Unlock()

	if cancel != nil {
		w.machine.Infof(w.name, "Stopping timer early on state transition to %s", w.machine.State())
		cancel()
	}
}

func (w *Watcher) timeStart() {
	w.Lock()
	cancel := w.cancelTimer
	w.Unlock()

	if cancel != nil {
		w.machine.Infof(w.name, "Timer was running, resetting to %v", w.time)
		cancel()
	}

	go func() {
		timer := time.NewTimer(w.time)
		ctx, cancel := context.WithCancel(context.Background())

		w.Lock()
		w.state = Running
		w.cancelTimer = cancel
		w.Unlock()

		if w.startEvent != "" {
			w.machine.Transition(w.startEvent)
		}

		select {
		case <-timer.C:
			w.Lock()
			w.state = Stopped
			w.cancelTimer()
			w.cancelTimer = nil
			w.Unlock()

			w.machine.NotifyWatcherState(w.name, w.CurrentState())
			if w.stopEvent != "" {
				w.machine.Transition(w.stopEvent)
			}

		case <-ctx.Done():
			w.Lock()
			w.cancelTimer = nil
			timer.Stop()
			w.state = Stopped
			w.Unlock()

			w.machine.NotifyWatcherState(w.name, w.CurrentState())

		case <-w.terminate:
			w.Lock()
			w.cancelTimer = nil
			w.state = Stopped
			w.Unlock()
			return
		}
	}()

	w.machine.NotifyWatcherState(w.name, w.CurrentState())
}

func (w *Watcher) watch() {
	if !w.shouldCheck() {
		w.machine.Infof(w.name, "Forcing timer off")
		w.forceTimerStop()
		return
	}

	w.machine.Infof(w.name, "Starting timer")
	w.timeStart()
}

func (w *Watcher) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	w.machine.Infof(w.name, "Timer watcher starting with %v timer", w.time)

	// handle initial state
	w.watch()

	for {
		select {
		case <-w.statechg:
			w.watch()
		case <-w.terminate:
			w.machine.Infof(w.name, "Handling terminate notification")
			return
		case <-ctx.Done():
			w.machine.Infof(w.name, "Stopping on context interrupt")
			return
		}
	}
}

func (w *Watcher) setProperties(p map[string]interface{}) (err error) {
	ti, ok := p["timer"]
	if !ok {
		return fmt.Errorf("timer is required")
	}

	tis, ok := ti.(string)
	if !ok {
		return fmt.Errorf("time must be a string like 1h")
	}

	w.time, err = time.ParseDuration(tis)
	if err != nil {
		return fmt.Errorf("invalid time %s: %s", tis, err)
	}

	return nil
}

func (w *Watcher) shouldCheck() bool {
	if len(w.states) == 0 {
		return true
	}

	for _, e := range w.states {
		if e == w.machine.State() {
			return true
		}
	}

	return false
}

func (w *Watcher) Type() string {
	return "timer"
}

func (w *Watcher) AnnounceInterval() time.Duration {
	return w.announceInterval
}

func (w *Watcher) Name() string {
	return w.name
}

func (w *Watcher) NotifyStateChance() {
	if len(w.statechg) < cap(w.statechg) {
		w.statechg <- struct{}{}
	}
}

func (w *Watcher) CurrentState() interface{} {
	w.Lock()
	defer w.Unlock()

	s := &StateNotification{
		Protocol:  "io.choria.machine.watcher.timer.v1.state",
		Type:      w.Type(),
		Name:      w.name,
		Identity:  w.machine.Identity(),
		ID:        w.machine.InstanceID(),
		Version:   w.machine.Version(),
		Timestamp: w.machine.TimeStampSeconds(),
		Machine:   w.machine.Name(),
		State:     stateNames[w.state],
		Timer:     w.time,
	}

	return s
}
