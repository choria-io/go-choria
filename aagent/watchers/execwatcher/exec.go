package execwatcher

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/google/shlex"

	"github.com/choria-io/go-choria/aagent/util"
	"github.com/choria-io/go-choria/aagent/watchers/event"
	"github.com/choria-io/go-choria/aagent/watchers/watcher"
)

type State int

const (
	Unknown State = iota
	Skipped
	Error
	Success
)

var stateNames = map[State]string{
	Unknown: "unknown",
	Skipped: "skipped",
	Error:   "error",
	Success: "success",
}

type Properties struct {
	Command                 string
	Timeout                 time.Duration
	SuppressSuccessAnnounce bool `mapstructure:"suppress_success_announce"`
	Environment             []string
}

type Watcher struct {
	*watcher.Watcher

	name            string
	machine         watcher.Machine
	previous        State
	interval        time.Duration
	previousRunTime time.Duration
	properties      *Properties
}

func New(machine watcher.Machine, name string, states []string, failEvent string, successEvent string, interval string, ai time.Duration, rawprop map[string]interface{}) (*Watcher, error) {
	var err error

	w := &Watcher{
		machine: machine,
		name:    name,
		properties: &Properties{
			Environment: []string{},
		},
	}

	w.Watcher, err = watcher.NewWatcher(name, "exec", ai, states, machine, failEvent, successEvent)
	if err != nil {
		return nil, err
	}

	err = w.setProperties(rawprop)
	if err != nil {
		return nil, fmt.Errorf("could not set properties: %v", err)
	}

	if interval != "" {
		w.interval, err = time.ParseDuration(interval)
		if err != nil {
			return nil, fmt.Errorf("invalid interval: %v", err)
		}

		if w.interval < 500*time.Millisecond {
			return nil, fmt.Errorf("interval %v is too small", w.interval)
		}
	}

	return w, nil
}

func (w *Watcher) validate() error {
	if w.properties.Command == "" {
		return fmt.Errorf("command is required")
	}

	if w.properties.Timeout == 0 {
		w.properties.Timeout = time.Second
	}

	return nil
}
func (w *Watcher) setProperties(props map[string]interface{}) error {
	if w.properties == nil {
		w.properties = &Properties{Environment: []string{}}
	}

	err := util.ParseMapStructure(props, w.properties)
	if err != nil {
		return err
	}

	return w.validate()
}

func (w *Watcher) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	w.Infof("exec watcher for %s starting", w.properties.Command)

	if w.interval != 0 {
		wg.Add(1)
		go w.intervalWatcher(ctx, wg)
	}

	for {
		select {
		case <-w.Watcher.StateChangeC():
			w.performWatch(ctx)

		case <-ctx.Done():
			w.Infof("Stopping on context interrupt")
			return
		}
	}
}

func (w *Watcher) intervalWatcher(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	tick := time.NewTicker(w.interval)

	for {
		select {
		case <-tick.C:
			w.performWatch(ctx)

		case <-ctx.Done():
			tick.Stop()
			return
		}
	}
}

func (w *Watcher) performWatch(ctx context.Context) {
	state, err := w.watch(ctx)
	err = w.handleCheck(state, err)
	if err != nil {
		w.Errorf("could not handle watcher event: %s", err)
	}
}

func (w *Watcher) handleCheck(s State, err error) error {
	w.Debugf("handling check for %s %s %v", w.properties.Command, stateNames[s], err)

	w.Lock()
	w.previous = s
	w.Unlock()

	switch s {
	case Error:
		w.NotifyWatcherState(w.name, w.CurrentState())
		return w.Transition(w.Watcher.FailEvent())

	case Success:
		if !w.properties.SuppressSuccessAnnounce {
			w.NotifyWatcherState(w.name, w.CurrentState())
		}

		return w.Transition(w.Watcher.SuccessEvent())
	}

	return nil
}

func (w *Watcher) CurrentState() interface{} {
	w.Lock()
	defer w.Unlock()

	s := &StateNotification{
		Event:           event.New(w.name, "exec", "v1", w.machine),
		Command:         w.properties.Command,
		PreviousOutcome: stateNames[w.previous],
		PreviousRunTime: w.previousRunTime.Nanoseconds(),
	}

	return s
}

func (w *Watcher) watch(ctx context.Context) (state State, err error) {
	if !w.ShouldWatch() {
		return Skipped, nil
	}

	start := time.Now()
	defer func() {
		w.Lock()
		w.previousRunTime = time.Since(start)
		w.Unlock()
	}()

	w.Infof("Running %s", w.properties.Command)

	timeoutCtx, cancel := context.WithTimeout(ctx, w.properties.Timeout)
	defer cancel()

	splitcmd, err := shlex.Split(w.properties.Command)
	if err != nil {
		w.Errorf("Metric watcher %s failed: %s", w.properties.Command, err)
		return Error, err
	}

	cmd := exec.CommandContext(timeoutCtx, splitcmd[0], splitcmd[1:]...)
	cmd.Env = append(cmd.Env, fmt.Sprintf("MACHINE_WATCHER_NAME=%s", w.name))
	cmd.Env = append(cmd.Env, fmt.Sprintf("MACHINE_NAME=%s", w.machine.Name()))
	cmd.Env = append(cmd.Env, fmt.Sprintf("PATH=%s%s%s", os.Getenv("PATH"), string(os.PathListSeparator), w.machine.Directory()))
	cmd.Env = append(cmd.Env, w.properties.Environment...)
	cmd.Dir = w.machine.Directory()

	output, err := cmd.CombinedOutput()
	if err != nil {
		w.Errorf("Exec watcher %s failed: %s", w.properties.Command, err)
		return Error, err
	}

	w.Debugf("Output from %s: %s", w.properties.Command, output)

	return Success, nil
}
