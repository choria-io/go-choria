// Copyright (c) 2021-2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package kvwatcher

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/choria-io/go-choria/aagent/model"
	"github.com/choria-io/go-choria/aagent/util"
	"github.com/choria-io/go-choria/aagent/watchers/event"
	"github.com/choria-io/go-choria/aagent/watchers/watcher"
	iu "github.com/choria-io/go-choria/internal/util"
	"github.com/choria-io/go-choria/providers/kv"
	"github.com/choria-io/tinyhiera"
	"github.com/google/go-cmp/cmp"
	"github.com/nats-io/nats.go"
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
	Bucket                    string
	Key                       string
	Mode                      string
	TransitionOnSuccessfulGet bool   `mapstructure:"on_successful_get"`
	TransitionOnMatch         bool   `mapstructure:"on_matching_update"`
	BucketPrefix              bool   `mapstructure:"bucket_prefix"`
	RepublishTrigger          string `mapstructure:"republish_trigger"`
	HieraConfig               bool   `mapstructure:"hiera_config"`
}

type Watcher struct {
	*watcher.Watcher
	properties *properties

	name     string
	machine  model.Machine
	kv       nats.KeyValue
	sub      *nats.Subscription
	interval time.Duration

	previousVal   any
	previousSeq   uint64
	previousState State
	polling       bool
	lastPoll      time.Time

	terminate chan struct{}
	mu        *sync.Mutex
}

func New(machine model.Machine, name string, states []string, required []model.ForeignMachineState, failEvent string, successEvent string, interval string, ai time.Duration, properties map[string]any) (any, error) {
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

	tw.Watcher, err = watcher.NewWatcher(name, wtype, ai, states, required, machine, failEvent, successEvent)
	if err != nil {
		return nil, err
	}

	err = tw.setProperties(properties)
	if err != nil {
		return nil, fmt.Errorf("could not set properties: %s", err)
	}

	return tw, nil
}

func (w *Watcher) setProperties(props map[string]any) error {
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

func (w *Watcher) connectKV() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	var err error
	mgr, err := w.machine.JetStreamConnection()
	if err != nil {
		return err
	}

	w.kv, err = kv.NewKV(mgr.NatsConn(), w.properties.Bucket, false)
	if err != nil {
		return err
	}

	return nil
}

func (w *Watcher) setupRepubListener(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	var err error
	mgr, err := w.machine.JetStreamConnection()
	if err != nil {
		return err
	}
	nc := mgr.NatsConn()

	parsed, err := w.ProcessTemplate(w.properties.RepublishTrigger)
	if err != nil {
		return fmt.Errorf("could not parse template for republish trigger: %v", err)
	}

	w.Infof("Setting up republish listener on %q", parsed)

	w.sub, err = nc.Subscribe(parsed, func(msg *nats.Msg) {
		if !iu.IsNATSRepublishHeaders(msg.Header) {
			w.Warnf("Received non republished message on %q: %s", msg.Subject, string(msg.Data))
			return
		}

		state, err := w.repubHandler(msg)
		w.Infof("Republish handler returned state %s: %v", stateNames[state], err)
		w.handleState(state, err)
	})

	if err == nil {
		go func() {
			<-ctx.Done()
			w.mu.Lock()
			w.sub.Unsubscribe()
			w.sub = nil
			w.mu.Unlock()
		}()
	}

	return err
}

// handles a published message as if its kv poll result, logic update here should also be changed in poll()
func (w *Watcher) repubHandler(msg *nats.Msg) (State, error) {
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

	w.Infof("Received republish trigger on subject %s", msg.Subject)

	defer w.stopPolling()

	parsedKey, err := w.ProcessTemplate(w.properties.Key)
	if err != nil {
		return Error, fmt.Errorf("could not parse template for key: %v", err)
	}

	if !strings.HasSuffix(msg.Subject, parsedKey) {
		return Error, fmt.Errorf("received republish message on %s that does not match expected key %s", msg.Subject, parsedKey)
	}

	dk := w.dataKey()
	if w.previousVal == nil {
		w.previousVal, _ = w.machine.DataGet(dk)
	}

	val := msg.Data
	op := msg.Header.Get("KV-Operation")

	seq, err := strconv.ParseUint(msg.Header.Get("Nats-Sequence"), 10, 64)
	if err != nil {
		return Error, err
	}

	parsedValue, err := w.parseValue(val)
	switch {
	case err != nil:
		w.Errorf("Could not parse value %s.%s: %s", w.properties.Bucket, w.properties.Key, err)
		return Error, err

	case (op == "DEL" || op == "PURGE") && w.previousVal == nil:
		return Unchanged, nil

	// deleted
	case (op == "DEL" || op == "PURGE") && w.previousVal != nil:
		w.Debugf("Removing data from %s", dk)
		err = w.machine.DataDelete(dk)
		if err != nil {
			w.Errorf("Could not delete key %s from machine: %s", dk, err)
			return Error, err
		}

		w.previousVal = nil
		w.previousSeq = seq

		return Changed, err

	// a change
	case !cmp.Equal(w.previousVal, parsedValue):
		err = w.machine.DataPut(dk, parsedValue)
		if err != nil {
			return Error, err
		}

		w.previousSeq = seq
		w.previousVal = parsedValue

		return Changed, nil

	// a put that didn't update, but we are asked to transition anyway
	// we do not trigger this on first start of the machine only once its running (previousSeq is 0)
	case cmp.Equal(w.previousVal, parsedValue) && w.properties.TransitionOnMatch && w.previousSeq > 0 && seq > w.previousSeq:
		w.previousSeq = seq

		return Changed, nil

	default:
		w.previousSeq = seq
		if w.properties.TransitionOnSuccessfulGet {
			return Changed, nil
		}

		return Unchanged, nil
	}
}

func (w *Watcher) parseValue(val []byte) (any, error) {
	var parsedValue any

	// we try to handle json files into a map[string]any this means nested lookups can be done
	// in other machines using the lookup template func, and it works just fine, deep compares are done
	// on the entire structure later
	v := bytes.TrimSpace(val)
	if bytes.HasPrefix(v, []byte("{")) && bytes.HasSuffix(v, []byte("}")) {
		parsedMapValue := map[string]any{}
		err := json.Unmarshal(v, &parsedMapValue)
		if err != nil {
			w.Warnf("unmarshal failed: %s", err)
		}

		// the data holds a tiny hiera configuration, we parse and merge it based on facts and use the result as parsed value
		if w.properties.HieraConfig {
			facts := map[string]any{}

			err := json.Unmarshal(w.machine.Facts(), &facts)
			if err != nil {
				return nil, err
			}

			parsedValue, err = tinyhiera.Resolve(parsedMapValue, map[string]any{"facts": facts})
			if err != nil {
				return nil, err
			}
		} else {
			parsedValue = parsedMapValue
		}

	} else if bytes.HasPrefix(v, []byte("[")) && bytes.HasSuffix(v, []byte("]")) {
		parsedValue = []any{}
		err := json.Unmarshal(v, &parsedValue)
		if err != nil {
			w.Warnf("unmarshal failed: %s", err)
		}

		if w.properties.HieraConfig {
			return nil, fmt.Errorf("hiera config not supported for arrays")
		}
	}

	if parsedValue == nil {
		parsedValue = string(val)
	}

	return parsedValue, nil
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
	store := w.kv
	w.mu.Unlock()

	defer w.stopPolling()

	// we try to bind to the store here on every poll so that if the store does not yet exist
	// at startup we will keep trying until it does
	if store == nil {
		err := w.connectKV()
		if err != nil {
			return Error, err
		}
	}

	lp := w.lastPoll
	since := time.Since(lp).Round(time.Second)
	if since < w.interval {
		w.Debugf("Skipping watch due to last watch %v ago", since)
		return Skipped, nil
	}
	w.lastPoll = time.Now()

	parsedKey, err := w.ProcessTemplate(w.properties.Key)
	if err != nil {
		return Error, fmt.Errorf("could not parse template for key: %v", err)
	}

	w.Infof("Polling for %s.%s", w.properties.Bucket, parsedKey)

	var parsedValue any

	dk := w.dataKey()
	if w.previousVal == nil {
		w.previousVal, _ = w.machine.DataGet(dk)
	}

	val, err := w.kv.Get(parsedKey)
	if err == nil {
		var err error // kv.Get() error is checked below so don't overwrite it
		parsedValue, err = w.parseValue(val.Value())
		if err != nil {
			w.Errorf("Could not parse value %s.%s: %s", w.properties.Bucket, parsedKey, err)
			return Error, err
		}
	}

	// if this changes also update repub handler
	switch {
	// key isn't there, nothing was previously found unchanged
	case errors.Is(err, nats.ErrKeyNotFound) && w.previousVal == nil:
		return Unchanged, nil

	// key isn't there, we had a value before its change due to delete
	case errors.Is(err, nats.ErrKeyNotFound) && w.previousVal != nil:
		w.Debugf("Removing data from %s", dk)
		err = w.machine.DataDelete(dk)
		if err != nil {
			w.Errorf("Could not delete key %s from machine: %s", dk, err)
			return Error, err
		}

		w.previousVal = nil
		w.previousSeq = 0

		return Changed, err

	// get failed in an unknown way
	case err != nil:
		w.Errorf("Could not get %s.%s: %s", w.properties.Bucket, parsedKey, err)
		return Error, err

	// a change
	case !cmp.Equal(w.previousVal, parsedValue):
		err = w.machine.DataPut(dk, parsedValue)
		if err != nil {
			return Error, err
		}

		w.previousSeq = val.Revision()
		w.previousVal = parsedValue
		return Changed, nil

	// a put that didn't update, but we are asked to transition anyway
	// we do not trigger this on first start of the machine only once its running (previousSeq is 0)
	case cmp.Equal(w.previousVal, parsedValue) && w.properties.TransitionOnMatch && w.previousSeq > 0 && val.Revision() > w.previousSeq:
		w.previousSeq = val.Revision()
		return Changed, nil

	default:
		w.previousSeq = val.Revision()
		if w.properties.TransitionOnSuccessfulGet {
			return Changed, nil
		}

		return Unchanged, nil
	}
}

func (w *Watcher) handleState(s State, err error) error {
	w.Debugf("handling state for %s.%s: %s: err:%v", w.properties.Bucket, w.properties.Key, stateNames[s], err)

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
	parsedKey, err := w.ProcessTemplate(w.properties.Key)
	if err != nil {
		w.Warnf("Failed to parse key value %s: %v", w.properties.Key, err)
		return w.properties.Key
	}

	if w.properties.BucketPrefix {
		return fmt.Sprintf("%s_%s", w.properties.Bucket, parsedKey)
	}

	return parsedKey
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

	if w.properties.RepublishTrigger != "" {
		err := w.setupRepubListener(ctx)
		if err != nil {
			w.Errorf("Could not set up republish listener: %s", err)
		}
	}

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

func (w *Watcher) CurrentState() any {
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
