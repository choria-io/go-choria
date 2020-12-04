package filewatcher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/choria-io/go-choria/aagent/util"
	"github.com/choria-io/go-choria/aagent/watchers/event"
	"github.com/choria-io/go-choria/aagent/watchers/watcher"
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

type Properties struct {
	Path    string
	Initial bool `mapstructure:"gather_initial_state"`
}

type Watcher struct {
	*watcher.Watcher

	name       string
	machine    watcher.Machine
	previous   State
	interval   time.Duration
	mtime      time.Time
	properties *Properties
}

func New(machine watcher.Machine, name string, states []string, failEvent string, successEvent string, interval string, ai time.Duration, properties map[string]interface{}) (*Watcher, error) {
	var err error

	w := &Watcher{
		properties: &Properties{},
		name:       name,
		machine:    machine,
		interval:   5 * time.Second,
	}

	w.Watcher, err = watcher.NewWatcher(name, "file", ai, states, machine, failEvent, successEvent)
	if err != nil {
		return nil, err
	}

	err = w.setProperties(properties)
	if err != nil {
		return nil, errors.Wrap(err, "could not set properties")
	}

	if !filepath.IsAbs(w.properties.Path) {
		w.properties.Path = filepath.Join(w.machine.Directory(), w.properties.Path)
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

func (w *Watcher) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	w.Infof("file watcher for %s starting", w.properties.Path)

	tick := time.NewTicker(w.interval)

	if w.properties.Initial {
		stat, err := os.Stat(w.properties.Path)
		if err == nil {
			w.mtime = stat.ModTime()
		}
	}

	for {
		select {
		case <-tick.C:
			w.performWatch(ctx)

		case <-w.Watcher.StateChangeC():
			w.performWatch(ctx)

		case <-ctx.Done():
			tick.Stop()
			w.Infof("Stopping on context interrupt")
			return
		}
	}
}

func (w *Watcher) performWatch(ctx context.Context) {
	state, err := w.watch()
	err = w.handleCheck(state, err)
	if err != nil {
		w.Errorf("could not handle watcher event: %s", err)
	}
}

func (w *Watcher) CurrentState() interface{} {
	w.Lock()
	defer w.Unlock()

	s := &StateNotification{
		Event:           event.New(w.name, "file", "v1", w.machine),
		Path:            w.properties.Path,
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
	w.Debugf("Handling check for %s %v %v", w.properties.Path, s, err)

	w.setPreviousState(s)

	switch s {
	case Error:
		w.NotifyWatcherState(w.name, w.CurrentState())
		return w.Transition(w.Watcher.FailEvent())

	case Changed:
		w.NotifyWatcherState(w.name, w.CurrentState())
		return w.Transition(w.Watcher.SuccessEvent())

	case Unchanged:
		// not notifying, regular announces happen

	case Skipped, Unknown:
		// clear the time so that next time after once being skipped or unknown
		// it will treat the file as not seen before and detect changes, but if
		// its set to do initial check it specifically will not do that because
		// the behavior of the first run in that case would be to only wait for
		// future changes, this retains that behavior on becoming valid again
		if !w.properties.Initial {
			w.mtime = time.Time{}
		}
	}

	return nil
}

func (w *Watcher) watch() (state State, err error) {
	if !w.Watcher.ShouldWatch() {
		return Skipped, nil
	}

	stat, err := os.Stat(w.properties.Path)
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

func (w *Watcher) validate() error {
	if w.properties.Path == "" {
		return fmt.Errorf("path is required")
	}

	return nil
}

func (w *Watcher) setProperties(props map[string]interface{}) error {
	err := util.ParseMapStructure(props, w.properties)
	if err != nil {
		return err
	}

	return w.validate()
}
