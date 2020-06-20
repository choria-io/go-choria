package nagioswatcher

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/google/shlex"
)

type State int

const (
	OK State = iota
	WARNING
	CRITICAL
	UNKNOWN
	SKIPPED
	NOTCHECKED
)

var stateNames = map[State]string{
	OK:       "OK",
	WARNING:  "WARNING",
	CRITICAL: "CRITICAL",
	UNKNOWN:  "UNKNOWN",

	// these are internal states that doesnt cause prom updates
	// or matching state transitions, they are there to force transitions
	// to unknown on the first time and to avoid immediate double checks
	// when transitioning between states
	SKIPPED:    "SKIPPED",
	NOTCHECKED: "NOTCHECKED",
}

var intStates = map[int]State{
	int(OK):         OK,
	int(WARNING):    WARNING,
	int(CRITICAL):   CRITICAL,
	int(UNKNOWN):    UNKNOWN,
	int(SKIPPED):    SKIPPED,
	int(NOTCHECKED): NOTCHECKED,
}

type Machine interface {
	State() string
	NotifyWatcherState(string, interface{})
	Name() string
	Directory() string
	Identity() string
	InstanceID() string
	Version() string
	TimeStampSeconds() int64
	TextFileDirectory() string
	Transition(t string, args ...interface{}) error
	Debugf(name string, format string, args ...interface{})
	Infof(name string, format string, args ...interface{})
	Errorf(name string, format string, args ...interface{})
}

type Watcher struct {
	name             string
	states           []string
	failEvent        string
	successEvent     string
	machine          Machine
	interval         time.Duration
	announceInterval time.Duration
	previousRunTime  time.Duration
	previousOutput   string
	previousCheck    time.Time
	previous         State
	statechg         chan struct{}

	plugin  string
	args    []string
	timeout time.Duration

	sync.Mutex
}

func New(machine Machine, name string, states []string, failEvent string, successEvent string, interval string, ai time.Duration, properties map[string]interface{}) (watcher *Watcher, err error) {
	w := &Watcher{
		name:             name,
		states:           states,
		failEvent:        failEvent,
		successEvent:     successEvent,
		machine:          machine,
		statechg:         make(chan struct{}, 1),
		previous:         NOTCHECKED,
		announceInterval: ai,
	}

	err = w.setProperties(properties)
	if err != nil {
		return nil, fmt.Errorf("could not set properties: %s", err)
	}

	if interval != "" {
		w.interval, err = time.ParseDuration(interval)
		if err != nil {
			return nil, fmt.Errorf("invalid interval: %s", err)
		}

		if w.interval < 500*time.Millisecond {
			return nil, fmt.Errorf("interval %v is too small", w.interval)
		}
	}

	return w, err
}

// Delete stops the watcher and remove it from the prom state after the check was removed from disk
func (w *Watcher) Delete() {
	w.Lock()
	defer w.Unlock()

	// suppress next check and set state to unknown
	w.previousCheck = time.Now()
	deletePromState(w.machine.Name(), w.machine.TextFileDirectory(), w.machine)
}

func (w *Watcher) Type() string {
	return "nagios"
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

	pd := ""
	parts := strings.SplitN(w.previousOutput, "|", 2)
	if len(parts) == 2 {
		pd = strings.TrimSpace(parts[1])
	}

	s := &StateNotification{
		Protocol:   "io.choria.machine.watcher.nagios.v1.state",
		Type:       "nagios",
		Name:       w.name,
		Identity:   w.machine.Identity(),
		ID:         w.machine.InstanceID(),
		Version:    w.machine.Version(),
		Timestamp:  w.machine.TimeStampSeconds(),
		Machine:    w.machine.Name(),
		Plugin:     w.plugin,
		Status:     stateNames[w.previous],
		StatusCode: int(w.previous),
		Output:     w.previousOutput,
		PerfData:   pd,
		CheckTime:  w.previousCheck.Unix(),
		RunTime:    w.previousRunTime.Seconds(),
	}

	return s
}

func (w *Watcher) setProperties(p map[string]interface{}) error {
	command, ok := p["plugin"]
	if !ok {
		return fmt.Errorf("plugin is required")
	}

	w.plugin, ok = command.(string)
	if !ok {
		return fmt.Errorf("plugin should be a string")
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
			return fmt.Errorf("invalid timeout: %s", err)
		}

		w.timeout = timeout
	}

	argsraw, ok := p["args"]
	if ok {
		args, ok := argsraw.([]interface{})
		if !ok {
			return fmt.Errorf("arguments should be a list of strings")
		}

		for _, arg := range args {
			val, ok := arg.(string)
			if !ok {
				return fmt.Errorf("arguments should be a list of strings")
			}

			w.args = append(w.args, val)
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

	if w.machine.TextFileDirectory() != "" {
		w.machine.Infof(w.name, "nagios watcher for %s starting, updating prometheus in %s", w.plugin, w.machine.TextFileDirectory())
	} else {
		w.machine.Infof(w.name, "nagios watcher for %s starting", w.plugin)
	}

	if w.interval != 0 {
		wg.Add(1)
		go w.intervalWatcher(ctx, wg)
	}

	// force a check at start, use machine splay to splay checks
	w.statechg <- struct{}{}

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
	if s == SKIPPED || s == NOTCHECKED {
		return nil
	}

	w.machine.Debugf(w.name, "handling check for %s %s %v", w.plugin, stateNames[s], err)

	w.Lock()
	w.previous = s
	w.Unlock()

	w.machine.NotifyWatcherState(w.name, w.CurrentState())
	w.machine.Debugf(w.name, "Notifying prometheus")

	err = updatePromState(w.machine.Name(), s, w.machine.TextFileDirectory(), w.machine)
	if err != nil {
		w.machine.Errorf(w.name, "Could not update prometheus: %s", err)
	}

	return w.machine.Transition(stateNames[s])
}

func (w *Watcher) watch(ctx context.Context) (state State, err error) {
	if !w.shouldWatch() {
		return SKIPPED, nil
	}

	start := time.Now()
	w.previousCheck = start
	defer func() {
		w.Lock()
		w.previousRunTime = time.Since(start)
		w.Unlock()
	}()

	w.machine.Infof(w.name, "Running %s", w.plugin)

	timeoutCtx, cancel := context.WithTimeout(ctx, w.timeout)
	defer cancel()

	splitcmd, err := shlex.Split(w.plugin)
	if err != nil {
		w.machine.Errorf(w.name, "Exec watcher %s failed: %s", w.plugin, err)
		return UNKNOWN, err
	}

	cmd := exec.CommandContext(timeoutCtx, splitcmd[0], splitcmd[1:]...)
	cmd.Env = append(cmd.Env, fmt.Sprintf("MACHINE_WATCHER_NAME=%s", w.name))
	cmd.Env = append(cmd.Env, fmt.Sprintf("MACHINE_NAME=%s", w.machine.Name()))
	cmd.Env = append(cmd.Env, fmt.Sprintf("PATH=%s%s%s", os.Getenv("PATH"), string(os.PathListSeparator), w.machine.Directory()))
	cmd.Dir = w.machine.Directory()

	var pstate *os.ProcessState

	output, err := cmd.CombinedOutput()
	if err != nil {
		eerr, ok := err.(*exec.ExitError)
		if ok {
			pstate = eerr.ProcessState
		} else {
			w.machine.Errorf(w.name, "Exec watcher %s failed: %s", w.plugin, err)
			w.previousOutput = err.Error()
			return UNKNOWN, err
		}
	} else {
		pstate = cmd.ProcessState
	}

	w.previousOutput = strings.TrimSpace(string(output))

	w.machine.Debugf(w.name, "Output from %s: %s", w.plugin, output)

	s, ok := intStates[pstate.ExitCode()]
	if ok {
		return s, nil
	}

	return UNKNOWN, nil
}

func (w *Watcher) shouldWatch() bool {
	since := time.Since(w.previousCheck)
	if !w.previousCheck.IsZero() && since < w.interval-time.Second {
		w.machine.Debugf(w.name, "Skipping check due to previous check being %v sooner than interval %v", since, w.interval)
		return false
	}

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
