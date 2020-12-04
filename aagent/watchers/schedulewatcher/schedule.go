package schedulewatcher

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
	Unknown State = iota
	Off
	On
	Skipped
)

var stateNames = map[State]string{
	Unknown: "unknown",
	Off:     "off",
	On:      "on",
	Skipped: "skipped",
}

type properties struct {
	Duration  time.Duration
	Schedules []string
}

type Watcher struct {
	*watcher.Watcher
	properties *properties
	name       string
	machine    watcher.Machine
	items      []*scheduleItem

	// each item sends a 1 or -1 into this to increment or decrement the counter
	// when the ctr is > 0 the switch should be on, this handles multiple schedules
	// overlapping and keeping it on for longer than a single schedule would be
	ctrq chan int
	ctr  int

	state         State
	previousState State

	sync.Mutex
}

func New(machine watcher.Machine, name string, states []string, failEvent string, successEvent string, interval string, ai time.Duration, properties map[string]interface{}) (*Watcher, error) {
	var err error

	w := &Watcher{
		name:    name,
		machine: machine,
		ctrq:    make(chan int, 1),
		ctr:     0,
	}

	w.Watcher, err = watcher.NewWatcher(name, "schedule", ai, states, machine, failEvent, successEvent)
	if err != nil {
		return nil, err
	}

	err = w.setProperties(properties)
	if err != nil {
		return nil, fmt.Errorf("could not set properties: %s", err)
	}

	return w, nil
}

func (w *Watcher) watchSchedule(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case i := <-w.ctrq:
			w.Infof("Handling state change counter %v while ctr=%v", i, w.ctr)
			w.Lock()

			w.ctr = w.ctr + i

			// shouldn't happen but lets handle it
			if w.ctr < 0 {
				w.ctr = 0
			}

			if w.ctr == 0 {
				w.Infof("State going off due to ctr change to 0")
				w.state = Off
			} else {
				w.Infof("State going on due to ctr change of %v", i)
				w.state = On
			}

			w.Unlock()

		case <-ctx.Done():
			return
		}
	}
}

func (w *Watcher) setPreviousState(s State) {
	w.Lock()
	defer w.Unlock()

	w.previousState = s
}

func (w *Watcher) watch() (err error) {
	if !w.ShouldWatch() {
		w.setPreviousState(Skipped)
		return nil
	}

	notifyf := func() error {
		w.setPreviousState(w.state)

		switch w.state {
		case Off, Unknown:
			w.NotifyWatcherState(w.name, w.CurrentState())
			return w.Transition(w.FailEvent())

		case On:
			w.setPreviousState(w.state)
			w.NotifyWatcherState(w.name, w.CurrentState())
			return w.Transition(w.SuccessEvent())

		case Skipped:
			// not doing anything when we aren't eligible, regular announces happen

		}

		return nil
	}

	// nothing changed
	if w.previousState == w.state {
		return nil
	}

	// previously skipped this means it should have become viable via state matchers
	// since last check so we need to fire triggers
	if w.previousState == Skipped {
		return notifyf()
	}

	return notifyf()
}

func (w *Watcher) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	w.Infof("schedule watcher starting with %d items", len(w.items))

	tick := time.NewTicker(500 * time.Millisecond)

	wg.Add(1)
	go w.watchSchedule(ctx, wg)

	for _, item := range w.items {
		wg.Add(1)
		go item.start(ctx, wg)
	}

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

func (w *Watcher) setProperties(props map[string]interface{}) error {
	if w.properties == nil {
		w.properties = &properties{}
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

	return w.validate()
}

func (w *Watcher) CurrentState() interface{} {
	w.Lock()
	defer w.Unlock()

	s := &StateNotification{
		Event: event.New(w.name, "schedule", "v1", w.machine),
		State: stateNames[w.state],
	}

	return s
}
