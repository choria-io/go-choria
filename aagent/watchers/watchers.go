package watchers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/choria-io/go-choria/aagent/watchers/execwatcher"
	"github.com/choria-io/go-choria/aagent/watchers/filewatcher"
	"github.com/choria-io/go-choria/aagent/watchers/schedulewatcher"
	"github.com/pkg/errors"
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

// SetMachine supplies the machine this manager will manage
func (m *Manager) SetMachine(t interface{}) (err error) {
	machine, ok := t.(Machine)
	if !ok {
		return errors.New("supplied machine does not implement watchers.Machine")
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

func (m *Manager) configureWatchers() (err error) {
	for _, w := range m.machine.Watchers() {
		w.ParseAnnounceInterval()

		m.machine.Infof("manager", "Starting %s watcher %s", w.Type, w.Name)

		switch w.Type {
		case "file":
			watcher, err := filewatcher.New(m.machine, w.Name, w.StateMatch, w.FailTransition, w.SuccessTransition, w.Interval, w.announceDuration, w.Properties)
			if err != nil {
				return errors.Wrapf(err, "could not create file watcher '%s'", w.Name)
			}

			err = m.AddWatcher(watcher)
			if err != nil {
				return err
			}

		case "exec":
			watcher, err := execwatcher.New(m.machine, w.Name, w.StateMatch, w.FailTransition, w.SuccessTransition, w.Interval, w.announceDuration, w.Properties)
			if err != nil {
				return errors.Wrapf(err, "could not create exec watcher '%s'", w.Name)
			}

			err = m.AddWatcher(watcher)
			if err != nil {
				return err
			}

		case "schedule":
			watcher, err := schedulewatcher.New(m.machine, w.Name, w.StateMatch, w.FailTransition, w.SuccessTransition, w.Interval, w.announceDuration, w.Properties)
			if err != nil {
				return errors.Wrapf(err, "could not create schedule watcher '%s'", w.Name)
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
		return errors.Wrap(err, "could not configure watchers")
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
