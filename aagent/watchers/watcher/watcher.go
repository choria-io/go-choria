package watcher

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/choria-io/go-choria/backoff"
	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/internal/util"
	"github.com/choria-io/go-choria/lifecycle"
	"github.com/nats-io/jsm.go/governor"
)

type Watcher struct {
	name             string
	wtype            string
	announceInterval time.Duration
	statechg         chan struct{}
	activeStates     []string
	machine          Machine
	succEvent        string
	failEvent        string
	data             map[string]string

	deleteCb       func()
	currentStateCb func() interface{}
	govCancel      func()

	dataMu sync.Mutex
	mu     sync.Mutex
}

const dataFileName = "machine_data.json"

func NewWatcher(name string, wtype string, announceInterval time.Duration, activeStates []string, machine Machine, fail string, success string) (*Watcher, error) {
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}

	if wtype == "" {
		return nil, fmt.Errorf("watcher type is required")
	}

	if machine == nil {
		return nil, fmt.Errorf("machine is required")
	}

	w := &Watcher{
		name:             name,
		wtype:            wtype,
		announceInterval: announceInterval,
		statechg:         make(chan struct{}, 1),
		failEvent:        fail,
		succEvent:        success,
		machine:          machine,
		activeStates:     activeStates,
		data:             map[string]string{},
	}

	err := w.loadData()
	if err != nil {
		w.Errorf("Could not load data, continuing: %s", err)
	}

	return w, nil
}

func (w *Watcher) FactsFile() (string, error) {
	tf, err := os.CreateTemp("", "")
	if err != nil {
		return "", err
	}

	_, err = tf.Write(w.machine.Facts())
	if err != nil {
		tf.Close()
		os.Remove(tf.Name())
		return "", err
	}
	tf.Close()

	return tf.Name(), nil
}

func (w *Watcher) DataCopyFile() (string, error) {
	dat := w.dataCopy()
	if len(dat) == 0 {
		return "", nil
	}

	j, err := json.Marshal(dat)
	if err != nil {
		return "", err
	}

	tf, err := os.CreateTemp("", "")
	if err != nil {
		return "", err
	}

	_, err = tf.Write(j)
	if err != nil {
		tf.Close()
		os.Remove(tf.Name())
		return "", err
	}
	tf.Close()

	return tf.Name(), nil
}

func (w *Watcher) CancelGovernor() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.govCancel != nil {
		w.govCancel()
	}
}

func (w *Watcher) sendGovernorLC(t lifecycle.GovernorEventType, name string, seq uint64) {
	w.machine.PublishLifecycleEvent(lifecycle.Governor,
		lifecycle.Identity(w.machine.Identity()),
		lifecycle.Component(w.machine.Name()),
		lifecycle.GovernorType(t),
		lifecycle.GovernorSequence(seq),
		lifecycle.GovernorName(name))
}

func (w *Watcher) EnterGovernor(ctx context.Context, name string, timeout time.Duration) (governor.Finisher, error) {
	mgr, err := w.machine.JetStreamConnection()
	if err != nil {
		return nil, fmt.Errorf("JetStream connection not set")
	}

	w.Infof("Obtaining a slot in the %s Governor with %v timeout", name, timeout)
	subj := choria.GovernorSubject(name, w.machine.MainCollective())
	gov := governor.NewJSGovernor(name, mgr, governor.WithLogger(w), governor.WithSubject(subj), governor.WithBackoff(backoff.FiveSec))

	var gCtx context.Context
	w.mu.Lock()
	gCtx, w.govCancel = context.WithTimeout(ctx, timeout)
	w.mu.Unlock()
	defer w.govCancel()

	fin, seq, err := gov.Start(gCtx, fmt.Sprintf("Auto Agent  %s#%s @ %s", w.machine.Name(), w.name, w.machine.Identity()))
	if err != nil {
		w.Errorf("Could not obtain a slot in the Governor %s: %s", name, err)
		w.sendGovernorLC(lifecycle.GovernorTimeoutEvent, name, 0)
		return nil, err
	}

	w.sendGovernorLC(lifecycle.GovernorEnterEvent, name, seq)

	finisher := func() error {
		w.sendGovernorLC(lifecycle.GovernorExitEvent, name, seq)
		return fin()
	}

	return finisher, nil
}

func (w *Watcher) dataCopy() map[string]string {
	w.dataMu.Lock()
	defer w.dataMu.Unlock()

	res := make(map[string]string, len(w.data))
	for k, v := range w.data {
		res[k] = v
	}

	return res
}

func (w *Watcher) DataGet(key string) (string, bool) {
	w.dataMu.Lock()
	defer w.dataMu.Unlock()

	v, ok := w.data[key]

	return v, ok
}

func (w *Watcher) DataPut(key string, val string) {
	w.dataMu.Lock()
	defer w.dataMu.Unlock()

	w.data[key] = val

	err := w.saveData()
	if err != nil {
		w.Errorf("Could not save data to %s: %s", dataFileName, err)
	}
}

func (w *Watcher) loadData() error {
	path := filepath.Join(w.machine.Directory(), dataFileName)
	if !util.FileExist(path) {
		return nil
	}

	j, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	w.dataMu.Lock()
	defer w.dataMu.Unlock()

	return json.Unmarshal(j, &w.data)
}

// lock should be held by caller
func (w *Watcher) saveData() error {
	if len(w.data) == 0 {
		return nil
	}

	j, err := json.Marshal(w.data)
	if err != nil {
		return err
	}

	tf, err := os.CreateTemp("", "")
	if err != nil {
		return err
	}
	defer os.Remove(tf.Name())

	_, err = tf.Write(j)
	tf.Close()
	if err != nil {
		return err
	}

	return os.Rename(tf.Name(), filepath.Join(w.machine.Directory(), dataFileName))
}

func (w *Watcher) Machine() Machine {
	return w.machine
}

func (w *Watcher) SuccessEvent() string {
	return w.succEvent
}

func (w *Watcher) FailEvent() string {
	return w.failEvent
}

func (w *Watcher) StateChangeC() chan struct{} {
	return w.statechg
}

func (w *Watcher) SetDeleteFunc(f func()) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.deleteCb = f
}

func (w *Watcher) NotifyWatcherState(state interface{}) {
	w.machine.NotifyWatcherState(w.name, state)
}

func (w *Watcher) SuccessTransition() error {
	if w.succEvent == "" {
		return nil
	}

	w.Infof("success transitioning using %s event", w.succEvent)
	return w.machine.Transition(w.succEvent)
}

func (w *Watcher) FailureTransition() error {
	if w.failEvent == "" {
		return nil
	}

	w.Infof("fail transitioning using %s event", w.succEvent)
	return w.machine.Transition(w.failEvent)
}

func (w *Watcher) Transition(event string) error {
	if event == "" {
		return nil
	}

	return w.machine.Transition(event)
}

func (w *Watcher) NotifyStateChance() {
	w.mu.Lock()
	defer w.mu.Unlock()

	select {
	case w.statechg <- struct{}{}:
	default:
	}
}

func (w *Watcher) CurrentState() interface{} {
	if w.currentStateCb != nil {
		return w.currentStateCb()
	}

	return nil
}

func (w *Watcher) AnnounceInterval() time.Duration {
	return w.announceInterval
}

func (w *Watcher) Type() string {
	return w.wtype
}

func (w *Watcher) Name() string {
	return w.name
}

func (w *Watcher) Delete() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.deleteCb != nil {
		w.deleteCb()
	}
}

func (w *Watcher) ShouldWatch() bool {
	if len(w.activeStates) == 0 {
		return true
	}

	for _, e := range w.activeStates {
		if e == w.machine.State() {
			return true
		}
	}

	return false
}

func (w *Watcher) Debugf(format string, args ...interface{}) {
	w.machine.Debugf(w.name, format, args...)
}

func (w *Watcher) Infof(format string, args ...interface{}) {
	w.machine.Infof(w.name, format, args...)
}

func (w *Watcher) Warnf(format string, args ...interface{}) {
	w.machine.Warnf(w.name, format, args...)
}

func (w *Watcher) Errorf(format string, args ...interface{}) {
	w.machine.Errorf(w.name, format, args...)
}
