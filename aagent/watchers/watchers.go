package watchers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/choria-io/go-choria/aagent/watchers/execwatcher"
	"github.com/choria-io/go-choria/aagent/watchers/filewatcher"
	"github.com/choria-io/go-choria/aagent/watchers/homekit"
	"github.com/choria-io/go-choria/aagent/watchers/nagioswatcher"
	"github.com/choria-io/go-choria/aagent/watchers/schedulewatcher"
)

type State int

const (
	Error State = iota
	Skipped
	Unchanged
	Changed
)

// Watcher is anything that can be used to watch the system for events
type Watcher interface {
	Name() string
	Type() string
	Run(context.Context, *sync.WaitGroup)
	NotifyStateChance()
	CurrentState() interface{}
	AnnounceInterval() time.Duration
	Delete()
}

// Machine is a Choria Machine
type Machine interface {
	Name() string
	State() string
	Directory() string
	Transition(t string, args ...interface{}) error
	NotifyWatcherState(string, interface{})
	Watchers() []*WatcherDef
	Identity() string
	InstanceID() string
	Version() string
	TimeStampSeconds() int64
	TextFileDirectory() string
	OverrideData() ([]byte, error)
	Debugf(name string, format string, args ...interface{})
	Infof(name string, format string, args ...interface{})
	Errorf(name string, format string, args ...interface{})
}

// Manager manages all the defined watchers in a specific machine
// implements machine.WatcherManager
type Manager struct {
	watchers map[string]Watcher
	machine  Machine
	sync.Mutex
}

func New() *Manager {
	return &Manager{
		watchers: make(map[string]Watcher),
	}
}

// Delete gets called before a watcher is being deleted after
// its files were removed from disk
func (m *Manager) Delete() {
	m.Lock()
	defer m.Unlock()

	for _, w := range m.watchers {
		w.Delete()
	}
}

// SetMachine supplies the machine this manager will manage
func (m *Manager) SetMachine(t interface{}) (err error) {
	machine, ok := t.(Machine)
	if !ok {
		return fmt.Errorf("supplied machine does not implement watchers.Machine")
	}

	m.machine = machine

	return nil
}

// AddWatcher adds a watcher to a managed machine
func (m *Manager) AddWatcher(w Watcher) error {
	m.Lock()
	defer m.Unlock()

	_, ok := m.watchers[w.Name()]
	if ok {
		m.machine.Errorf("manager", "Already have a watcher %s", w.Name())
		return fmt.Errorf("watcher %s already exist", w.Name())
	}

	m.watchers[w.Name()] = w

	return nil
}

// WatcherState retrieves the current status for a given watcher, boolean result is false for unknown watchers
func (m *Manager) WatcherState(watcher string) (interface{}, bool) {
	m.Lock()
	defer m.Unlock()
	w, ok := m.watchers[watcher]
	if !ok {
		return nil, false
	}

	return w.CurrentState(), true
}

func (m *Manager) configureWatchers() (err error) {
	for _, w := range m.machine.Watchers() {
		w.ParseAnnounceInterval()

		m.machine.Infof("manager", "Starting %s watcher %s", w.Type, w.Name)

		switch w.Type {
		case "file":
			watcher, err := filewatcher.New(m.machine, w.Name, w.StateMatch, w.FailTransition, w.SuccessTransition, w.Interval, w.announceDuration, w.Properties)
			if err != nil {
				return fmt.Errorf("could not create file watcher '%s': %s", w.Name, err)
			}

			err = m.AddWatcher(watcher)
			if err != nil {
				return err
			}

		case "exec":
			watcher, err := execwatcher.New(m.machine, w.Name, w.StateMatch, w.FailTransition, w.SuccessTransition, w.Interval, w.announceDuration, w.Properties)
			if err != nil {
				return fmt.Errorf("could not create exec watcher '%s': %s", w.Name, err)
			}

			err = m.AddWatcher(watcher)
			if err != nil {
				return err
			}

		case "schedule":
			watcher, err := schedulewatcher.New(m.machine, w.Name, w.StateMatch, w.FailTransition, w.SuccessTransition, w.Interval, w.announceDuration, w.Properties)
			if err != nil {
				return fmt.Errorf("could not create schedule watcher '%s': %s", w.Name, err)
			}

			err = m.AddWatcher(watcher)
			if err != nil {
				return err
			}

		case "nagios":
			watcher, err := nagioswatcher.New(m.machine, w.Name, w.StateMatch, w.FailTransition, w.SuccessTransition, w.Interval, w.announceDuration, w.Properties)
			if err != nil {
				return fmt.Errorf("could not create exec watcher '%s': %s", w.Name, err)
			}

			err = m.AddWatcher(watcher)
			if err != nil {
				return err
			}

		case "homekit":
			watcher, err := homekit.New(m.machine, w.Name, w.StateMatch, w.FailTransition, w.SuccessTransition, w.announceDuration, w.Properties)
			if err != nil {
				return fmt.Errorf("could not create homekit watcher '%s': %s", w.Name, err)
			}

			err = m.AddWatcher(watcher)
			if err != nil {
				return err
			}

		default:
			return fmt.Errorf("unknown watcher '%s'", w.Type)
		}
	}

	return nil
}

// Run starts all the defined watchers and periodically announce
// their state based on AnnounceInterval
func (m *Manager) Run(ctx context.Context, wg *sync.WaitGroup) error {
	if m.machine == nil {
		return fmt.Errorf("manager requires a machine to manage")
	}

	err := m.configureWatchers()
	if err != nil {
		return err
	}

	for _, watcher := range m.watchers {
		wg.Add(1)
		go watcher.Run(ctx, wg)

		if watcher.AnnounceInterval() > 0 {
			wg.Add(1)
			go m.announceWatcherState(ctx, wg, watcher)
		}
	}

	return nil
}

func (m *Manager) announceWatcherState(ctx context.Context, wg *sync.WaitGroup, w Watcher) {
	defer wg.Done()

	announceTick := time.NewTicker(w.AnnounceInterval())

	for {
		select {
		case <-announceTick.C:
			m.machine.NotifyWatcherState(w.Name(), w.CurrentState())
		case <-ctx.Done():
			m.machine.Infof("manager", "Stopping on context interrupt")
			return
		}
	}
}

// NotifyStateChance implements machine.WatcherManager
func (m *Manager) NotifyStateChance() {
	m.Lock()
	defer m.Unlock()

	for _, watcher := range m.watchers {
		watcher.NotifyStateChance()
	}
}
