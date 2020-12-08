package timerwatcher

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/choria-io/go-choria/aagent/util"
	"github.com/choria-io/go-choria/aagent/watchers/event"
	"github.com/choria-io/go-choria/aagent/watchers/watcher"
)

type State int

const (
	Stopped State = iota
	Running

	wtype   = "timer"
	version = "v1"
)

var stateNames = map[State]string{
	Running: "running",
	Stopped: "stopped",
}

type properties struct {
	Timer time.Duration
}

type Watcher struct {
	*watcher.Watcher
	properties *properties

	name    string
	machine watcher.Machine
	state   State

	terminate   chan struct{}
	cancelTimer func()
	mu          *sync.Mutex
}

func New(machine watcher.Machine, name string, states []string, failEvent string, successEvent string, interval string, ai time.Duration, properties map[string]interface{}) (interface{}, error) {
	var err error

	tw := &Watcher{
		name:      name,
		machine:   machine,
		state:     0,
		terminate: make(chan struct{}),
		mu:        &sync.Mutex{},
	}

	tw.Watcher, err = watcher.NewWatcher(name, wtype, ai, states, machine, failEvent, successEvent)
	if err != nil {
		return nil, err
	}

	err = tw.setProperties(properties)
	if err != nil {
		return nil, fmt.Errorf("could not set properties: %s", err)
	}

	return tw, nil
}

func (w *Watcher) Delete() {
	close(w.terminate)
}

func (w *Watcher) forceTimerStop() {
	w.mu.Lock()
	cancel := w.cancelTimer
	w.mu.Unlock()

	if cancel != nil {
		w.Infof("Stopping timer early on state transition to %s", w.machine.State())
		cancel()
	}
}

func (w *Watcher) timeStart() {
	w.mu.Lock()
	cancel := w.cancelTimer
	w.mu.Unlock()

	if cancel != nil {
		w.Infof(w.name, "Timer was running, resetting to %v", w.properties.Timer)
		cancel()
	}

	go func() {
		timer := time.NewTimer(w.properties.Timer)
		ctx, cancel := context.WithCancel(context.Background())

		w.mu.Lock()
		w.state = Running
		w.cancelTimer = cancel
		w.mu.Unlock()

		select {
		case <-timer.C:
			w.mu.Lock()
			w.state = Stopped
			w.cancelTimer()
			w.cancelTimer = nil
			w.mu.Unlock()

			w.NotifyWatcherState(w.CurrentState())
			w.SuccessTransition()

		case <-ctx.Done():
			w.mu.Lock()
			w.cancelTimer = nil
			timer.Stop()
			w.state = Stopped
			w.mu.Unlock()

			w.NotifyWatcherState(w.CurrentState())

		case <-w.terminate:
			w.mu.Lock()
			w.cancelTimer = nil
			w.state = Stopped
			w.mu.Unlock()
			return
		}
	}()

	w.NotifyWatcherState(w.CurrentState())
}

func (w *Watcher) watch() {
	if !w.ShouldWatch() {
		w.Infof("Forcing timer off")
		w.forceTimerStop()
		return
	}

	w.Infof("Starting timer")
	w.timeStart()
}

func (w *Watcher) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	w.Infof("Timer watcher starting with %v timer", w.properties.Timer)

	// handle initial state
	w.watch()

	for {
		select {
		case <-w.StateChangeC():
			w.watch()
		case <-w.terminate:
			w.Infof("Handling terminate notification")
			return
		case <-ctx.Done():
			w.Infof("Stopping on context interrupt")
			return
		}
	}
}

func (w *Watcher) validate() error {
	if w.properties.Timer < time.Second {
		w.properties.Timer = time.Second
	}

	return nil
}

func (w *Watcher) setProperties(props map[string]interface{}) error {
	if w.properties == nil {
		w.properties = &properties{}
	}

	err := util.ParseMapStructure(props, w.properties)
	if err != nil {
		return err
	}

	return w.validate()
}

func (w *Watcher) CurrentState() interface{} {
	w.mu.Lock()
	defer w.mu.Unlock()

	s := &StateNotification{
		Event: event.New(w.name, wtype, version, w.machine),
		State: stateNames[w.state],
		Timer: w.properties.Timer,
	}

	return s
}
