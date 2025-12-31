// Copyright (c) 2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package ccmmanifestwatcher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/goccy/go-yaml"

	"github.com/choria-io/ccm/manager"
	ccmmodel "github.com/choria-io/ccm/model"
	"github.com/choria-io/ccm/resources/apply"
	"github.com/choria-io/go-choria/aagent/model"
	"github.com/choria-io/go-choria/aagent/util"
	"github.com/choria-io/go-choria/aagent/watchers/event"
	"github.com/choria-io/go-choria/aagent/watchers/watcher"
	iu "github.com/choria-io/go-choria/internal/util"
)

type State int

const (
	Unknown State = iota
	Error
	Stable
	Changes
	Skipped

	wtype   = "ccm_manifest"
	version = "v1"
)

var stateNames = map[State]string{
	Unknown: "unknown",
	Error:   "error",
	Stable:  "stable",
	Changes: "changes",
	Skipped: "skipped",
}

type Properties struct {
	ManifestFile    string         `mapstructure:"manifest_file"`
	Manifest        map[string]any `mapstructure:"manifest"`
	Noop            bool           `mapstructure:"noop"`
	HealthCheckOnly bool           `mapstructure:"healthcheck_only"`
	Governor        string         `mapstructure:"governor"`
	GovernorTimeout time.Duration  `mapstructure:"governor_timeout"`
	Splay           bool           `mapstructure:"splay"`
	Timeout         time.Duration
}

type Watcher struct {
	*watcher.Watcher

	name            string
	machine         model.Machine
	previous        State
	interval        time.Duration
	previousRunTime time.Duration
	properties      *Properties

	lastWatch time.Time

	wmu *sync.Mutex
	mu  *sync.Mutex
}

func New(machine model.Machine, name string, states []string, required []model.ForeignMachineState, failEvent string, successEvent string, interval string, ai time.Duration, rawprop map[string]any) (any, error) {
	var err error

	ccm := &Watcher{
		machine: machine,
		name:    name,
		mu:      &sync.Mutex{},
		wmu:     &sync.Mutex{},
	}

	ccm.Watcher, err = watcher.NewWatcher(name, wtype, ai, states, required, machine, failEvent, successEvent)
	if err != nil {
		return nil, err
	}

	err = ccm.setProperties(rawprop)
	if err != nil {
		return nil, fmt.Errorf("could not set properties: %v", err)
	}

	if interval != "" {
		ccm.interval, err = iu.ParseDuration(interval)
		if err != nil {
			return nil, fmt.Errorf("invalid interval: %v", err)
		}

		if ccm.interval < 500*time.Millisecond {
			return nil, fmt.Errorf("interval %v is too small", ccm.interval)
		}
	}

	return ccm, nil
}

func (w *Watcher) setProperties(props map[string]any) error {
	if w.properties == nil {
		w.properties = &Properties{}
	}

	err := util.ParseMapStructure(props, w.properties)
	if err != nil {
		return err
	}

	return w.validate()
}

func (w *Watcher) validate() error {
	if w.properties.ManifestFile == "" && len(w.properties.Manifest) == 0 {
		return fmt.Errorf("manifest_file or manifest is required")
	}

	if w.properties.Timeout == 0 {
		w.properties.Timeout = time.Minute
	}

	if w.properties.Governor != "" && w.properties.GovernorTimeout == 0 {
		w.Infof("Setting Governor timeout to 5 minutes while unset")
		w.properties.GovernorTimeout = 5 * time.Minute
	}

	return nil
}

func (w *Watcher) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	if len(w.properties.Manifest) > 0 {
		w.Infof("Watcher for embedded manifest starting")
	} else {
		w.Infof("Watcher for %s starting", w.properties.ManifestFile)
	}

	if w.interval != 0 {
		wg.Add(1)
		go w.intervalWatcher(ctx, wg)
	}

	for {
		select {
		case <-w.Watcher.StateChangeC():
			w.performWatch(ctx, true)

		case <-ctx.Done():
			w.Infof("Stopping on context interrupt")
			w.CancelGovernor()
			return
		}
	}
}

func (w *Watcher) intervalWatcher(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	tick := time.NewTicker(time.Millisecond)
	if w.properties.Splay {
		splay := rand.N(30 * time.Second)
		w.Infof("Performing initial execution after %v", splay)
		if splay < 1 {
			splay = 1
		}
		tick.Reset(splay)
	}

	for {
		select {
		case <-tick.C:
			w.performWatch(ctx, false)
			tick.Reset(w.interval)

		case <-ctx.Done():
			tick.Stop()
			return
		}
	}
}

func (w *Watcher) performWatch(ctx context.Context, force bool) {
	w.wmu.Lock()
	defer w.wmu.Unlock()

	if !force && time.Since(w.lastWatch) < w.interval {
		return
	}

	err := w.handleCheck(w.watch(ctx))
	if err != nil {
		w.Errorf("could not handle watcher event: %s", err)
	}
}

func (w *Watcher) watch(ctx context.Context) (state State, err error) {
	if !w.ShouldWatch() {
		return Skipped, nil
	}

	var manifest io.Reader
	var manifestType string

	if len(w.properties.Manifest) > 0 {
		y, err := yaml.Marshal(w.properties.Manifest)
		if err != nil {
			w.Errorf("Could not marshal manifest: %v", err)
			return Error, fmt.Errorf("invalid manifest: %w", err)
		}
		manifest = bytes.NewReader(y)
		manifestType = "embedded"
	} else {
		y, err := os.ReadFile(filepath.Join(w.machine.Directory(), filepath.Base(w.properties.ManifestFile)))
		if err != nil {
			w.Errorf("Could not read manifest file: %v", err)
			return Error, fmt.Errorf("invalid manifest file: %w", err)
		}
		manifest = bytes.NewReader(y)
		manifestType = w.properties.ManifestFile
	}

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

	w.Infof("Applying %s manifest", manifestType)

	timeoutCtx, cancel := context.WithTimeout(ctx, w.properties.Timeout)
	defer cancel()

	mgr, ccmLog, err := w.ccmManager(w.machine.Data(), w.machine.Facts())
	if err != nil {
		return 0, err
	}

	_, a, err := apply.ResolveManifestReader(timeoutCtx, mgr, w.machine.Directory(), manifest)
	if err != nil {
		w.Errorf("Could not resolve manifest: %v", err)
		return Error, fmt.Errorf("could not resolve manifest: %w", err)
	}

	_, err = a.Execute(timeoutCtx, mgr, w.properties.HealthCheckOnly, ccmLog)
	if err != nil {
		w.Errorf("Could not apply manifest: %v", err)
		return Error, fmt.Errorf("could not apply manifest: %w", err)
	}

	summary, err := mgr.SessionSummary()
	if err != nil {
		w.Errorf("Could not get session summary: %v", err)
		return Error, fmt.Errorf("could not get session summary: %w", err)
	}

	switch {
	case summary.TotalResources == summary.StableResources:
		w.Infof("Manifest applied successfully")
		return Stable, nil
	case summary.TotalErrors > 0:
		w.Errorf("Manifest failed to apply with %d errors", summary.TotalErrors)
		return Error, nil
	default:
		w.Infof("Manifest applied with %d changes", summary.ChangedResources)
		return Changes, nil
	}
}

func (w *Watcher) ccmManager(data map[string]any, facts json.RawMessage) (*manager.CCM, ccmmodel.Logger, error) {
	var fdata map[string]any
	err := json.Unmarshal(facts, &fdata)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid facts: %w", err)
	}

	log := NewCCMLogger(w)
	var opts []manager.Option
	if w.properties.Noop {
		opts = append(opts, manager.WithNoop())
	}

	mgr, err := manager.NewManager(log, log, opts...)
	if err != nil {
		return nil, nil, err
	}
	mgr.SetFacts(fdata)
	mgr.SetExternalData(data)

	// try to figure out a sane root for things like source, file() etc in manifests
	wd := w.machine.Directory()
	if w.properties.ManifestFile != "" {
		if filepath.IsAbs(w.properties.ManifestFile) {
			wd = filepath.Dir(w.properties.ManifestFile)
		} else {
			abs, err := filepath.Abs(filepath.Join(w.machine.Directory(), filepath.Dir(w.properties.ManifestFile)))
			if err != nil {
				wd = w.machine.Directory()
			} else {
				wd = abs
			}
		}
	}
	mgr.SetWorkingDirectory(wd)

	return mgr, log, nil
}

func (w *Watcher) CurrentState() any {
	w.mu.Lock()
	defer w.mu.Unlock()

	s := &StateNotification{
		Event:           event.New(w.name, wtype, version, w.machine),
		PreviousOutcome: stateNames[w.previous],
		PreviousRunTime: w.previousRunTime.Nanoseconds(),
	}

	return s
}

func (w *Watcher) handleCheck(s State, err error) error {
	w.mu.Lock()
	w.previous = s
	w.mu.Unlock()

	switch s {
	case Error:
		if err != nil {
			w.Errorf("Check failed: %s", err)
		}

		w.NotifyWatcherState(w.CurrentState())
		return w.FailureTransition()

	case Changes:
		// in noop a change means something should be done, so we treat as failure
		if w.properties.Noop {
			w.Infof("Sending failure transition")
			w.NotifyWatcherState(w.CurrentState())
			return w.FailureTransition()
		}

		w.NotifyWatcherState(w.CurrentState())
		return w.SuccessTransition()

	case Stable:
		w.NotifyWatcherState(w.CurrentState())
		return w.SuccessTransition()
	}

	return nil
}
