package kvwatcher

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/choria-io/go-choria/aagent/model"
	"github.com/choria-io/go-choria/aagent/util"
	"github.com/choria-io/go-choria/aagent/watchers/event"
	"github.com/choria-io/go-choria/aagent/watchers/watcher"
	iu "github.com/choria-io/go-choria/internal/util"
	"github.com/nats-io/jsm.go/kv"
)

type State int

const (
	Error State = iota
	Changed
	Unchanged
	Skipped

	wtype     = "kv"
	version   = "v1"
	pollMode  = "poll"
	watchMode = "watch"
)

var stateNames = map[State]string{
	Error:     "error",
	Changed:   "changed",
	Unchanged: "unchanged",
	Skipped:   "skipped",
}

type properties struct {
	Bucket            string
	Key               string
	Mode              string
	TransitionOnMatch bool `mapstructure:"on_matching_update"`
	BucketPrefix      bool `mapstructure:"bucket_prefix"`
}

type Watcher struct {
	*watcher.Watcher
	properties *properties

	name     string
	machine  model.Machine
	kv       kv.RoKV
	interval time.Duration

	previousVal   interface{}
	previousSeq   uint64
	previousState State
	polling       bool
	lastPoll      time.Time

	terminate chan struct{}
	mu        *sync.Mutex
}

func New(machine model.Machine, name string, states []string, failEvent string, successEvent string, interval string, ai time.Duration, properties map[string]interface{}) (interface{}, error) {
	var err error

	tw := &Watcher{
		name:      name,
		machine:   machine,
		terminate: make(chan struct{}),
		mu:        &sync.Mutex{},
	}

	tw.interval, err = iu.ParseDuration(interval)
	if err != nil {
		return nil, err
	}

	tw.Watcher, err = watcher.NewWatcher(name, wtype, ai, states, machine, failEvent, successEvent)
	if err != nil {
		return nil, err
	}

	err = tw.setProperties(properties)
	if err != nil {
		return nil, fmt.Errorf("could not set properties: %s", err)
	}

	mgr, err := machine.JetStreamConnection()
	if err != nil {
		return nil, err
	}

	tw.kv, err = kv.NewRoClient(mgr.NatsConn(), tw.properties.Bucket)
	if err != nil {
		return nil, err
	}

	return tw, nil
}

func (w *Watcher) setProperties(props map[string]interface{}) error {
	if w.properties == nil {
		w.properties = &properties{
			BucketPrefix: true,
		}
	}

	err := util.ParseMapStructure(props, w.properties)
	if err != nil {
		return err
	}

	return w.validate()
}

func (w *Watcher) validate() error {
	if w.properties.Bucket == "" {
		return fmt.Errorf("bucket is required")
	}

	if w.properties.Mode == "" {
		w.properties.Mode = pollMode
	}

	if w.properties.Mode != pollMode && w.properties.Mode != watchMode {
		return fmt.Errorf("mode should be '%s' or '%s'", pollMode, watchMode)
	}

	if w.properties.Mode == pollMode && w.properties.Key == "" {
		return fmt.Errorf("poll mode requires a key")
	}

	if w.properties.Mode == watchMode {
		return fmt.Errorf("watch mode not supported")
	}

	return nil
}

func (w *Watcher) Delete() {
	close(w.terminate)
}

func (w *Watcher) stopPolling() {
	w.mu.Lock()
	w.polling = false
	w.mu.Unlock()
}

func (w *Watcher) poll() (State, error) {
	if !w.ShouldWatch() {
		return Skipped, nil
	}

	w.mu.Lock()
	if w.polling {
		w.mu.Unlock()
		return Skipped, nil
	}
	w.polling = true
	w.mu.Unlock()

	defer w.stopPolling()

	lp := w.lastPoll
	since := time.Since(lp).Round(time.Second)
	if since < w.interval {
		w.Debugf("Skipping watch due to last watch %v ago", since)
		return Skipped, nil
	}
	w.lastPoll = time.Now()

	w.Infof("Polling for %s.%s", w.properties.Bucket, w.properties.Key)

	dk := w.dataKey()
	val, err := w.kv.Get(w.properties.Key)

	switch {
	// key isn't there, nothing was previously found its unchanged
	case err == kv.ErrUnknownKey && w.previousVal == "":
		return Unchanged, nil

	// key isn't there, we had a value before its a change due to delete
	case err == kv.ErrUnknownKey && w.previousVal != "":
		err = w.machine.DataDelete(dk)
		if err != nil {
			w.Errorf("Could not delete key %s from machine: %s", dk, err)
			return Error, err
		}

		w.previousVal = ""
		return Changed, err

	// get failed in an unknown way
	case err != nil:
		w.Errorf("Could not get %s.%s: %s", w.properties.Bucket, w.properties.Key, err)
		return Error, err

	// a change
	case w.previousVal != string(val.Value()):
		err = w.machine.DataPut(dk, string(val.Value()))
		if err != nil {
			return Error, err
		}

		w.previousSeq = val.Sequence()
		w.previousVal = string(val.Value())
		return Changed, nil

	// a put that didnt update, but we are asked to transition anyway
	// we do not trigger this on first start of the machine only once its running (previousSeq is 0)
	case w.previousVal == string(val.Value()) && w.properties.TransitionOnMatch && w.previousSeq > 0 && val.Sequence() > w.previousSeq:
		w.previousSeq = val.Sequence()
		return Changed, nil

	default:
		w.previousSeq = val.Sequence()
		return Unchanged, nil
	}
}

func (w *Watcher) handleState(s State, err error) error {
	w.Debugf("handling state for %s.%s: %s: %s", w.properties.Bucket, w.properties.Key, stateNames[s], err)

	w.mu.Lock()
	w.previousState = s
	w.mu.Unlock()

	switch s {
	case Error:
		return w.FailureTransition()
	case Changed:
		return w.SuccessTransition()
	case Unchanged, Skipped:
	}

	return nil
}

func (w *Watcher) dataKey() string {
	if w.properties.BucketPrefix {
		return fmt.Sprintf("%s_%s", w.properties.Bucket, w.properties.Key)
	}

	return w.properties.Key
}

func (w *Watcher) pollKey(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	dk := w.dataKey()
	w.previousVal, _ = w.machine.DataGet(dk)

	w.handleState(w.poll())

	ticker := time.NewTicker(w.interval)

	for {
		select {
		case <-ticker.C:
			w.handleState(w.poll())

		case <-ctx.Done():
			return
		}
	}
}

func (w *Watcher) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	if w.properties.Key == "" {
		w.Infof("Key-Value watcher starting with bucket %q in %q mode", w.properties.Bucket, w.properties.Mode)
	} else {
		w.Infof("Key-Value watcher starting with bucket %q and key %q in %q mode", w.properties.Bucket, w.properties.Key, w.properties.Mode)
	}

	watchCtx, watchCancel := context.WithCancel(ctx)
	defer watchCancel()

	switch w.properties.Mode {
	case watchMode:
		// TODO: set up watcher

	case pollMode:
		wg.Add(1)
		go w.pollKey(watchCtx, wg)
	}

	for {
		select {
		case <-w.StateChangeC():
			w.handleState(w.poll())

		case <-w.terminate:
			w.Infof("Handling terminate notification")
			watchCancel()
			return
		case <-ctx.Done():
			w.Infof("Stopping on context interrupt")
			return
		}
	}
}

func (w *Watcher) CurrentState() interface{} {
	w.mu.Lock()
	defer w.mu.Unlock()

	s := &StateNotification{
		Event:  event.New(w.name, wtype, version, w.machine),
		State:  stateNames[w.previousState],
		Key:    w.properties.Key,
		Bucket: w.properties.Bucket,
		Mode:   w.properties.Mode,
	}

	return s
}
