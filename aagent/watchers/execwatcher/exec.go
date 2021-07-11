package execwatcher

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/choria-io/go-choria/aagent/util"
	"github.com/choria-io/go-choria/aagent/watchers/event"
	"github.com/choria-io/go-choria/aagent/watchers/watcher"
	iu "github.com/choria-io/go-choria/internal/util"
	"github.com/google/shlex"
)

type State int

const (
	Unknown State = iota
	Skipped
	Error
	Success

	wtype   = "exec"
	version = "v1"
)

var stateNames = map[State]string{
	Unknown: "unknown",
	Skipped: "skipped",
	Error:   "error",
	Success: "success",
}

type Properties struct {
	Command                 string
	Environment             []string
	Governor                string
	GovernorTimeout         time.Duration `mapstructure:"governor_timeout"`
	OutputAsData            bool          `mapstructure:"parse_as_data"`
	SuppressSuccessAnnounce bool          `mapstructure:"suppress_success_announce"`
	Timeout                 time.Duration
}

type Watcher struct {
	*watcher.Watcher

	name            string
	machine         watcher.Machine
	previous        State
	interval        time.Duration
	previousRunTime time.Duration
	properties      *Properties

	watching bool

	mu *sync.Mutex
}

func New(machine watcher.Machine, name string, states []string, failEvent string, successEvent string, interval string, ai time.Duration, rawprop map[string]interface{}) (interface{}, error) {
	var err error

	exec := &Watcher{
		machine: machine,
		name:    name,
		mu:      &sync.Mutex{},
		properties: &Properties{
			Environment: []string{},
		},
	}

	exec.Watcher, err = watcher.NewWatcher(name, wtype, ai, states, machine, failEvent, successEvent)
	if err != nil {
		return nil, err
	}

	err = exec.setProperties(rawprop)
	if err != nil {
		return nil, fmt.Errorf("could not set properties: %v", err)
	}

	if interval != "" {
		exec.interval, err = iu.ParseDuration(interval)
		if err != nil {
			return nil, fmt.Errorf("invalid interval: %v", err)
		}

		if exec.interval < 500*time.Millisecond {
			return nil, fmt.Errorf("interval %v is too small", exec.interval)
		}
	}

	return exec, nil
}

func (w *Watcher) validate() error {
	if w.properties.Command == "" {
		return fmt.Errorf("command is required")
	}

	if w.properties.Timeout == 0 {
		w.properties.Timeout = time.Second
	}

	if w.properties.Governor != "" && w.properties.GovernorTimeout == 0 {
		w.Infof("Setting Governor timeout to 5 minutes while unset")
		w.properties.GovernorTimeout = 5 * time.Minute
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
			w.CancelGovernor()
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
	if w.isWatching() {
		return
	}

	state, err := w.watch(ctx)
	err = w.handleCheck(state, err)
	if err != nil {
		w.Errorf("could not handle watcher event: %s", err)
	}
}

func (w *Watcher) handleCheck(s State, err error) error {
	w.Debugf("handling check for %s %s %v", w.properties.Command, stateNames[s], err)

	w.mu.Lock()
	w.previous = s
	w.mu.Unlock()

	switch s {
	case Error:
		w.NotifyWatcherState(w.CurrentState())
		return w.FailureTransition()

	case Success:
		if !w.properties.SuppressSuccessAnnounce {
			w.NotifyWatcherState(w.CurrentState())
		}

		return w.SuccessTransition()
	}

	return nil
}

func (w *Watcher) CurrentState() interface{} {
	w.mu.Lock()
	defer w.mu.Unlock()

	s := &StateNotification{
		Event:           event.New(w.name, wtype, version, w.machine),
		Command:         w.properties.Command,
		PreviousOutcome: stateNames[w.previous],
		PreviousRunTime: w.previousRunTime.Nanoseconds(),
	}

	return s
}

func (w *Watcher) startWatching() {
	w.mu.Lock()
	w.watching = true
	w.mu.Unlock()
}

func (w *Watcher) isWatching() bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.watching
}

func (w *Watcher) stopWatching() {
	w.mu.Lock()
	w.watching = false
	w.mu.Unlock()
}

func (w *Watcher) watch(ctx context.Context) (state State, err error) {
	if !w.ShouldWatch() {
		return Skipped, nil
	}

	w.startWatching()
	defer w.stopWatching()

	if w.properties.Governor != "" {
		fin, err := w.EnterGovernor(ctx, w.properties.Governor, w.properties.GovernorTimeout)
		if err != nil {
			w.Errorf("Cannot enter Governor %s: %s", w.properties.Governor, err)
			return Error, err
		}
		defer fin()
	}

	start := time.Now()
	defer func() {
		w.mu.Lock()
		w.previousRunTime = time.Since(start)
		w.mu.Unlock()
	}()

	w.Infof("Running %s", w.properties.Command)

	timeoutCtx, cancel := context.WithTimeout(ctx, w.properties.Timeout)
	defer cancel()

	splitcmd, err := shlex.Split(w.properties.Command)
	if err != nil {
		w.Errorf("Exec watcher %s failed: %s", w.properties.Command, err)
		return Error, err
	}

	if len(splitcmd) == 0 {
		w.Errorf("Invalid command %q", w.properties.Command)
		return Error, err
	}

	var args []string
	if len(splitcmd) > 1 {
		args = splitcmd[1:]
	}

	df, err := w.DataCopyFile()
	if err != nil {
		if df != "" {
			os.Remove(df)
		}
		w.Errorf("Could not get a copy of the data into a temporary file, skipping execution: %s", err)
		return Error, err
	}
	defer os.Remove(df)

	cmd := exec.CommandContext(timeoutCtx, splitcmd[0], args...)
	cmd.Dir = w.machine.Directory()

	cmd.Env = append(cmd.Env, fmt.Sprintf("MACHINE_WATCHER_NAME=%s", w.name))
	cmd.Env = append(cmd.Env, fmt.Sprintf("MACHINE_NAME=%s", w.machine.Name()))
	cmd.Env = append(cmd.Env, fmt.Sprintf("PATH=%s%s%s", os.Getenv("PATH"), string(os.PathListSeparator), w.machine.Directory()))
	cmd.Env = append(cmd.Env, w.properties.Environment...)
	cmd.Env = append(cmd.Env, fmt.Sprintf("WATCHER_DATA=%s", df))

	output, err := cmd.CombinedOutput()
	if err != nil {
		w.Errorf("Exec watcher %s failed: %s", w.properties.Command, err)
		return Error, err
	}

	w.Debugf("Output from %s: %s", w.properties.Command, output)

	if w.properties.OutputAsData {
		err = w.setOutputAsData(output)
		if err != nil {
			w.Errorf("Could not save output data: %s", err)
			return Error, err
		}
	}

	return Success, nil
}

func (w *Watcher) setOutputAsData(output []byte) error {
	dat := map[string]string{}
	err := json.Unmarshal(output, &dat)
	if err != nil {
		return err
	}

	for k, v := range dat {
		w.DataPut(k, v)
	}

	return nil
}
