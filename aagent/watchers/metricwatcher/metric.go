package metricwatcher

import (
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
)

type Machine interface {
	State() string
	Name() string
	Directory() string
	Identity() string
	InstanceID() string
	Version() string
	TimeStampSeconds() int64
	NotifyWatcherState(string, interface{})
	TextFileDirectory() string
	Transition(t string, args ...interface{}) error
	Debugf(name string, format string, args ...interface{})
	Infof(name string, format string, args ...interface{})
	Errorf(name string, format string, args ...interface{})
}

type Metric struct {
	Labels  map[string]string  `json:"labels"`
	Metrics map[string]float64 `json:"metrics"`
	name    string
	machine string
	seen    int
}

type Watcher struct {
	name             string
	states           []string
	machine          Machine
	announceInterval time.Duration
	failEvent        string
	command          string        `mapstructure:"command"`
	checkInterval    time.Duration `mapstructure:"interval"`
	previousRunTime  time.Duration
	previousResult   *Metric
	statechg         chan struct{}

	sync.Mutex
}

func New(machine Machine, name string, states []string, failEvent string, successEvent string, ai time.Duration, properties map[string]interface{}) (watcher *Watcher, err error) {
	w := &Watcher{
		name:             name,
		states:           states,
		machine:          machine,
		announceInterval: ai,
		failEvent:        failEvent,
		checkInterval:    time.Minute,
		statechg:         make(chan struct{}),
	}

	err = w.setProperties(properties)
	if err != nil {
		return nil, fmt.Errorf("could not set properties: %s", err)
	}

	savePromState(machine.TextFileDirectory(), machine)

	return w, nil
}

func (w *Watcher) Delete() {
	err := deletePromState(w.machine.TextFileDirectory(), w.machine, w.machine.Name(), w.name)
	if err != nil {
		w.machine.Errorf(w.name, "could not delete from prometheus: %s", err)
	}
}

func (w *Watcher) Type() string {
	return "metric"
}

func (w *Watcher) AnnounceInterval() time.Duration {
	w.Lock()
	defer w.Unlock()

	return w.announceInterval
}

func (w *Watcher) Name() string {
	return w.name
}

func (w *Watcher) NotifyStateChance() {
	w.Lock()
	defer w.Unlock()

	if len(w.statechg) < cap(w.statechg) {
		w.statechg <- struct{}{}
	}
}

func (w *Watcher) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	w.machine.Infof(w.name, "metric watcher for %s starting", w.command)

	splay := time.Duration(rand.Intn(int(w.checkInterval.Seconds()))) * time.Second
	w.machine.Infof(w.name, "Splaying first check by %v", splay)

	select {
	case <-time.NewTimer(splay).C:
		w.performWatch(ctx)
	case <-ctx.Done():
		return
	}

	tick := time.NewTicker(w.checkInterval)

	for {
		select {
		case <-tick.C:
			w.performWatch(ctx)

		case <-w.statechg:
			w.performWatch(ctx)

		case <-ctx.Done():
			w.machine.Infof(w.name, "Stopping on context interrupt")
			tick.Stop()
			return
		}
	}
}

func (w *Watcher) watch(ctx context.Context) (state []byte, err error) {
	if !w.shouldWatch() {
		return nil, nil
	}

	start := time.Now()
	defer func() {
		w.Lock()
		w.previousRunTime = time.Since(start)
		w.Unlock()
	}()

	w.machine.Infof(w.name, "Running %s", w.command)

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	splitcmd, err := shlex.Split(w.command)
	if err != nil {
		w.machine.Errorf(w.name, "Metric watcher %s failed: %s", w.command, err)
		return nil, err
	}

	cmd := exec.CommandContext(timeoutCtx, splitcmd[0], splitcmd[1:]...)
	cmd.Env = append(cmd.Env, fmt.Sprintf("MACHINE_WATCHER_NAME=%s", w.name))
	cmd.Env = append(cmd.Env, fmt.Sprintf("MACHINE_NAME=%s", w.machine.Name()))
	cmd.Env = append(cmd.Env, fmt.Sprintf("PATH=%s%s%s", os.Getenv("PATH"), string(os.PathListSeparator), w.machine.Directory()))
	cmd.Dir = w.machine.Directory()

	output, err := cmd.CombinedOutput()
	if err != nil {
		w.machine.Errorf(w.name, "Metric watcher %s failed: %s", w.command, err)
		return nil, err
	}

	w.machine.Debugf(w.name, "Output from %s: %s", w.command, output)

	return output, nil
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

func (w *Watcher) performWatch(ctx context.Context) {
	metric, err := w.watch(ctx)
	err = w.handleCheck(metric, err)
	if err != nil {
		w.machine.Errorf(w.name, "could not handle watcher event: %s", err)
	}
}

func (w *Watcher) handleCheck(output []byte, err error) error {
	metric := &Metric{}

	if err == nil {
		err = json.Unmarshal(output, metric)
	}

	if err != nil {
		w.machine.NotifyWatcherState(w.name, w.CurrentState())
		return w.machine.Transition(w.failEvent)
	}

	err = updatePromState(w.machine.TextFileDirectory(), w.machine, w.machine.Name(), w.name, metric)
	if err != nil {
		w.machine.Errorf(w.name, "Could not update prometheus: %s", err)
	}

	w.Lock()
	w.previousResult = metric
	w.Unlock()

	w.machine.NotifyWatcherState(w.name, w.CurrentState())
	return nil
}

func (w *Watcher) CurrentState() interface{} {
	w.Lock()
	defer w.Unlock()

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
		Event: event.Event{
			Protocol:  "io.choria.machine.watcher.metric.v1.state",
			Type:      "metric",
			Name:      w.name,
			Identity:  w.machine.Identity(),
			ID:        w.machine.InstanceID(),
			Version:   w.machine.Version(),
			Timestamp: w.machine.TimeStampSeconds(),
			Machine:   w.machine.Name(),
		},
		Metrics: res,
	}

	return s
}

func (w *Watcher) validate() error {
	if w.command == "" {
		return fmt.Errorf("command is required")
	}

	if w.checkInterval < time.Second {
		w.checkInterval = time.Second
	}

	return nil
}

func (w *Watcher) setProperties(props map[string]interface{}) error {
	var properties struct {
		Command  string
		Interval time.Duration
	}

	err := util.ParseMapStructure(props, &properties)
	if err != nil {
		return err
	}

	w.command = properties.Command
	w.checkInterval = properties.Interval

	return w.validate()
}
