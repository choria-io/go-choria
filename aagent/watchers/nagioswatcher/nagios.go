package nagioswatcher

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"math/rand"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/shlex"
	"github.com/tidwall/gjson"

	"github.com/choria-io/go-choria/aagent/util"
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
	OverrideData() ([]byte, error)
	Transition(t string, args ...interface{}) error
	Debugf(name string, format string, args ...interface{})
	Infof(name string, format string, args ...interface{})
	Errorf(name string, format string, args ...interface{})
}

type PerfData struct {
	Unit  string  `json:"unit"`
	Label string  `json:"label"`
	Value float64 `json:"value"`
}

type Execution struct {
	Executed time.Time  `json:"execute"`
	Status   int        `json:"status"`
	PerfData []PerfData `json:"perfdata,omitempty"`
}

type Watcher struct {
	name             string
	machineName      string
	textFileDir      string
	gossFile         string
	states           []string
	failEvent        string
	successEvent     string
	machine          Machine
	interval         time.Duration
	announceInterval time.Duration
	previousRunTime  time.Duration
	previousOutput   string
	previousPerfData []PerfData
	previousCheck    time.Time
	previousPlugin   string
	previous         State
	force            bool
	statechg         chan struct{}
	plugin           string
	builtin          string
	annotations      map[string]string
	timeout          time.Duration
	history          []*Execution

	sync.Mutex
}

func New(machine Machine, name string, states []string, failEvent string, successEvent string, interval string, ai time.Duration, properties map[string]interface{}) (watcher *Watcher, err error) {
	w := &Watcher{
		name:             name,
		machineName:      machine.Name(),
		textFileDir:      machine.TextFileDirectory(),
		states:           states,
		failEvent:        failEvent,
		successEvent:     successEvent,
		machine:          machine,
		statechg:         make(chan struct{}, 1),
		previous:         NOTCHECKED,
		announceInterval: ai,
		history:          []*Execution{},
		annotations:      make(map[string]string),
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

	updatePromState(w.machineName, UNKNOWN, machine.TextFileDirectory(), machine)

	return w, err
}

// Delete stops the watcher and remove it from the prom state after the check was removed from disk
func (w *Watcher) Delete() {
	w.Lock()
	defer w.Unlock()

	// suppress next check and set state to unknown
	w.previousCheck = time.Now()
	deletePromState(w.machineName, w.textFileDir, w.machine)
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

	s := &StateNotification{
		Protocol:    "io.choria.machine.watcher.nagios.v1.state",
		Type:        "nagios",
		Name:        w.name,
		Identity:    w.machine.Identity(),
		ID:          w.machine.InstanceID(),
		Version:     w.machine.Version(),
		Timestamp:   w.machine.TimeStampSeconds(),
		Machine:     w.machineName,
		Plugin:      w.previousPlugin,
		Status:      stateNames[w.previous],
		StatusCode:  int(w.previous),
		Output:      w.previousOutput,
		PerfData:    w.previousPerfData,
		RunTime:     w.previousRunTime.Seconds(),
		History:     w.history,
		Annotations: w.annotations,
	}

	if !w.previousCheck.IsZero() {
		s.CheckTime = w.previousCheck.Unix()
	}

	return s
}

var valParse = regexp.MustCompile(`^([-*\d+\.]+)(us|ms|s|%|B|KB|MB|TB|c)*`)

// https://stackoverflow.com/questions/46886118/what-is-the-nagios-performance-data-format
func (w *Watcher) parsePerfData(pd string) (perf []PerfData) {
	parts := strings.Split(pd, "|")
	if len(parts) != 2 {
		return perf
	}

	rawMetrics := strings.Split(strings.TrimSpace(parts[1]), " ")
	for _, rawMetric := range rawMetrics {
		metric := strings.TrimSpace(rawMetric)
		metric = strings.TrimPrefix(metric, "'")
		metric = strings.TrimSuffix(metric, "'")

		if len(metric) == 0 {
			continue
		}

		// throwing away thresholds for now
		mparts := strings.Split(metric, ";")
		mparts = strings.Split(mparts[0], "=")
		if len(mparts) != 2 {
			continue
		}

		label := strings.Replace(mparts[0], " ", "_", -1)
		valParts := valParse.FindStringSubmatch(mparts[1])
		rawValue := valParts[1]
		value, err := strconv.ParseFloat(rawValue, 64)
		if err != nil {
			continue
		}

		pdi := PerfData{
			Label: label,
			Value: value,
		}
		if len(valParts) == 3 {
			pdi.Unit = valParts[2]
		}

		perf = append(perf, pdi)
	}

	return perf
}

func (w *Watcher) validate() error {
	if w.builtin != "" && w.plugin != "" {
		return fmt.Errorf("cannot set plugin and builtin")
	}

	if w.builtin == "" && w.plugin == "" {
		return fmt.Errorf("plugin or builtin is required")
	}

	if w.builtin == "goss" && w.gossFile == "" {
		return fmt.Errorf("gossfile property is required for the goss builtin check")
	}

	if w.timeout == 0 {
		w.timeout = time.Second
	}

	return nil
}

func (w *Watcher) setProperties(props map[string]interface{}) error {
	var properties struct {
		Annotations map[string]string
		Plugin      string
		Gossfile    string
		Builtin     string
		Timeout     time.Duration
	}

	err := util.ParseMapStructure(props, &properties)
	if err != nil {
		return err
	}

	w.annotations = properties.Annotations
	w.plugin = properties.Plugin
	w.gossFile = properties.Gossfile
	w.builtin = properties.Builtin
	w.timeout = properties.Timeout

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
		w.machine.Infof("Forcing a check of %s", w.machineName)
		w.force = true
		w.statechg <- struct{}{}
		return
	}

	w.Lock()
	w.previous = s
	w.Unlock()

	err := updatePromState(w.machineName, s, w.textFileDir, w.machine)
	if err != nil {
		w.machine.Errorf(w.name, "Could not update prometheus: %s", err)
	}
}

func (w *Watcher) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	if w.textFileDir != "" {
		w.machine.Infof(w.name, "nagios watcher starting, updating prometheus in %s", w.textFileDir)
	} else {
		w.machine.Infof(w.name, "nagios watcher starting, prometheus integration disabled")
	}

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

	splay := time.Duration(rand.Intn(int(w.interval.Seconds()))) * time.Second
	w.machine.Infof(w.name, "Splaying first check by %v", splay)

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
	start := time.Now().UTC()
	state, err := w.watch(ctx)
	err = w.handleCheck(start, state, false, err)
	if err != nil {
		w.machine.Errorf(w.name, "could not handle watcher event: %s", err)
	}
}

func (w *Watcher) handleCheck(start time.Time, s State, external bool, err error) error {
	if s == SKIPPED || s == NOTCHECKED {
		return nil
	}

	w.machine.Debugf(w.name, "handling check for %s %s %v", w.plugin, stateNames[s], err)

	w.Lock()
	w.previous = s

	if len(w.history) >= 15 {
		w.history = w.history[1:]
	}
	w.history = append(w.history, &Execution{Executed: start, Status: int(s), PerfData: w.previousPerfData})

	w.Unlock()

	// dont notify if we are externally transitioning because probably notifications were already sent
	if !external {
		w.machine.NotifyWatcherState(w.name, w.CurrentState())
	}

	w.machine.Debugf(w.name, "Notifying prometheus")

	err = updatePromState(w.machineName, s, w.textFileDir, w.machine)
	if err != nil {
		w.machine.Errorf(w.name, "Could not update prometheus: %s", err)
	}

	if external {
		return nil
	}

	return w.machine.Transition(stateNames[s])
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
		"o": func(path string, dflt interface{}) string {
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
	timeoutCtx, cancel := context.WithTimeout(ctx, w.timeout)
	defer cancel()

	plugin, err := w.processOverrides(w.plugin)
	if err != nil {
		w.machine.Errorf(w.name, "could not process overrides for plugin command: %s", err)
		return UNKNOWN, "", err
	}

	w.machine.Infof(w.name, "Running %s", w.plugin)

	w.machine.Infof(w.name, "command post processing is: %s", plugin)

	splitcmd, err := shlex.Split(plugin)
	if err != nil {
		w.machine.Errorf(w.name, "Exec watcher %s failed: %s", plugin, err)
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
			w.machine.Errorf(w.name, "Exec watcher %s failed: %s", w.plugin, err)
			w.previousOutput = err.Error()
			return UNKNOWN, "", err
		}
	} else {
		pstate = cmd.ProcessState
	}

	output = string(outb)

	w.machine.Debugf(w.name, "Output from %s: %s", w.plugin, output)

	s, ok := intStates[pstate.ExitCode()]
	if ok {
		return s, output, nil
	}

	return UNKNOWN, output, nil
}

func (w *Watcher) watchUsingBuiltin(_ context.Context) (state State, output string, err error) {
	switch {
	case w.builtin == "heartbeat":
		return w.builtinHeartbeat()
	case strings.HasPrefix(w.builtin, "goss"):
		return w.watchUsingGoss()
	default:
		return UNKNOWN, "", fmt.Errorf("unsupported builtin %q", w.builtin)
	}
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

	var output string

	switch {
	case w.plugin != "":
		state, output, err = w.watchUsingPlugin(ctx)
	case w.builtin != "":
		state, output, err = w.watchUsingBuiltin(ctx)
	default:
		state = UNKNOWN
		err = fmt.Errorf("command or builtin required")
	}

	w.previousOutput = strings.TrimSpace(output)
	w.previousPerfData = w.parsePerfData(output)

	return state, err
}

func (w *Watcher) shouldWatch() bool {
	if w.force {
		w.force = false
		return true
	}

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
