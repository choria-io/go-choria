package filewatcher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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
	Unknown State = iota
	Error
	Skipped
	Unchanged
	Changed

	wtype   = "file"
	version = "v1"
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
	machine    model.Machine
	previous   State
	interval   time.Duration
	mtime      time.Time
	properties *Properties
	mu         *sync.Mutex
}

func New(machine model.Machine, name string, states []string, failEvent string, successEvent string, interval string, ai time.Duration, properties map[string]interface{}) (interface{}, error) {
	var err error

	fw := &Watcher{
		properties: &Properties{},
		name:       name,
		machine:    machine,
		interval:   5 * time.Second,
		mu:         &sync.Mutex{},
	}

	fw.Watcher, err = watcher.NewWatcher(name, wtype, ai, states, machine, failEvent, successEvent)
	if err != nil {
		return nil, err
	}

	err = fw.setProperties(properties)
	if err != nil {
		return nil, fmt.Errorf("could not set properties: %s", err)
	}

	if !filepath.IsAbs(fw.properties.Path) {
		fw.properties.Path = filepath.Join(fw.machine.Directory(), fw.properties.Path)
	}

	if interval != "" {
		fw.interval, err = iu.ParseDuration(interval)
		if err != nil {
			return nil, fmt.Errorf("invalid interval: %s", err)
		}
	}

	if fw.interval < 500*time.Millisecond {
		return nil, fmt.Errorf("interval %v is too small", fw.interval)
	}

	return fw, err
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

func (w *Watcher) performWatch(_ context.Context) {
	state, err := w.watch()
	err = w.handleCheck(state, err)
	if err != nil {
		w.Errorf("could not handle watcher event: %s", err)
	}
}

func (w *Watcher) CurrentState() interface{} {
	w.mu.Lock()
	defer w.mu.Unlock()

	s := &StateNotification{
		Event:           event.New(w.name, wtype, version, w.machine),
		Path:            w.properties.Path,
		PreviousOutcome: stateNames[w.previous],
	}

	return s
}

func (w *Watcher) setPreviousState(s State) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.previous = s
}

func (w *Watcher) handleCheck(s State, err error) error {
	w.Debugf("Handling check for %s %v %v", w.properties.Path, s, err)

	w.setPreviousState(s)

	switch s {
	case Error:
		w.NotifyWatcherState(w.CurrentState())
		return w.FailureTransition()

	case Changed:
		w.NotifyWatcherState(w.CurrentState())
		return w.SuccessTransition()

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
	if w.properties == nil {
		w.properties = &Properties{}
	}

	err := util.ParseMapStructure(props, w.properties)
	if err != nil {
		return err
	}

	return w.validate()
}
