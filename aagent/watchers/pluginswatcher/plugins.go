// Copyright (c) 2021-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package machines

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/choria-io/go-choria/aagent/model"
	"github.com/choria-io/go-choria/aagent/util"
	"github.com/choria-io/go-choria/aagent/watchers/event"
	"github.com/choria-io/go-choria/aagent/watchers/watcher"
	iu "github.com/choria-io/go-choria/internal/util"
	"github.com/mitchellh/mapstructure"
)

type State int

var (
	// PublicKey allows a public key to be compiled in to the binary during CI while using a standard
	// compiled in machine.yaml, effectively this is equivalent to setting the public_key property
	PublicKey = ""
)

const (
	Unknown State = iota
	Skipped
	Error
	Updated
	Unchanged

	wtype   = "plugins"
	version = "v1"
)

var stateNames = map[State]string{
	Unknown:   "unknown",
	Skipped:   "skipped",
	Error:     "error",
	Updated:   "updated",
	Unchanged: "unchanged",
}

type Specification struct {
	Plugins   []byte `json:"plugins"`
	Signature string `json:"signature,omitempty"`
}

type ManagedPlugin struct {
	Name                     string `json:"name" yaml:"name"`
	NamePrefix               string `json:"-" yaml:"-"`
	Source                   string `json:"source" yaml:"source"`
	Username                 string `json:"username" yaml:"username"`
	Password                 string `json:"password" yaml:"password"`
	ContentChecksumsChecksum string `json:"verify_checksum" yaml:"verify_checksum" mapstructure:"verify_checksum"`
	ArchiveChecksum          string `json:"checksum" yaml:"checksum" mapstructure:"checksum"`
	Matcher                  string `json:"match" yaml:"match" mapstructure:"match"`
	Governor                 string `json:"governor" yaml:"governor" mapstructure:"governor"`

	Interval string `json:"-"`
	Target   string `json:"-"`
}

type Properties struct {
	// DataItem is the data item key to get ManagedPlugin from, typically sourced from Key-Value store
	DataItem string `mapstructure:"data_item"`
	// PurgeUnknown will remove plugins not declared in DataItem
	PurgeUnknown bool `mapstructure:"purge_unknown"`
	// MachineManageInterval is the interval that created management machines will use to manage their archives
	MachineManageInterval time.Duration
	// PublicKey is the optional ed25519 public key used to sign the specification, when set
	// the specification received will be validated and any invalid specification will be discarded
	PublicKey string `mapstructure:"public_key"`
	// Directory sets the directory where plugins are being deployed into, when empty defaults to plugins directory like /etc/choria/machines
	Directory string `mapstructure:"plugins_directory"`
	// ManagerMachinePrefix the prefix used in constructing names for the management machines
	ManagerMachinePrefix string `mapstructure:"manager_machine_prefix"`
}

type Watcher struct {
	*watcher.Watcher

	name            string
	machine         model.Machine
	previous        State
	interval        time.Duration
	previousRunTime time.Duration
	previousManaged []*ManagedPlugin
	properties      *Properties

	lastWatch time.Time

	wmu *sync.Mutex
	mu  *sync.Mutex
}

func New(machine model.Machine, name string, states []string, failEvent string, successEvent string, interval string, ai time.Duration, rawprop map[string]any) (any, error) {
	var err error

	plugins := &Watcher{
		name:       name,
		machine:    machine,
		properties: &Properties{},
		lastWatch:  time.Time{},
		wmu:        &sync.Mutex{},
		mu:         &sync.Mutex{},
	}

	plugins.Watcher, err = watcher.NewWatcher(name, wtype, ai, states, machine, failEvent, successEvent)
	if err != nil {
		return nil, err
	}

	err = plugins.setProperties(rawprop)
	if err != nil {
		return nil, fmt.Errorf("could not set properties: %v", err)
	}

	if interval != "" {
		plugins.interval, err = iu.ParseDuration(interval)
		if err != nil {
			return nil, fmt.Errorf("invalid interval: %v", err)
		}

		if plugins.interval < 2*time.Second {
			return nil, fmt.Errorf("interval %v is too small", plugins.interval)
		}
	}

	// Loads the public key from plugin.choria.machine.signing_key when set, overriding the value set here
	if pk := machine.SignerKey(); pk != "" {
		plugins.properties.PublicKey = pk
	}

	return plugins, nil
}

func (w *Watcher) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	w.Infof("plugins watcher %s starting", w.name)

	if w.interval != 0 {
		wg.Add(1)
		go w.intervalWatcher(ctx, wg)
	}

	w.performWatch(ctx, false)

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

func (w *Watcher) watch(ctx context.Context) (state State, err error) {
	if !w.ShouldWatch() {
		return Skipped, nil
	}

	start := time.Now()
	defer func() {
		w.mu.Lock()
		w.previousRunTime = time.Since(start)
		w.mu.Unlock()
	}()

	desired, err := w.desiredState()
	if err != nil {
		return Error, err
	}

	w.mu.Lock()
	w.previousManaged = desired
	w.mu.Unlock()

	purged := false
	updated := false

	if w.properties.PurgeUnknown {
		purged, err = w.purgeUnknownPlugins(ctx, desired)
		if err != nil {
			return Error, err
		}
	}

	for _, m := range desired {
		if m == nil || m.Name == "" {
			continue
		}

		match, err := w.isNodeMatch(m)
		if err != nil {
			w.Debugf("Could not match machine %s to node: %s", m.Name, err)
			continue
		}
		if !match {
			continue
		}

		targetDir := w.targetDirForManagerMachine(m.Name)
		target := filepath.Join(targetDir, "machine.yaml")
		spec, err := w.renderMachine(m)
		if err != nil {
			w.Errorf("Failed to render machine %s: %v", m.Name, err)
			continue
		}

		if iu.FileExist(target) {
			specHash, err := iu.Sha256HashBytes(spec)
			if err != nil {
				w.Errorf("Could not determine hash for spec for %s: %s", m.Name, err)
				continue
			}

			ok, _, err := iu.FileHasSha256Sum(target, specHash)
			if err != nil {
				w.Errorf("Could not compare spec with target %s: %s", target, err)
				continue
			}

			if ok {
				w.Debugf("Machine in %s has the correct content, continuing", target)
				continue
			}

			w.Warnf("Machine in %s has incorrect content, updating", target)

			err = os.RemoveAll(targetDir)
			if err != nil {
				w.Errorf("Could not remove unmatched machine in %s: %s", targetDir, err)
				return Error, err
			}
		}

		w.Warnf("Deploying Choria Autonomous Agent %s from %s", m.Name, m.Source)

		err = os.MkdirAll(targetDir, 0700)
		if err != nil {
			w.Errorf("Could not create directory for %s: %s", m.Name, err)
			continue
		}

		err = os.WriteFile(target, spec, 0600)
		if err != nil {
			w.Errorf("Could not write machine spec for %s: %s", m.Name, err)
			os.RemoveAll(targetDir)
			continue
		}

		updated = true
	}

	if purged || updated {
		return Updated, nil
	}

	return Unchanged, nil
}

func (w *Watcher) handleCheck(s State, err error) error {
	w.Debugf("handling state for %s %v", stateNames[s], err)

	w.mu.Lock()
	w.previous = s
	w.mu.Unlock()

	switch s {
	case Error:
		if err != nil {
			w.Errorf("Managing plugins failed: %s", err)
		}

		w.NotifyWatcherState(w.CurrentState())
		return w.FailureTransition()

	case Updated:
		w.NotifyWatcherState(w.CurrentState())
		return w.SuccessTransition()

	}

	return nil
}

func (w *Watcher) renderMachine(m *ManagedPlugin) ([]byte, error) {
	buf := bytes.NewBuffer([]byte{})
	t := template.New("machine")

	p, err := t.Parse(string(mdat))
	if err != nil {
		return nil, err
	}

	err = p.Execute(buf, m)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (w *Watcher) targetDirForManagedPlugins() string {
	if w.properties.Directory != "" {
		return w.properties.Directory
	}

	return filepath.Dir(w.machine.Directory())
}

func (w *Watcher) targetDirForManagerMachine(m string) string {
	return filepath.Join(filepath.Dir(w.machine.Directory()), fmt.Sprintf("%s_%s", w.properties.ManagerMachinePrefix, m))
}

func (w *Watcher) targetDirForManagedPlugin(m string) string {
	return filepath.Join(w.targetDirForManagedPlugins(), m)
}

func (w *Watcher) purgeUnknownPlugins(ctx context.Context, desired []*ManagedPlugin) (bool, error) {
	current, err := w.currentPlugins()
	if err != nil {
		return false, err
	}

	w.Debugf("Purging unknown plugins from current list %v", current)

	purged := false
	for _, m := range current {
		keep := false
		for _, d := range desired {
			if d == nil || d.Name == "" {
				continue
			}

			if m == d.Name {
				if ok, _ := w.isNodeMatch(d); ok {
					keep = true
					break
				}
			}
		}

		if !keep {
			w.Warnf("Removing existing managed machine %s that is not in new desired set", m)
			target := w.targetDirForManagerMachine(m)
			err = os.RemoveAll(target)
			if err != nil {
				w.Errorf("Could not remove %s: %s", target, err)
				continue
			}

			w.Debugf("Sleeping for 2 seconds to allow manager to exit")
			iu.InterruptibleSleep(ctx, 2*time.Second)

			target = w.targetDirForManagedPlugin(m)
			err = os.RemoveAll(target)
			if err != nil {
				w.Errorf("Could not remove %s: %s", target, err)
				continue
			}

			purged = true
		}
	}

	return purged, nil
}

func (w *Watcher) currentPlugins() ([]string, error) {
	dirs, err := os.ReadDir(w.targetDirForManagedPlugins())
	if err != nil {
		return nil, err
	}

	var found []string

	for _, e := range dirs {
		if !e.IsDir() {
			continue
		}

		parts := strings.SplitN(e.Name(), "_", 2)
		if len(parts) != 2 {
			continue
		}

		if parts[0] == w.properties.ManagerMachinePrefix {
			found = append(found, parts[1])
		}
	}

	return found, nil
}

func (w *Watcher) loadAndValidateData() ([]byte, error) {
	dat, ok := w.machine.DataGet(w.properties.DataItem)
	if !ok {
		return nil, fmt.Errorf("data item %s not present", w.properties.DataItem)
	}

	spec := &Specification{}
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook:       mapstructure.ComposeDecodeHookFunc(mapstructure.StringToTimeDurationHookFunc()),
		Result:           &spec,
		WeaklyTypedInput: true,
	})
	if err != nil {
		return nil, err
	}

	err = decoder.Decode(dat)
	if err != nil {
		return nil, err
	}

	payload, err := base64.StdEncoding.DecodeString(string(spec.Plugins))
	if err != nil {
		w.Errorf("Invalid base64 encoded plugins specification, removing data: %s", err)
		w.machine.DataDelete(w.properties.DataItem)
		return nil, fmt.Errorf("invalid data_item")
	}

	if w.properties.PublicKey != "" {
		if len(spec.Signature) == 0 {
			w.Errorf("No signature found in specification, removing data")
			w.machine.DataDelete(w.properties.DataItem)
			return nil, fmt.Errorf("invalid data_item")
		}

		pk, err := hex.DecodeString(w.properties.PublicKey)
		if err != nil {
			w.Errorf("invalid public key: %s", err)
			return nil, fmt.Errorf("invalid data_item")
		}

		sig, err := hex.DecodeString(spec.Signature)
		if err != nil {
			w.Errorf("invalid signature string, removing data %s: %s", w.properties.DataItem, err)
			w.machine.DataDelete(w.properties.DataItem)
			return nil, fmt.Errorf("invalid data_item")
		}

		if !ed25519.Verify(pk, payload, sig) {
			w.Errorf("Signature in data_item %s did not verify using configured public key '%s', removing data", w.properties.DataItem, w.properties.PublicKey)
			w.machine.DataDelete(w.properties.DataItem)
			return nil, fmt.Errorf("invalid data_item")
		}
	}

	return payload, nil
}

func (w *Watcher) desiredState() ([]*ManagedPlugin, error) {
	data, err := w.loadAndValidateData()
	if err != nil {
		return nil, err
	}

	var desired []*ManagedPlugin

	err = json.Unmarshal(data, &desired)
	if err != nil {
		return nil, fmt.Errorf("invalid plugins specification: %s", err)
	}

	for _, m := range desired {
		m.NamePrefix = w.properties.ManagerMachinePrefix
		m.Interval = w.properties.MachineManageInterval.String()
		m.Target = w.targetDirForManagedPlugins()

		if m.Name == "" {
			return nil, fmt.Errorf("name is required")
		}

		if m.Source == "" {
			return nil, fmt.Errorf("source is required for %s", m.Name)
		}

		if m.ArchiveChecksum == "" {
			return nil, fmt.Errorf("checksum is required for %s", m.Name)
		}

		if m.Target == "" {
			return nil, fmt.Errorf("could not determine target for managed plugin for %s", m.Name)
		}

		if m.ContentChecksumsChecksum == "" {
			return nil, fmt.Errorf("verify_checksum is required for %s", m.Name)
		}
	}

	return desired, nil
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

func (w *Watcher) intervalWatcher(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	tick := time.NewTicker(w.interval)

	for {
		select {
		case <-tick.C:
			w.performWatch(ctx, false)

		case <-ctx.Done():
			tick.Stop()
			return
		}
	}
}

func (w *Watcher) setProperties(props map[string]any) error {
	if w.properties == nil {
		w.properties = &Properties{}
	}

	err := util.ParseMapStructure(props, w.properties)
	if err != nil {
		return err
	}

	if PublicKey != "" {
		w.properties.PublicKey = PublicKey
	}

	if w.properties.ManagerMachinePrefix == "" {
		w.properties.ManagerMachinePrefix = "mm"
	}

	return w.validate()
}

func (w *Watcher) validate() error {
	if w.properties.DataItem == "" {
		return fmt.Errorf("data_item is required")
	}
	if w.machine.Directory() == "" && w.properties.Directory == "" {
		return fmt.Errorf("machine store is not configured")
	}

	if strings.Contains(w.properties.ManagerMachinePrefix, "_") {
		return fmt.Errorf("manager_machine_prefix may not contain underscore")
	}

	if w.properties.MachineManageInterval == 0 {
		w.properties.MachineManageInterval = 2 * time.Minute
	}

	return nil
}

func (w *Watcher) CurrentState() any {
	w.mu.Lock()
	defer w.mu.Unlock()

	s := &StateNotification{
		Event:                  event.New(w.name, wtype, version, w.machine),
		PreviousManagedPlugins: []string{},
		PreviousOutcome:        stateNames[w.previous],
		PreviousRunTime:        w.previousRunTime.Nanoseconds(),
	}

	for _, m := range w.previousManaged {
		s.PreviousManagedPlugins = append(s.PreviousManagedPlugins, m.Name)
	}

	return s
}
