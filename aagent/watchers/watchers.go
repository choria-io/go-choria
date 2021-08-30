package watchers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/choria-io/go-choria/aagent/model"
	"github.com/choria-io/go-choria/internal/util"
	"github.com/nats-io/jsm.go"
	"github.com/tidwall/gjson"
)

type State int

// Machine is a Choria Machine
type Machine interface {
	model.Machine
	Watchers() []*WatcherDef
}

// Manager manages all the defined watchers in a specific machine
// implements machine.WatcherManager
type Manager struct {
	watchers map[string]model.Watcher
	machine  Machine

	ctx    context.Context
	cancel func()

	sync.Mutex
}

var (
	plugins map[string]model.WatcherConstructor

	mu sync.Mutex
)

// RegisterWatcherPlugin registers a new type of watcher
func RegisterWatcherPlugin(name string, plugin model.WatcherConstructor) error {
	mu.Lock()
	defer mu.Unlock()

	if plugins == nil {
		plugins = map[string]model.WatcherConstructor{}
	}

	_, exit := plugins[plugin.Type()]
	if exit {
		return fmt.Errorf("plugin %q already exist", plugin.Type())
	}

	plugins[plugin.Type()] = plugin

	util.BuildInfo().RegisterMachineWatcher(name)

	return nil
}

func New(ctx context.Context) *Manager {
	m := &Manager{
		watchers: make(map[string]model.Watcher),
	}

	m.ctx, m.cancel = context.WithCancel(ctx)

	return m
}

func ParseWatcherState(state []byte) (interface{}, error) {
	r := gjson.GetBytes(state, "protocol")
	if !r.Exists() {
		return nil, fmt.Errorf("no protocol header in state json")
	}

	proto := r.String()
	var plugin model.WatcherConstructor

	mu.Lock()
	for _, w := range plugins {
		if w.EventType() == proto {
			plugin = w
		}
	}
	mu.Unlock()

	if plugin == nil {
		return nil, fmt.Errorf("unknown event type %q", proto)
	}

	return plugin.UnmarshalNotification(state)
}

// Delete gets called before a watcher is being deleted after
// its files were removed from disk
func (m *Manager) Delete() {
	m.machine.Infof(m.machine.Name(), "Stopping manager")
	m.cancel()

	m.Lock()
	defer m.Unlock()

	for _, w := range m.watchers {
		w.Delete()
	}
}

// JetStreamConnection is a NATS connection for accessing the JetStream API
func (m *Manager) JetStreamConnection() (*jsm.Manager, error) {
	m.Lock()
	defer m.Unlock()

	return m.machine.JetStreamConnection()
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
func (m *Manager) AddWatcher(w model.Watcher) error {
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
		err = w.ParseAnnounceInterval()
		if err != nil {
			return fmt.Errorf("could not create %s watcher '%s': %s", w.Type, w.Name, err)
		}

		m.machine.Infof("manager", "Starting %s watcher %s", w.Type, w.Name)

		var watcher model.Watcher
		var err error
		var ok bool

		mu.Lock()
		plugin, known := plugins[w.Type]
		mu.Unlock()
		if !known {
			return fmt.Errorf("unknown watcher '%s'", w.Type)
		}

		wi, err := plugin.New(m.machine, w.Name, w.StateMatch, w.FailTransition, w.SuccessTransition, w.Interval, w.AnnounceDuration, w.Properties)
		if err != nil {
			return fmt.Errorf("could not create %s watcher '%s': %s", w.Type, w.Name, err)
		}

		watcher, ok = wi.(model.Watcher)
		if !ok {
			return fmt.Errorf("%q watcher is not a valid watcher", w.Type)
		}

		err = m.AddWatcher(watcher)
		if err != nil {
			return err
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

func (m *Manager) announceWatcherState(ctx context.Context, wg *sync.WaitGroup, w model.Watcher) {
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
