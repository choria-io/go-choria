package machine

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"path/filepath"
	"sync"
	"time"

	"github.com/xeipuuv/gojsonschema"

	"github.com/ghodss/yaml"

	"github.com/choria-io/go-choria/aagent/watchers"
	"github.com/choria-io/go-choria/internal/util"

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

	// SplayStart causes a random sleep of maximum this many seconds before the machine starts
	SplayStart int `json:"splay_start" yaml:"splay_start"`

	instanceID       string
	identity         string
	directory        string
	manifest         string
	txtfileDir       string
	overridesFile    string
	choriaStatusFile string
	choriaStatusFreq int
	startTime        time.Time

	manager     WatcherManager
	fsm         *fsm.FSM
	notifiers   []NotificationService
	knownStates map[string]bool

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

	// Description is a human friendly description of the purpose of this transition
	Description string `json:"description" yaml:"description"`
}

// WatcherManager manages watchers
type WatcherManager interface {
	Run(context.Context, *sync.WaitGroup) error
	NotifyStateChance()
	SetMachine(interface{}) error
	WatcherState(watcher string) (interface{}, bool)
	Delete()
}

func yamlPath(dir string) string {
	return dir + "/" + "machine.yaml"
}

func FromDir(dir string, manager WatcherManager) (m *Machine, err error) {
	mpath := yamlPath(dir)

	if !util.FileExist(mpath) {
		return nil, fmt.Errorf("cannot read %s: %s", mpath, err)
	}

	m, err = FromYAML(mpath, manager)
	if err != nil {
		return nil, fmt.Errorf("could not load machine.yaml: %s", err)
	}

	m.directory, err = filepath.Abs(dir)

	return m, err
}

// FromYAML loads a machine from a YAML definition
func FromYAML(file string, manager WatcherManager) (m *Machine, err error) {
	afile, err := filepath.Abs(file)
	if err != nil {
		return nil, fmt.Errorf("could not determine absolute path for %s: %s", file, err)
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
	m.manifest = afile
	m.instanceID = m.UniqueID()
	m.knownStates = make(map[string]bool)

	err = m.manager.SetMachine(m)
	if err != nil {
		return nil, fmt.Errorf("could not register with manager: %s", err)
	}

	err = m.Setup()
	if err != nil {
		return nil, err
	}

	return m, nil
}

// ValidateDir validates a machine.yaml against the v1 schema
func ValidateDir(dir string) (validationErrors []string, err error) {
	mpath := yamlPath(dir)
	yml, err := ioutil.ReadFile(mpath)
	if err != nil {
		return nil, err
	}

	jbytes, err := yaml.YAMLToJSON(yml)
	if err != nil {
		return nil, fmt.Errorf("could not transform YAML to JSON: %s", err)
	}

	schemaLoader := gojsonschema.NewReferenceLoader("https://choria.io/schemas/choria/machine/v1/manifest.json")
	documentLoader := gojsonschema.NewBytesLoader(jbytes)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return nil, fmt.Errorf("could not perform schema validation: %s", err)
	}

	if result.Valid() {
		return []string{}, nil
	}

	validationErrors = []string{}
	for _, desc := range result.Errors() {
		validationErrors = append(validationErrors, desc.String())
	}

	return validationErrors, nil
}

// SetChoriaStatusFile sets the path and write frequency of the choria status file
func (m *Machine) SetChoriaStatusFile(f string, freq int) {
	m.Lock()
	defer m.Unlock()

	m.choriaStatusFile = f
	m.choriaStatusFreq = freq
}

// ChoriaStatusFile is the path to and write frequency of the choria status file, empty when not set
func (m *Machine) ChoriaStatusFile() (string, int) {
	m.Lock()
	defer m.Unlock()

	return m.choriaStatusFile, m.choriaStatusFreq
}

// SetIdentity sets the identity of the node hosting this machine
func (m *Machine) SetIdentity(id string) {
	m.Lock()
	defer m.Unlock()

	m.identity = id
}

func (m *Machine) SetTextFileDirectory(d string) {
	m.Lock()
	defer m.Unlock()

	m.txtfileDir = d
}

func (m *Machine) TextFileDirectory() string {
	m.Lock()
	defer m.Unlock()

	return m.txtfileDir
}

func (m *Machine) SetOverridesFile(f string) {
	m.Lock()
	defer m.Unlock()

	m.overridesFile = f
}

func (m *Machine) OverrideData() ([]byte, error) {
	m.Lock()
	source := m.overridesFile
	m.Unlock()

	if source == "" {
		return []byte{}, nil
	}

	// todo: maybe some caching here
	return ioutil.ReadFile(source)
}

// Watchers retrieves the watcher definitions
func (m *Machine) Watchers() []*watchers.WatcherDef {
	return m.WatcherDefs
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
			for i, notifier := range m.notifiers {
				if i == 0 {
					m.manager.NotifyStateChance()
				}

				err := notifier.NotifyPostTransition(&TransitionNotification{
					Protocol:   "io.choria.machine.v1.transition",
					Identity:   m.Identity(),
					ID:         m.InstanceID(),
					Version:    m.Version(),
					Timestamp:  m.TimeStampSeconds(),
					Machine:    m.MachineName,
					Transition: e.Event,
					FromState:  e.Src,
					ToState:    e.Dst,
					Info:       m,
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

		err = w.ValidateStates(m.KnownStates())
		if err != nil {
			return err
		}

		err = w.ValidateTransitions(m.KnownTransitions())
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
		return fmt.Errorf("validation failed: %s", err)
	}

	return m.buildFSM()
}

// Start runs the machine in the background
func (m *Machine) Start(ctx context.Context, wg *sync.WaitGroup) (started chan struct{}) {
	m.ctx, m.cancel = context.WithCancel(ctx)

	started = make(chan struct{})

	runf := func() {
		if m.SplayStart > 0 {
			sleep := time.Duration(rand.Intn(m.SplayStart)) * time.Second
			m.Infof(m.MachineName, "Sleeping %v before starting Autonomous Agent", sleep)

			t := time.NewTimer(sleep)

			select {
			case <-t.C:
			case <-m.ctx.Done():
				m.Infof(m.MachineName, "Exiting on context interrupt")
				return
			}
		}

		m.Infof(m.MachineName, "Starting Choria Machine %s version %s from %s", m.MachineName, m.MachineVersion, m.directory)
		m.startTime = time.Now().UTC()

		err := m.manager.Run(m.ctx, wg)
		if err != nil {
			m.Errorf(m.MachineName, "Could not start manager: %s", err)
		}

		started <- struct{}{}
	}

	go runf()

	return started
}

// Delete deletes a running machine by canceling its context and giving its manager
// a change to do clean up before final termination
func (m *Machine) Delete() {
	m.manager.Delete()

	if m.cancel != nil {
		m.Infof("runner", "Stopping")
		m.cancel()
	}
}

// Stop stops a running machine by canceling its context
func (m *Machine) Stop() {
	if m.cancel != nil {
		m.Infof("runner", "Stopping")
		m.cancel()
	}
}

// Transition performs the machine transition as defined by event t
func (m *Machine) Transition(t string, args ...interface{}) error {
	m.Lock()
	defer m.Unlock()

	if t == "" {
		return nil
	}

	if m.Can(t) {
		m.fsm.Event(t, args...)
	} else {
		m.Warnf("machine", "Could not fire '%s' event while in %s", t, m.fsm.Current())
	}

	return nil
}

// Can determines if a transition could be performed
func (m *Machine) Can(t string) bool {
	return m.fsm.Can(t)
}

// KnownTransitions is a list of known transition names
func (m *Machine) KnownTransitions() []string {
	transitions := make([]string, len(m.Transitions))

	for i, t := range m.Transitions {
		transitions[i] = t.Name
	}

	return transitions
}

// KnownStates is a list of all the known states in the Machine gathered by looking at initial state and all the states mentioned in transitions
func (m *Machine) KnownStates() []string {
	m.Lock()
	defer m.Unlock()

	lister := func() []string {
		states := []string{}

		for k := range m.knownStates {
			states = append(states, k)
		}

		return states
	}

	if len(m.knownStates) > 0 {
		return lister()
	}

	m.knownStates = make(map[string]bool)

	m.knownStates[m.InitialState] = true

	for _, t := range m.Transitions {
		m.knownStates[t.Destination] = true

		for _, e := range t.From {
			m.knownStates[e] = true
		}
	}

	return lister()
}
