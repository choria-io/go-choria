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
	"github.com/choria-io/go-choria/aagent/watchers/watcher"
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

	sync.Mutex
}

func New(machine watcher.Machine, name string, states []string, failEvent string, successEvent string, ai time.Duration, rawprops map[string]interface{}) (*Watcher, error) {
	var err error

	w := &Watcher{
		name:    name,
		machine: machine,
	}

	w.Watcher, err = watcher.NewWatcher(name, "metric", ai, states, machine, failEvent, successEvent)
	if err != nil {
		return nil, err
	}

	err = w.setProperties(rawprops)
	if err != nil {
		return nil, fmt.Errorf("could not set properties: %s", err)
	}

	savePromState(machine.TextFileDirectory(), w)

	return w, nil
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

func (w *Watcher) watch(ctx context.Context) (state []byte, err error) {
	if !w.ShouldWatch() {
		return nil, nil
	}

	start := time.Now()
	defer func() {
		w.Lock()
		w.previousRunTime = time.Since(start)
		w.Unlock()
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
	metric, err := w.watch(ctx)
	err = w.handleCheck(metric, err)
	if err != nil {
		w.Errorf("could not handle watcher event: %s", err)
	}
}

func (w *Watcher) handleCheck(output []byte, err error) error {
	metric := &Metric{Labels: map[string]string{}}

	if err == nil {
		err = json.Unmarshal(output, metric)
	}

	for k, v := range w.properties.Labels {
		metric.Labels[k] = v
	}

	if err != nil {
		w.NotifyWatcherState(w.name, w.CurrentState())
		return w.Transition(w.FailEvent())
	}

	err = updatePromState(w.machine.TextFileDirectory(), w, w.machine.Name(), w.name, metric)
	if err != nil {
		w.Errorf("Could not update prometheus: %s", err)
	}

	w.Lock()
	w.previousResult = metric
	w.Unlock()

	w.NotifyWatcherState(w.name, w.CurrentState())

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
		Event:   event.New(w.name, "metric", "v1", w.machine),
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
