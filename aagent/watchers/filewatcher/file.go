package filewatcher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/pkg/errors"
)

type State int

const (
	Unknown State = iota
	Error
	Skipped
	Unchanged
	Changed
)

var stateNames = map[State]string{
	Unknown:   "unknown",
	Error:     "error",
	Skipped:   "skipped",
	Unchanged: "unchanged",
	Changed:   "changed",
}

type Machine interface {
	State() string
	Name() string
	Identity() string
	InstanceID() string
	Version() string
	TimeStampSeconds() int64
	Directory() string
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
	path             string
	machine          Machine
	mtime            time.Time
	initial          bool
	interval         time.Duration
	announceInterval time.Duration
	statechg         chan struct{}
	previous         State
	lastAnnounce     time.Time

	sync.Mutex
}

func New(machine Machine, name string, states []string, failEvent string, successEvent string, interval string, ai time.Duration, properties map[string]interface{}) (watcher *Watcher, err error) {
	w := &Watcher{
		name:             name,
		successEvent:     successEvent,
		failEvent:        failEvent,
		states:           states,
		machine:          machine,
		interval:         5 * time.Second,
		announceInterval: ai,
		statechg:         make(chan struct{}, 1),
	}

	err = w.setProperties(properties)
	if err != nil {
		return nil, errors.Wrap(err, "could not set properties")
	}

	if interval != "" {
		w.interval, err = time.ParseDuration(interval)
		if err != nil {
			return nil, errors.Wrap(err, "invalid interval")
		}
	}

	if w.interval < 500*time.Millisecond {
		return nil, errors.Errorf("interval %v is too small", w.interval)
	}

	return w, err
}

func (w *Watcher) Type() string {
	return "file"
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

func (w *Watcher) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	w.machine.Infof(w.name, "file watcher for %s starting", w.path)

	tick := time.NewTicker(w.interval)

	if w.initial {
		stat, err := os.Stat(w.path)
		if err == nil {
			w.mtime = stat.ModTime()
		}
	}

	for {
		select {
		case <-tick.C:
			w.performWatch(ctx)

		case <-w.statechg:
			w.performWatch(ctx)

		case <-ctx.Done():
			tick.Stop()
			w.machine.Infof(w.name, "Stopping on context interrupt")
			return
		}
	}
}

func (w *Watcher) performWatch(ctx context.Context) {
	state, err := w.watch()
	err = w.handleCheck(state, err)
	if err != nil {
		w.machine.Errorf(w.name, "could not handle watcher event: %s", err)
	}
}

func (w *Watcher) CurrentState() interface{} {
	w.Lock()
	defer w.Unlock()

	s := &StateNotification{
		Protocol:        "io.choria.machine.watcher.file.v1.state",
		Type:            "file",
		Name:            w.name,
		Identity:        w.machine.Identity(),
		ID:              w.machine.InstanceID(),
		Version:         w.machine.Version(),
		Timestamp:       w.machine.TimeStampSeconds(),
		Machine:         w.machine.Name(),
		Path:            w.path,
		PreviousOutcome: stateNames[w.previous],
	}

	return s
}

func (w *Watcher) setPreviousState(s State) {
	w.Lock()
	defer w.Unlock()

	w.previous = s
}

func (w *Watcher) handleCheck(s State, err error) error {
	w.machine.Debugf(w.name, "handling check for %s %v %v", w.path, s, err)

	w.setPreviousState(s)

	switch s {
	case Error:
		w.machine.NotifyWatcherState(w.name, w.CurrentState())
		return w.machine.Transition(w.failEvent)

	case Changed:
		w.machine.NotifyWatcherState(w.name, w.CurrentState())
		return w.machine.Transition(w.successEvent)

	case Unchanged:
		// not notifying, regular announces happen

	case Skipped, Unknown:
		// clear the time so that next time after once being skipped or unknown
		// it will treat the file as not seen before and detect changes, but if
		// its set to do initial check it specifically will not do that because
		// the behavior of the first run in that case would be to only wait for
		// future changes, this retains that behavior on becoming valid again
		if !w.initial {
			w.mtime = time.Time{}
		}
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

func (w *Watcher) watch() (state State, err error) {
	if !w.shouldCheck() {
		return Skipped, nil
	}

	stat, err := os.Stat(w.path)
	if err != nil {
		w.mtime = time.Time{}

		return Error, fmt.Errorf("does not exist")
	}

	if stat.ModTime().After(w.mtime) {
		w.mtime = stat.ModTime()
		return Changed, nil
	}

	return Unchanged, err
}

func (w *Watcher) setProperties(p map[string]interface{}) error {
	path, ok := p["path"]
	if !ok {
		return fmt.Errorf("path is required")
	}

	w.path, ok = path.(string)
	if !ok {
		return fmt.Errorf("path should be a string")
	}

	if !filepath.IsAbs(w.path) {
		w.path = filepath.Join(w.machine.Directory(), w.path)
	}

	initial, ok := p["gather_initial_state"]
	if ok {
		w.initial, ok = initial.(bool)
		if !ok {
			return fmt.Errorf("gather_initial_state should be bool")
		}
	}

	return nil
}
