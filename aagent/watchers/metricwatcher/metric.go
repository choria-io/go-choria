package metricwatcher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/google/shlex"

	"github.com/choria-io/go-choria/aagent/util"
	"github.com/choria-io/go-choria/aagent/watchers/event"
	"github.com/choria-io/go-choria/aagent/watchers/watcher"
)

const (
	wtype   = "metric"
	version = "v1"
)

type Metric struct {
	Labels  map[string]string  `json:"labels"`
	Metrics map[string]float64 `json:"metrics"`
	name    string
	machine string
	seen    int
}

type properties struct {
	Command  string
	Interval time.Duration
	Labels   map[string]string
}

type Watcher struct {
	*watcher.Watcher

	name            string
	machine         watcher.Machine
	previousRunTime time.Duration
	previousResult  *Metric
	properties      *properties

	watching bool
	mu       *sync.Mutex
}

func New(machine watcher.Machine, name string, states []string, failEvent string, successEvent string, interval string, ai time.Duration, rawprops map[string]interface{}) (interface{}, error) {
	var err error

	mw := &Watcher{
		name:    name,
		machine: machine,
		mu:      &sync.Mutex{},
	}

	mw.Watcher, err = watcher.NewWatcher(name, wtype, ai, states, machine, failEvent, successEvent)
	if err != nil {
		return nil, err
	}

	err = mw.setProperties(rawprops)
	if err != nil {
		return nil, fmt.Errorf("could not set properties: %s", err)
	}

	savePromState(machine.TextFileDirectory(), mw)

	return mw, nil
}

func (w *Watcher) Delete() {
	err := deletePromState(w.machine.TextFileDirectory(), w, w.machine.Name(), w.name)
	if err != nil {
		w.Errorf("could not delete from prometheus: %s", err)
	}
}

func (w *Watcher) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	w.Infof("metric watcher for %s starting", w.properties.Command)

	splay := time.Duration(rand.Intn(int(w.properties.Interval.Seconds()))) * time.Second
	w.Infof("Splaying first check by %v", splay)

	select {
	case <-time.NewTimer(splay).C:
		w.performWatch(ctx)
	case <-ctx.Done():
		return
	}

	tick := time.NewTicker(w.properties.Interval)

	for {
		select {
		case <-tick.C:
			w.performWatch(ctx)

		case <-w.StateChangeC():
			w.performWatch(ctx)

		case <-ctx.Done():
			w.Infof("Stopping on context interrupt")
			tick.Stop()
			return
		}
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

func (w *Watcher) watch(ctx context.Context) (state []byte, err error) {
	if !w.ShouldWatch() {
		return nil, nil
	}

	w.startWatching()
	defer w.stopWatching()

	start := time.Now()
	defer func() {
		w.mu.Lock()
		w.previousRunTime = time.Since(start)
		w.mu.Unlock()
	}()

	w.Infof("Running %s", w.properties.Command)

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	splitcmd, err := shlex.Split(w.properties.Command)
	if err != nil {
		w.Errorf("Metric watcher %s failed: %s", w.properties.Command, err)
		return nil, err
	}

	cmd := exec.CommandContext(timeoutCtx, splitcmd[0], splitcmd[1:]...)
	cmd.Env = append(cmd.Env, fmt.Sprintf("MACHINE_WATCHER_NAME=%s", w.name))
	cmd.Env = append(cmd.Env, fmt.Sprintf("MACHINE_NAME=%s", w.machine.Name()))
	cmd.Env = append(cmd.Env, fmt.Sprintf("PATH=%s%s%s", os.Getenv("PATH"), string(os.PathListSeparator), w.machine.Directory()))
	cmd.Dir = w.machine.Directory()

	output, err := cmd.CombinedOutput()
	if err != nil {
		w.Errorf("Metric watcher %s failed: %s", w.properties.Command, err)
		return nil, err
	}

	w.Debugf("Output from %s: %s", w.properties.Command, output)

	return output, nil
}

func (w *Watcher) performWatch(ctx context.Context) {
	if w.isWatching() {
		return
	}

	metric, err := w.watch(ctx)
	err = w.handleCheck(metric, err)
	if err != nil {
		w.Errorf("could not handle watcher event: %s", err)
	}
}

func (w *Watcher) parseJSONCheck(output []byte) (*Metric, error) {
	metric := &Metric{
		Labels:  map[string]string{"format": "choria"},
		Metrics: map[string]float64{},
	}

	err := json.Unmarshal(output, metric)
	if err != nil {
		return metric, err
	}

	for k, v := range w.properties.Labels {
		metric.Labels[k] = v
	}

	return metric, nil
}

func (w *Watcher) parseNagiosCheck(output []byte) (*Metric, error) {
	metric := &Metric{
		Labels:  map[string]string{"format": "nagios"},
		Metrics: map[string]float64{},
	}

	perf := util.ParsePerfData(string(output))
	if perf == nil {
		return metric, nil
	}

	for _, p := range perf {
		metric.Metrics[p.Label] = p.Value
	}

	return metric, nil
}

func (w *Watcher) handleCheck(output []byte, err error) error {
	var metric *Metric

	if err == nil {
		if bytes.HasPrefix(bytes.TrimSpace(output), []byte("{")) {
			metric, err = w.parseJSONCheck(output)
		} else {
			metric, err = w.parseNagiosCheck(output)
		}
	}

	if err != nil {
		w.NotifyWatcherState(w.CurrentState())
		return w.FailureTransition()
	}

	for k, v := range w.properties.Labels {
		metric.Labels[k] = v
	}

	err = updatePromState(w.machine.TextFileDirectory(), w, w.machine.Name(), w.name, metric)
	if err != nil {
		w.Errorf("Could not update prometheus: %s", err)
	}

	w.mu.Lock()
	w.previousResult = metric
	w.mu.Unlock()

	w.NotifyWatcherState(w.CurrentState())

	return nil
}

func (w *Watcher) CurrentState() interface{} {
	w.mu.Lock()
	defer w.mu.Unlock()

	var res Metric
	if w.previousResult == nil {
		res = Metric{
			Labels:  make(map[string]string),
			Metrics: make(map[string]float64),
		}
	} else {
		res = *w.previousResult
	}

	res.Metrics["choria_runtime_seconds"] = w.previousRunTime.Seconds()

	s := &StateNotification{
		Event:   event.New(w.name, wtype, version, w.machine),
		Metrics: res,
	}

	return s
}

func (w *Watcher) validate() error {
	if w.properties.Command == "" {
		return fmt.Errorf("command is required")
	}

	if w.properties.Interval < time.Second {
		w.properties.Interval = time.Second
	}

	return nil
}

func (w *Watcher) setProperties(props map[string]interface{}) error {
	if w.properties == nil {
		w.properties = &properties{
			Labels: make(map[string]string),
		}
	}

	err := util.ParseMapStructure(props, w.properties)
	if err != nil {
		return err
	}

	return w.validate()
}
