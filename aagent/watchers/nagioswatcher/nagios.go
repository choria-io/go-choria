// Copyright (c) 2020-2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package nagioswatcher

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"math/rand/v2"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/choria-io/go-choria/aagent/model"
	"github.com/google/shlex"
	"github.com/tidwall/gjson"

	"github.com/choria-io/go-choria/aagent/util"
	"github.com/choria-io/go-choria/aagent/watchers/event"
	"github.com/choria-io/go-choria/aagent/watchers/watcher"
	iu "github.com/choria-io/go-choria/internal/util"
)

type State int

const (
	OK State = iota
	WARNING
	CRITICAL
	UNKNOWN
	SKIPPED
	NOTCHECKED

	wtype   = "nagios"
	version = "v1"
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

// StateName returns friendly name for a state
func StateName(s int) string {
	state, ok := intStates[s]
	if !ok {
		return stateNames[UNKNOWN]
	}

	return stateNames[state]
}

type properties struct {
	Annotations map[string]string
	Plugin      string
	Gossfile    string
	Builtin     string
	Timeout     time.Duration
	LastMessage time.Duration `mapstructure:"last_message"`
	CertExpiry  time.Duration `mapstructure:"pubcert_expire"`
	TokenExpiry time.Duration `mapstructure:"token_expire"`
}

type Execution struct {
	Executed time.Time       `json:"execute"`
	Status   int             `json:"status"`
	PerfData []util.PerfData `json:"perfdata,omitempty"`
}

type Watcher struct {
	*watcher.Watcher

	properties       *properties
	name             string
	machine          model.Machine
	interval         time.Duration
	previousRunTime  time.Duration
	previousOutput   string
	previousPerfData []util.PerfData
	previousCheck    time.Time
	previousPlugin   string
	previous         State
	force            bool
	history          []*Execution
	machineName      string
	textFileDir      string

	watching bool
	mu       *sync.Mutex
}

func New(machine model.Machine, name string, states []string, required []model.ForeignMachineState, failEvent string, successEvent string, interval string, ai time.Duration, properties map[string]any) (any, error) {
	var err error

	nw := &Watcher{
		machineName: machine.Name(),
		textFileDir: machine.TextFileDirectory(),
		name:        name,
		machine:     machine,
		previous:    NOTCHECKED,
		history:     []*Execution{},
		mu:          &sync.Mutex{},
	}

	nw.Watcher, err = watcher.NewWatcher(name, wtype, ai, states, required, machine, failEvent, successEvent)
	if err != nil {
		return nil, err
	}

	err = nw.setProperties(properties)
	if err != nil {
		return nil, fmt.Errorf("could not set properties: %s", err)
	}

	if interval != "" {
		nw.interval, err = iu.ParseDuration(interval)
		if err != nil {
			return nil, fmt.Errorf("invalid interval: %s", err)
		}

		if nw.interval < 500*time.Millisecond {
			return nil, fmt.Errorf("interval %v is too small", nw.interval)
		}
	}

	updatePromState(nw.machineName, UNKNOWN, machine.TextFileDirectory(), nw)

	return nw, err
}

// Delete stops the watcher and remove it from the prom state after the check was removed from disk
func (w *Watcher) Delete() {
	w.mu.Lock()
	defer w.mu.Unlock()

	// suppress next check and set state to unknown
	w.previousCheck = time.Now()
	deletePromState(w.machineName, w.textFileDir, w)
}

func (w *Watcher) CurrentState() any {
	w.mu.Lock()
	defer w.mu.Unlock()

	s := &StateNotification{
		Event:       event.New(w.name, wtype, version, w.machine),
		Plugin:      w.previousPlugin,
		Status:      stateNames[w.previous],
		StatusCode:  int(w.previous),
		Output:      w.previousOutput,
		PerfData:    w.previousPerfData,
		RunTime:     w.previousRunTime.Seconds(),
		History:     w.history,
		Annotations: w.properties.Annotations,
		CheckTime:   w.previousCheck.Unix(),
	}

	if !w.previousCheck.IsZero() {
		s.CheckTime = w.previousCheck.Unix()
	}

	return s
}

func (w *Watcher) validate() error {
	if w.properties.Builtin != "" && w.properties.Plugin != "" {
		return fmt.Errorf("cannot set plugin and builtin")
	}

	if w.properties.Builtin == "" && w.properties.Plugin == "" {
		return fmt.Errorf("plugin or builtin is required")
	}

	if w.properties.Builtin == "goss" && w.properties.Gossfile == "" {
		return fmt.Errorf("gossfile property is required for the goss builtin check")
	}

	if w.properties.Builtin == "choria_status" && w.properties.LastMessage == 0 {
		return fmt.Errorf("last_message property is required for the choria_status builtin check")
	}

	if w.properties.Timeout == 0 {
		w.properties.Timeout = time.Second
	}

	return nil
}

func (w *Watcher) setProperties(props map[string]any) error {
	if w.properties == nil {
		w.properties = &properties{
			Annotations: make(map[string]string),
			Timeout:     time.Second,
		}
	}

	err := util.ParseMapStructure(props, &w.properties)
	if err != nil {
		return err
	}

	return w.validate()
}

func (w *Watcher) NotifyStateChance() {
	var s State
	switch w.machine.State() {
	case "OK":
		s = OK
	case "WARNING":
		s = WARNING
	case "CRITICAL":
		s = CRITICAL
	case "UNKNOWN":
		s = UNKNOWN
	case "FORCE_CHECK":
		w.Infof("Forcing a check of %s", w.machineName)
		w.force = true
		w.StateChangeC() <- struct{}{}
		return
	}

	w.mu.Lock()
	w.previous = s
	w.mu.Unlock()

	err := updatePromState(w.machineName, s, w.textFileDir, w)
	if err != nil {
		w.Errorf("Could not update prometheus: %s", err)
	}
}

func (w *Watcher) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	if w.textFileDir != "" {
		w.Infof("nagios watcher starting, updating prometheus in %s", w.textFileDir)
	} else {
		w.Infof("nagios watcher starting, prometheus integration disabled")
	}

	if w.interval != 0 {
		wg.Add(1)
		go w.intervalWatcher(ctx, wg)
	}

	for {
		select {
		case <-w.StateChangeC():
			w.performWatch(ctx)

		case <-ctx.Done():
			w.Infof("Stopping on context interrupt")
			return
		}
	}
}

func (w *Watcher) intervalWatcher(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	splay := rand.N(w.interval)
	w.Infof("Splaying first check by %v", splay)

	select {
	case <-time.NewTimer(splay).C:
	case <-ctx.Done():
		return
	}

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

	start := time.Now().UTC()
	state, err := w.watch(ctx)
	err = w.handleCheck(start, state, false, err)
	if err != nil {
		w.Errorf("could not handle watcher event: %s", err)
	}
}

func (w *Watcher) handleCheck(start time.Time, s State, external bool, err error) error {
	if s == SKIPPED || s == NOTCHECKED {
		return nil
	}

	w.Debugf("handling check for %s %s %v", w.properties.Plugin, stateNames[s], err)

	w.mu.Lock()
	w.previous = s

	if len(w.history) >= 15 {
		w.history = w.history[1:]
	}
	w.history = append(w.history, &Execution{Executed: start, Status: int(s), PerfData: w.previousPerfData})

	w.mu.Unlock()

	// dont notify if we are externally transitioning because probably notifications were already sent
	if !external {
		w.NotifyWatcherState(w.CurrentState())
	}

	w.Debugf("Notifying prometheus")

	err = updatePromState(w.machineName, s, w.textFileDir, w)
	if err != nil {
		w.Errorf("Could not update prometheus: %s", err)
	}

	if external {
		return nil
	}

	return w.Transition(stateNames[s])
}

func (w *Watcher) processOverrides(c string) (string, error) {
	res, err := template.New(w.name).Funcs(w.funcMap()).Parse(c)
	if err != nil {
		return c, err
	}

	wr := new(bytes.Buffer)
	err = res.Execute(wr, struct{}{})
	if err != nil {
		return c, err
	}

	return wr.String(), nil
}

func (w *Watcher) funcMap() template.FuncMap {
	return template.FuncMap{
		"o": func(path string, dflt any) string {
			overrides, err := w.machine.OverrideData()
			if err != nil {
				return fmt.Sprintf("%v", dflt)
			}

			if len(overrides) == 0 {
				return fmt.Sprintf("%v", dflt)
			}

			r := gjson.GetBytes(overrides, w.machineName+"."+path)
			if !r.Exists() {
				return fmt.Sprintf("%v", dflt)
			}

			return r.String()
		},
	}
}

func (w *Watcher) watchUsingPlugin(ctx context.Context) (state State, output string, err error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, w.properties.Timeout)
	defer cancel()

	plugin, err := w.processOverrides(w.properties.Plugin)
	if err != nil {
		w.Errorf("could not process overrides for plugin command: %s", err)
		return UNKNOWN, "", err
	}

	w.Infof("Running %s", w.properties.Plugin)

	splitcmd, err := shlex.Split(plugin)
	if err != nil {
		w.Errorf("Exec watcher %s failed: %s", plugin, err)
		return UNKNOWN, "", err
	}

	w.previousPlugin = plugin

	cmd := exec.CommandContext(timeoutCtx, splitcmd[0], splitcmd[1:]...)
	cmd.Env = append(cmd.Env, fmt.Sprintf("MACHINE_WATCHER_NAME=%s", w.name))
	cmd.Env = append(cmd.Env, fmt.Sprintf("MACHINE_NAME=%s", w.machineName))
	cmd.Env = append(cmd.Env, fmt.Sprintf("PATH=%s%s%s", os.Getenv("PATH"), string(os.PathListSeparator), w.machine.Directory()))
	cmd.Dir = w.machine.Directory()

	var pstate *os.ProcessState

	outb, err := cmd.CombinedOutput()
	if err != nil {
		eerr, ok := err.(*exec.ExitError)
		if ok {
			pstate = eerr.ProcessState
		} else {
			w.Errorf("Exec watcher %s failed: %s", w.properties.Plugin, err)
			w.previousOutput = err.Error()
			return UNKNOWN, "", err
		}
	} else {
		pstate = cmd.ProcessState
	}

	output = string(outb)

	w.Debugf("Output from %s: %s", w.properties.Plugin, output)

	s, ok := intStates[pstate.ExitCode()]
	if ok {
		return s, output, nil
	}

	return UNKNOWN, output, nil
}

func (w *Watcher) watchUsingBuiltin(_ context.Context) (state State, output string, err error) {
	w.previousPlugin = w.properties.Builtin

	switch {
	case w.properties.Builtin == "heartbeat":
		return w.builtinHeartbeat()
	case strings.HasPrefix(w.properties.Builtin, "goss"):
		return w.watchUsingGoss()
	case w.properties.Builtin == "choria_status":
		return w.watchUsingChoria()
	default:
		return UNKNOWN, "", fmt.Errorf("unsupported builtin %q", w.properties.Builtin)
	}
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
		return SKIPPED, nil
	}

	w.startWatching()
	defer w.stopWatching()

	start := time.Now()
	w.previousCheck = start
	defer func() {
		w.mu.Lock()
		w.previousRunTime = time.Since(start)
		w.mu.Unlock()
	}()

	var output string

	switch {
	case w.properties.Plugin != "":
		state, output, err = w.watchUsingPlugin(ctx)
	case w.properties.Builtin != "":
		state, output, err = w.watchUsingBuiltin(ctx)
	default:
		state = UNKNOWN
		err = fmt.Errorf("command or builtin required")
	}

	w.previousOutput = strings.TrimSpace(output)
	w.previousPerfData = util.ParsePerfData(output)

	return state, err
}

func (w *Watcher) ShouldWatch() bool {
	if w.force {
		w.force = false
		return true
	}

	since := time.Since(w.previousCheck)
	if !w.previousCheck.IsZero() && since < w.interval-time.Second {
		w.Debugf("Skipping check due to previous check being %v sooner than interval %v", since, w.interval)
		return false
	}

	return w.Watcher.ShouldWatch()
}
