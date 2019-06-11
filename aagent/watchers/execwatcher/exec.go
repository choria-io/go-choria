package execwatcher

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/google/shlex"
	"github.com/pkg/errors"
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

type Machine interface {
	State() string
	Transition(t string, args ...interface{}) error
	NotifyWatcherState(string, interface{})
	Name() string
	Directory() string
	Identity() string
	InstanceID() string
	Version() string
	TimeStampSeconds() int64
	Debugf(name string, format string, args ...interface{})
	Infof(name string, format string, args ...interface{})
	Errorf(name string, format string, args ...interface{})
}

type Watcher struct {
	name                    string
	states                  []string
	failEvent               string
	successEvent            string
	command                 string
	machine                 Machine
	interval                time.Duration
	announceInterval        time.Duration
	statechg                chan struct{}
	previous                State
	previousRunTime         time.Duration
	lastAnnounce            time.Time
	timeout                 time.Duration
	environment             []string
	suppressSuccessAnnounce bool

	sync.Mutex
}

func New(machine Machine, name string, states []string, failEvent string, successEvent string, interval string, ai time.Duration, properties map[string]interface{}) (watcher *Watcher, err error) {
	w := &Watcher{
		name:         name,
		successEvent: successEvent,
		failEvent:    failEvent,
		states:       states,
		machine:      machine,
		statechg:     make(chan struct{}, 1),
		interval:     0,
		environment:  []string{},

		announceInterval: ai,
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

		if w.interval < 500*time.Millisecond {
			return nil, errors.Errorf("interval %v is too small", w.interval)
		}
	}

	return w, err
}

func (w *Watcher) Type() string {
	return "exec"
}

func (w *Watcher) setProperties(p map[string]interface{}) error {
	command, ok := p["command"]
	if !ok {
		return fmt.Errorf("command is required")
	}

	w.command, ok = command.(string)
	if !ok {
		return fmt.Errorf("command should be a string")
	}

	w.timeout = 10 * time.Second
	t, ok := p["timeout"]
	if ok {
		ts, ok := t.(string)
		if !ok {
			return fmt.Errorf("timeout should be a duration string - example 10s, 1h or 1m")
		}

		timeout, err := time.ParseDuration(ts)
		if err != nil {
			return errors.Wrap(err, "invalid timeout")
		}

		w.timeout = timeout
	}

	suppress, ok := p["suppress_success_announce"]
	if ok {
		w.suppressSuccessAnnounce, ok = suppress.(bool)
		if !ok {
			return fmt.Errorf("suppress_announce should be boolean")
		}
	}

	environment, ok := p["environment"]
	if ok {
		envs, ok := environment.([]interface{})
		if !ok {
			return fmt.Errorf("environment should be a list of strings")
		}

		for _, env := range envs {
			val, ok := env.(string)
			if !ok {
				return fmt.Errorf("environment should be a list of strings")
			}

			w.environment = append(w.environment, val)
		}
	}

	return nil
}

func (w *Watcher) NotifyStateChance() {
	if len(w.statechg) < cap(w.statechg) {
		w.statechg <- struct{}{}
	}
}

func (w *Watcher) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	w.machine.Infof(w.name, "exec watcher for %s starting", w.command)

	if w.interval != 0 {
		wg.Add(1)
		go w.intervalWatcher(ctx, wg)
	}

	for {
		select {
		case <-w.statechg:
			w.performWatch(ctx)

		case <-ctx.Done():
			w.machine.Infof(w.name, "Stopping on context interrupt")
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
		w.machine.Errorf(w.name, "could not handle watcher event: %s", err)
	}
}

func (w *Watcher) handleCheck(s State, err error) error {
	w.machine.Debugf(w.name, "handling check for %s %s %v", w.command, stateNames[s], err)

	w.Lock()
	w.previous = s
	w.Unlock()

	switch s {
	case Error:
		w.machine.NotifyWatcherState(w.name, w.CurrentState())
		return w.machine.Transition(w.failEvent)

	case Success:
		if !w.suppressSuccessAnnounce {
			w.machine.NotifyWatcherState(w.name, w.CurrentState())
		}

		return w.machine.Transition(w.successEvent)
	}

	return nil
}

func (w *Watcher) AnnounceInterval() time.Duration {
	return w.announceInterval
}

func (w *Watcher) Name() string {
	return w.name
}

func (w *Watcher) CurrentState() interface{} {
	w.Lock()
	defer w.Unlock()

	s := &StateNotification{
		Protocol:        "io.choria.machine.watcher.exec.v1.state",
		Type:            "exec",
		Name:            w.name,
		Identity:        w.machine.Identity(),
		ID:              w.machine.InstanceID(),
		Version:         w.machine.Version(),
		Timestamp:       w.machine.TimeStampSeconds(),
		Machine:         w.machine.Name(),
		Command:         w.command,
		PreviousOutcome: stateNames[w.previous],
		PreviousRunTime: w.previousRunTime.Nanoseconds(),
	}

	return s
}

func (w *Watcher) watch(ctx context.Context) (state State, err error) {
	if !w.shouldWatch() {
		return Skipped, nil
	}

	start := time.Now()
	defer func() {
		w.Lock()
		w.previousRunTime = time.Now().Sub(start)
		w.Unlock()
	}()

	w.machine.Infof(w.name, "Running %s", w.command)

	timeoutCtx, cancel := context.WithTimeout(ctx, w.timeout)
	defer cancel()

	splitcmd, err := shlex.Split(w.command)
	if err != nil {
		w.machine.Errorf(w.name, "Exec watcher %s failed: %s", w.command, err)
		return Error, err
	}

	cmd := exec.CommandContext(timeoutCtx, splitcmd[0], splitcmd[1:]...)
	cmd.Env = append(cmd.Env, fmt.Sprintf("MACHINE_WATCHER_NAME=%s", w.name))
	cmd.Env = append(cmd.Env, fmt.Sprintf("MACHINE_NAME=%s", w.machine.Name()))
	cmd.Env = append(cmd.Env, fmt.Sprintf("PATH=%s%s%s", os.Getenv("PATH"), string(os.PathListSeparator), w.machine.Directory()))

	for _, e := range w.environment {
		cmd.Env = append(cmd.Env, e)
	}

	cmd.Dir = w.machine.Directory()

	output, err := cmd.CombinedOutput()
	if err != nil {
		w.machine.Errorf(w.name, "Exec watcher %s failed: %s", w.command, err)
		return Error, err
	}

	w.machine.Debugf(w.name, "Output from %s: %s", w.command, output)

	return Success, nil
}

func (w *Watcher) shouldWatch() bool {
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
