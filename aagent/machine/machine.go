package machine

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"

	"github.com/choria-io/go-choria/aagent/watchers"
	"github.com/go-yaml/yaml"

	"github.com/looplab/fsm"
)

// Machine is a autonomous agent implemented as a Finite State Machine and hosted within Choria Server
type Machine struct {
	// MachineName is the unique name for this machine
	MachineName string `json:"name" yaml:"name"`

	// MachineVersion is the semver compliant version for the running machine
	MachineVersion string `json:"version" yaml:"version"`

	// InitialState is the state this machine starts in when it first starts
	InitialState string `json:"initial_state" yaml:"initial_state"`

	// Transitions contain a list of valid events of transitions this machine can move through
	Transitions []*Transition `json:"transitions" yaml:"transitions"`

	// WatcherDefs contains all the watchers that can interact with the system
	WatcherDefs []*watchers.WatcherDef `json:"watchers" yaml:"watchers"`

	manager   WatcherManager
	fsm       *fsm.FSM
	notifiers []NotificationService
	directory string

	ctx    context.Context
	cancel context.CancelFunc
	sync.Mutex
}

// Transition describes a transition event within the Finite State Machine
type Transition struct {
	// Name is the name for the transition shown in logs and graphs
	Name string `json:"name" yaml:"name"`

	// From is a list of valid state names from where this transition event is valid
	From []string `json:"from" yaml:"from"`

	// Destination is the name of the target state this event will move the machine into
	Destination string `json:"destination" yaml:"destination"`
}

// WatcherManager manages watchers
type WatcherManager interface {
	Run(context.Context, *sync.WaitGroup) error
	NotifyStateChance()
	SetMachine(interface{}) error
}

func FromDir(dir string, manager WatcherManager) (m *Machine, err error) {
	mpath := dir + "/" + "machine.yaml"
	_, err = os.Stat(mpath)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot read %s", mpath)
	}

	m, err = FromYAML(mpath, manager)
	m.directory = dir

	return m, err
}

// FromYAML loads a macine from a YAML definition
func FromYAML(file string, manager WatcherManager) (m *Machine, err error) {
	afile, err := filepath.Abs(file)
	if err != nil {
		return nil, errors.Wrapf(err, "could not determine absolute path for %s", file)
	}

	f, err := ioutil.ReadFile(afile)
	if err != nil {
		return nil, err
	}

	m = &Machine{}
	err = yaml.Unmarshal(f, m)
	if err != nil {
		return nil, err
	}

	m.notifiers = []NotificationService{}
	m.manager = manager
	m.directory = filepath.Dir(afile)

	err = m.manager.SetMachine(m)
	if err != nil {
		return nil, errors.Wrap(err, "could not register with manager")
	}

	err = m.Setup()
	if err != nil {
		return nil, err
	}

	return m, nil
}

// Returns the directory where the machine definition is, "" when unknown
func (m *Machine) Directory() string {
	return m.directory
}

// Watchers retrieves the watcher definitions
func (m *Machine) Watchers() []*watchers.WatcherDef {
	return m.WatcherDefs
}

// Name is the name of the machine
func (m *Machine) Name() string {
	return m.MachineName
}

// Graph produce a dot graph of the fsm
func (m *Machine) Graph() string {
	return fsm.Visualize(m.fsm)
}

func (m *Machine) buildFSM() error {
	events := fsm.Events{}

	for _, t := range m.Transitions {
		events = append(events, fsm.EventDesc{
			Dst:  t.Destination,
			Src:  t.From,
			Name: t.Name,
		})
	}

	if len(events) == 0 {
		return fmt.Errorf("no transitions found")
	}

	f := fsm.NewFSM(m.InitialState, events, fsm.Callbacks{
		"enter_state": func(e *fsm.Event) {
			for _, notifier := range m.notifiers {
				m.manager.NotifyStateChance()

				err := notifier.NotifyPostTransition(&TransitionNotification{
					Event:   e.Event,
					From:    e.Src,
					To:      e.Dst,
					Machine: m.MachineName,
				})
				if err != nil {
					m.Errorf("machine", "Could not publish event notification for %s: %s", e.Event, err)
				}
			}
		},
	})

	m.fsm = f

	return nil
}

// Validate performs basic validation on the machine settings
func (m *Machine) Validate() error {
	if m.MachineName == "" {
		return fmt.Errorf("a machine name is required")
	}

	if m.MachineVersion == "" {
		return fmt.Errorf("a machine version is required")
	}

	if m.InitialState == "" {
		return fmt.Errorf("an initial state is required")
	}

	if len(m.Transitions) == 0 {
		return fmt.Errorf("no transitions defined")
	}

	if len(m.WatcherDefs) == 0 {
		return fmt.Errorf("no watchers defined")
	}

	for _, w := range m.Watchers() {
		err := w.ParseAnnounceInterval()
		if err != nil {
			return err
		}
	}

	return nil
}

// Setup validates and prepares the machine for execution
func (m *Machine) Setup() error {
	err := m.Validate()
	if err != nil {
		return errors.Wrapf(err, "validation failed")
	}

	return m.buildFSM()
}

// Start runs the machine in the background
func (m *Machine) Start(ctx context.Context, wg *sync.WaitGroup) error {
	m.ctx, m.cancel = context.WithCancel(ctx)

	return m.manager.Run(ctx, wg)
}

// Stop stops a running machine by canceling its context
func (m *Machine) Stop() {
	if m.cancel != nil {
		m.Infof("runner", "Stopping")
		m.cancel()
	}
}

// State returns the current state of the machine
func (m *Machine) State() string {
	return m.fsm.Current()
}

// Transition performs the machine transition as defined by event t
func (m *Machine) Transition(t string, args ...interface{}) error {
	m.Lock()
	defer m.Unlock()

	if t == "" {
		return nil
	}

	if m.fsm.Can(t) {
		m.fsm.Event(t, args...)
	} else {
		m.Warnf("machine", "Could not fire '%s' event while in %s", t, m.fsm.Current())
	}

	return nil
}
