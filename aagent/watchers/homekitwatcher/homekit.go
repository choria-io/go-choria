package homekitwatcher

import (
	"context"
	"crypto/md5"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/brutella/hc"
	"github.com/brutella/hc/accessory"

	"github.com/choria-io/go-choria/aagent/util"
	"github.com/choria-io/go-choria/aagent/watchers/event"
	"github.com/choria-io/go-choria/aagent/watchers/watcher"
)

type State int

const (
	Unknown State = iota
	On
	Off
	// used to indicate that an external event - rpc or other watcher - initiated a transition
	OnNoTransition
	OffNoTransition

	wtype   = "homekit"
	version = "v1"
)

var stateNames = map[State]string{
	Unknown: "unknown",
	On:      "on",
	Off:     "off",
}

type transport interface {
	Stop() <-chan struct{}
	Start()
}

type properties struct {
	SerialNumber  string `mapstructure:"serial_number"`
	Model         string
	Pin           string
	SetupId       string   `mapstructure:"setup_id"`
	ShouldOn      []string `mapstructure:"on_when"`
	ShouldOff     []string `mapstructure:"off_when"`
	ShouldDisable []string `mapstructure:"disable_when"`
	InitialState  State    `mapstructure:"-"`
	Path          string   `mapstructure:"-"`
	Initial       bool
}

type Watcher struct {
	*watcher.Watcher

	name        string
	machine     watcher.Machine
	previous    State
	interval    time.Duration
	hkt         transport
	ac          *accessory.Switch
	buttonPress chan State
	started     bool
	properties  *properties
	mu          *sync.Mutex
}

func New(machine watcher.Machine, name string, states []string, failEvent string, successEvent string, interval string, ai time.Duration, rawprop map[string]interface{}) (interface{}, error) {
	var err error

	hkw := &Watcher{
		name:        name,
		machine:     machine,
		interval:    5 * time.Second,
		buttonPress: make(chan State, 1),
		mu:          &sync.Mutex{},
		properties: &properties{
			Model:         "Autonomous Agent",
			ShouldOn:      []string{},
			ShouldOff:     []string{},
			ShouldDisable: []string{},
			Path:          filepath.Join(machine.Directory(), wtype, fmt.Sprintf("%x", md5.Sum([]byte(name)))),
		},
	}

	hkw.Watcher, err = watcher.NewWatcher(name, wtype, ai, states, machine, failEvent, successEvent)
	if err != nil {
		return nil, err
	}

	err = hkw.setProperties(rawprop)
	if err != nil {
		return nil, fmt.Errorf("could not set properties: %s", err)
	}

	return hkw, err
}

func (w *Watcher) Delete() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.hkt != nil {
		<-w.hkt.Stop()
	}
}

func (w *Watcher) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	if w.ShouldWatch() {
		w.ensureStarted()

		w.Infof("homekit watcher for %s starting in state %s", w.name, stateNames[w.properties.InitialState])
		switch w.properties.InitialState {
		case On:
			w.buttonPress <- On
		case Off:
			w.buttonPress <- Off
		default:
			w.Watcher.StateChangeC() <- struct{}{}
		}
	}

	for {
		select {
		// button call backs
		case e := <-w.buttonPress:
			err := w.handleStateChange(e)
			if err != nil {
				w.Errorf("Could not handle button %s press: %v", stateNames[e], err)
			}

		// rpc initiated state changes would trigger this
		case <-w.Watcher.StateChangeC():
			mstate := w.machine.State()

			switch {
			case w.shouldBeOn(mstate):
				w.ensureStarted()
				w.buttonPress <- OnNoTransition

			case w.shouldBeOff(mstate):
				w.ensureStarted()
				w.buttonPress <- OffNoTransition

			case w.shouldBeDisabled(mstate):
				w.ensureStopped()
			}

		case <-ctx.Done():
			w.Infof("Stopping on context interrupt")
			w.ensureStopped()
			return
		}
	}
}

func (w *Watcher) ensureStopped() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.started {
		return
	}

	w.Infof("Stopping homekit integration")
	<-w.hkt.Stop()
	w.Infof("Homekit integration stopped")

	w.started = false
}

func (w *Watcher) ensureStarted() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.started {
		return
	}

	w.Infof("Starting homekit integration")

	// kind of want to just hk.Start() here but stop kills a context that
	// start does not recreate so we have to go back to start
	err := w.startAccessoryUnlocked()
	if err != nil {
		w.Errorf("Could not start homekit service: %s", err)
		return
	}

	go w.hkt.Start()

	w.started = true
}

func (w *Watcher) shouldBeOff(s string) bool {
	for _, state := range w.properties.ShouldOff {
		if state == s {
			return true
		}
	}

	return false
}

func (w *Watcher) shouldBeOn(s string) bool {
	for _, state := range w.properties.ShouldOn {
		if state == s {
			return true
		}
	}

	return false
}

func (w *Watcher) shouldBeDisabled(s string) bool {
	for _, state := range w.properties.ShouldDisable {
		if state == s {
			return true
		}
	}

	return false
}

func (w *Watcher) handleStateChange(s State) error {
	if !w.ShouldWatch() {
		return nil
	}

	switch s {
	case On:
		w.setPreviousState(s)
		w.NotifyWatcherState(w.CurrentState())
		return w.SuccessTransition()

	case OnNoTransition:
		w.setPreviousState(On)
		w.ac.Switch.On.SetValue(true)
		w.NotifyWatcherState(w.CurrentState())
		return nil

	case Off:
		w.setPreviousState(s)
		w.NotifyWatcherState(w.CurrentState())
		return w.FailureTransition()

	case OffNoTransition:
		w.setPreviousState(Off)
		w.ac.Switch.On.SetValue(false)
		w.machine.NotifyWatcherState(w.name, w.CurrentState())
		return nil
	}

	return fmt.Errorf("invalid state change event: %s", stateNames[s])
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

func (w *Watcher) startAccessoryUnlocked() error {
	info := accessory.Info{
		Name:             strings.Title(strings.Replace(w.name, "_", " ", -1)),
		SerialNumber:     w.properties.SerialNumber,
		Manufacturer:     "Choria",
		Model:            w.properties.Model,
		FirmwareRevision: w.machine.Version(),
	}
	w.ac = accessory.NewSwitch(info)

	t, err := hc.NewIPTransport(hc.Config{Pin: w.properties.Pin, SetupId: w.properties.SetupId, StoragePath: w.properties.Path}, w.ac.Accessory)
	if err != nil {
		return err
	}

	hc.OnTermination(func() {
		<-t.Stop()
	})

	w.ac.Switch.On.OnValueRemoteUpdate(func(new bool) {
		w.mu.Lock()
		defer w.mu.Unlock()

		w.Infof("Handling app button press: %v", new)

		if !w.ShouldWatch() {
			w.Infof("Ignoring event while in %s state", w.machine.State())
			// undo the button press
			w.ac.Switch.On.SetValue(!new)
			return
		}

		if new {
			w.Infof("Setting state to On")
			w.buttonPress <- On
		} else {
			w.Infof("Setting state to Off")
			w.buttonPress <- Off
		}
	})

	w.ac.Switch.On.SetValue(w.previous == On)

	w.hkt = t

	return nil
}

func (w *Watcher) validate() error {
	if len(w.properties.Pin) > 0 && len(w.properties.Pin) != 8 {
		return fmt.Errorf("pin should be 8 characters long")
	}

	if len(w.properties.SetupId) > 0 && len(w.properties.SetupId) != 4 {
		return fmt.Errorf("setup_id should be 4 characters long")
	}

	if w.properties.Path == "" {
		return fmt.Errorf("machine path could not be determined")
	}

	return nil
}

func (w *Watcher) setProperties(props map[string]interface{}) error {
	if w.properties == nil {
		w.properties = &properties{
			ShouldDisable: []string{},
			ShouldOff:     []string{},
			ShouldOn:      []string{},
		}
	}

	err := util.ParseMapStructure(props, w.properties)
	if err != nil {
		return err
	}

	_, set := props["initial"]
	switch {
	case !set:
		w.properties.InitialState = Unknown
	case w.properties.Initial:
		w.properties.InitialState = On
	default:
		w.properties.InitialState = Off

	}

	return w.validate()
}
