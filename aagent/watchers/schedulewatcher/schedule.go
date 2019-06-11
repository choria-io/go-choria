package schedulewatcher

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
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

type Machine interface {
	State() string
	Name() string
	Identity() string
	InstanceID() string
	Version() string
	TimeStampSeconds() int64
	Transition(t string, args ...interface{}) error
	NotifyWatcherState(string, interface{})
	Debugf(name string, format string, args ...interface{})
	Infof(name string, format string, args ...interface{})
	Errorf(name string, format string, args ...interface{})
}

type Watcher struct {
	name             string
	states           []string
	failEvent        string
	successEvent     string
	machine          Machine
	duration         time.Duration
	announceInterval time.Duration
	lastAnnounce     time.Time
	schedules        []string
	items            []*scheduleItem
	statechg         chan struct{}

	// each item sends a 1 or -1 into this to increment or decrement the counter
	// when the ctr is > 0 the switch should be on, this handles multiple schedules
	// overlapping and keeping it on for longer than a single schedule would be
	ctrq chan int
	ctr  int

	state         State
	previousState State

	sync.Mutex
}

func New(machine Machine, name string, states []string, failEvent string, successEvent string, interval string, ai time.Duration, properties map[string]interface{}) (watcher *Watcher, err error) {
	w := &Watcher{
		name:             name,
		successEvent:     successEvent,
		failEvent:        failEvent,
		states:           states,
		machine:          machine,
		announceInterval: ai,
		statechg:         make(chan struct{}, 1),
		ctrq:             make(chan int, 1),
		ctr:              0,
	}

	err = w.setProperties(properties)
	if err != nil {
		return nil, errors.Wrap(err, "could not set properties")
	}

	return w, nil
}

func (w *Watcher) watchSchedule(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case i := <-w.ctrq:
			w.machine.Debugf(w.name, "Handling state change counter %v while ctr=%v", i, w.ctr)
			w.Lock()

			w.ctr = w.ctr + i

			// shouldnt happen but lets handle it
			if w.ctr < 0 {
				w.ctr = 0
			}

			if w.ctr == 0 {
				w.machine.Debugf(w.name, "State going off due to ctr change to 0")
				w.state = Off
			} else {
				w.machine.Debugf(w.name, "State going on due to ctr change of %v", i)
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
	if !w.shouldCheck() {
		w.setPreviousState(Skipped)
		return nil
	}

	notifyf := func() error {
		w.setPreviousState(w.state)

		switch w.state {
		case Off, Unknown:
			w.machine.NotifyWatcherState(w.name, w.CurrentState())
			return w.machine.Transition(w.failEvent)

		case On:
			w.setPreviousState(w.state)
			w.machine.NotifyWatcherState(w.name, w.CurrentState())
			return w.machine.Transition(w.successEvent)

		case Skipped:
			// not doing anything when we aren't eligable, regular announces happen

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

	w.machine.Infof(w.name, "schedule watcher starting with %d items", len(w.items))

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
				w.machine.Errorf(w.name, "Could not handle current scheduler state: %s", err)
			}

		case <-w.statechg:
			err := w.watch()
			if err != nil {
				w.machine.Errorf(w.name, "Could not handle current scheduler state: %s", err)
			}

		case <-ctx.Done():
			tick.Stop()
			w.machine.Infof(w.name, "Stopping on context interrupt")
			return
		}
	}
}

func (w *Watcher) Type() string {
	return "schedule"
}

func (w *Watcher) AnnounceInterval() time.Duration {
	return w.announceInterval
}

func (w *Watcher) Name() string {
	return w.name
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

func (w *Watcher) NotifyStateChance() {
	if len(w.statechg) < cap(w.statechg) {
		w.statechg <- struct{}{}
	}
}

func (w *Watcher) setProperties(p map[string]interface{}) (err error) {
	durationi, ok := p["duration"]
	if !ok {
		return fmt.Errorf("duration is required")
	}

	durationspec, ok := durationi.(string)
	if !ok {
		return fmt.Errorf("duration must be a string like 1h")
	}

	w.duration, err = time.ParseDuration(durationspec)
	if err != nil {
		return errors.Wrapf(err, "invalid duration %s", durationspec)
	}

	specs, ok := p["schedules"]
	if !ok {
		return fmt.Errorf("schedules is required")
	}

	speclist, ok := specs.([]interface{})
	if !ok {
		return fmt.Errorf("schedules must be a list of strings")
	}

	if len(speclist) == 0 {
		return fmt.Errorf("at least one schedule is required")
	}

	for _, specitem := range speclist {
		spec, ok := specitem.(string)
		if !ok {
			return fmt.Errorf("schedules must be a list of strings")
		}

		w.schedules = append(w.schedules, spec)

		item, err := newSchedItem(spec, w)
		if err != nil {
			return errors.Wrapf(err, "could not parse '%s'", spec)
		}

		w.items = append(w.items, item)
	}

	return nil
}

func (w *Watcher) CurrentState() interface{} {
	w.Lock()
	defer w.Unlock()

	s := &StateNotification{
		Protocol:  "io.choria.machine.watcher.schedule.v1.state",
		Type:      w.Type(),
		Name:      w.name,
		Identity:  w.machine.Identity(),
		ID:        w.machine.InstanceID(),
		Version:   w.machine.Version(),
		Timestamp: w.machine.TimeStampSeconds(),
		Machine:   w.machine.Name(),
		State:     stateNames[w.state],
	}

	return s
}
