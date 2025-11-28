// Copyright (c) 2019-2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package filewatcher

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/choria-io/go-choria/aagent/model"
	"github.com/choria-io/go-choria/aagent/util"
	"github.com/choria-io/go-choria/aagent/watchers/event"
	"github.com/choria-io/go-choria/aagent/watchers/watcher"
	iu "github.com/choria-io/go-choria/internal/util"
)

type State int

const (
	wtype   = "file"
	version = "v1"
)

const (
	Unknown State = iota
	Error
	Skipped
	Unchanged
	Changed
)

var stateNames = map[State]string{
	Unknown:   "unknown",
	Error:     "error",
	Skipped:   "skipped",
	Unchanged: "unchanged",
	Changed:   "changed",
}

type Properties struct {
	// Path is path to the file to watch relative to the watcher manifest directory
	Path string
	// Initial gathers the initial file mode, stats etc for regular announces but only perform first watch after interval
	Initial bool `mapstructure:"gather_initial_state"`
	// Contents place specific content into the file, supports template parsing and data lookup
	Contents string
	// Owner is should own the file when managing content
	Owner string
	// Group is what group should own the file when managing content
	Group string
	// Mode is the file mode to apply when managing content, must be a string like "0700"
	Mode string
}

type Watcher struct {
	*watcher.Watcher

	name       string
	machine    model.Machine
	previous   State
	interval   time.Duration
	mtime      time.Time
	properties *Properties
	mu         *sync.Mutex
}

func New(machine model.Machine, name string, states []string, required []model.ForeignMachineState, failEvent string, successEvent string, interval string, ai time.Duration, properties map[string]any) (any, error) {
	var err error

	fw := &Watcher{
		properties: &Properties{},
		name:       name,
		machine:    machine,
		interval:   5 * time.Second,
		mu:         &sync.Mutex{},
	}

	fw.Watcher, err = watcher.NewWatcher(name, wtype, ai, states, required, machine, failEvent, successEvent)
	if err != nil {
		return nil, err
	}

	err = fw.setProperties(properties)
	if err != nil {
		return nil, fmt.Errorf("could not set properties: %s", err)
	}

	if !filepath.IsAbs(fw.properties.Path) {
		fw.properties.Path = filepath.Join(fw.machine.Directory(), fw.properties.Path)
	}

	if interval != "" {
		fw.interval, err = iu.ParseDuration(interval)
		if err != nil {
			return nil, fmt.Errorf("invalid interval: %s", err)
		}
	}

	if fw.interval < 500*time.Millisecond {
		return nil, fmt.Errorf("interval %v is too small", fw.interval)
	}

	return fw, err
}

func (w *Watcher) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	w.Infof("file watcher for %s starting", w.properties.Path)

	tick := time.NewTicker(w.interval)

	if w.properties.Initial {
		stat, err := os.Stat(w.properties.Path)
		if err == nil {
			w.mtime = stat.ModTime()
		}
	}

	for {
		select {
		case <-tick.C:
			w.performWatch(ctx)

		case <-w.Watcher.StateChangeC():
			w.performWatch(ctx)

		case <-ctx.Done():
			tick.Stop()
			w.Infof("Stopping on context interrupt")
			return
		}
	}
}

func (w *Watcher) performWatch(_ context.Context) {
	state, err := w.watch()
	err = w.handleCheck(state, err)
	if err != nil {
		w.Errorf("could not handle watcher event: %s", err)
	}
}

func (w *Watcher) CurrentState() any {
	w.mu.Lock()
	defer w.mu.Unlock()

	s := &StateNotification{
		Event:           event.New(w.name, wtype, version, w.machine),
		Path:            w.properties.Path,
		PreviousOutcome: stateNames[w.previous],
	}

	return s
}

func (w *Watcher) setPreviousState(s State) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.previous = s
}

func (w *Watcher) handleCheck(s State, err error) error {
	w.Debugf("Handling check for %s %v %v", w.properties.Path, s, err)

	w.setPreviousState(s)

	switch s {
	case Error:
		w.NotifyWatcherState(w.CurrentState())
		return w.FailureTransition()

	case Changed:
		w.NotifyWatcherState(w.CurrentState())
		return w.SuccessTransition()

	case Unchanged:
	// not notifying, regular announces happen

	case Skipped:
	// nothing really to do, we keep old mtime next time round
	// we'll correctly notify of changes

	case Unknown:
		w.mtime = time.Time{}
	}

	return nil
}

func (w *Watcher) watchFileStat() (state State, err error) {
	stat, err := os.Stat(w.properties.Path)
	switch {
	case err == nil && stat.ModTime().After(w.mtime):
		w.mtime = stat.ModTime()
		return Changed, nil

	case os.IsNotExist(err):
		w.mtime = time.Time{}
		return Error, fmt.Errorf("does not exist")

	case err != nil:
		w.mtime = time.Time{}
		return Error, err

	default:
		return Unchanged, err
	}
}

func (w *Watcher) createFile(content []byte, mode os.FileMode) (state State, err error) {
	dir, err := filepath.Abs(filepath.Dir(w.properties.Path))
	if err != nil {
		return Error, err
	}

	tf, err := os.CreateTemp(dir, "choria-fwatcher-*")
	if err != nil {
		return Error, fmt.Errorf("could not create temporary file: %w", err)
	}
	defer tf.Close()
	defer os.Remove(tf.Name())

	err = tf.Chmod(mode)
	if err != nil {
		return Error, fmt.Errorf("could not change file permissions: %w", err)
	}

	_, err = tf.Write(content)
	if err != nil {
		return Error, fmt.Errorf("could not write temporary file: %w", err)
	}

	if runtime.GOOS != "windows" {
		owner, err := w.ProcessTemplate(w.properties.Owner)
		if err != nil {
			return Error, fmt.Errorf("could not process file owner template: %w", err)
		}

		group, err := w.ProcessTemplate(w.properties.Group)
		if err != nil {
			return Error, fmt.Errorf("could not process file owner template: %w", err)
		}

		if owner != "" && group != "" {
			usrIdString, err := user.Lookup(owner)
			if err != nil {
				return Error, fmt.Errorf("could not lookup user %q (%q): %w", owner, w.properties.Owner, err)
			}
			uid, err := strconv.Atoi(usrIdString.Uid)
			if err != nil {
				return Error, fmt.Errorf("could not convert user id %s to integer: %w", usrIdString.Uid, err)
			}

			grpIdString, err := user.LookupGroup(group)
			if err != nil {
				return Error, fmt.Errorf("could not lookup group %s: %w", group, err)
			}
			gid, err := strconv.Atoi(grpIdString.Gid)
			if err != nil {
				return Error, fmt.Errorf("could not convert group id %s to integer: %w", grpIdString.Gid, err)
			}

			err = tf.Chown(uid, gid)
			if err != nil {
				return Error, fmt.Errorf("could not change file ownership: %w", err)
			}
		}
	}

	err = tf.Close()
	if err != nil {
		return Error, fmt.Errorf("could not close temporary file: %w", err)
	}

	err = os.Rename(tf.Name(), w.properties.Path)
	if err != nil {
		return Error, fmt.Errorf("could not rename temporary file: %w", err)
	}

	stat, err := os.Stat(w.properties.Path)
	if err == nil {
		w.mtime = stat.ModTime()
	}

	return Changed, nil
}

func (w *Watcher) watchFileContent() (state State, err error) {
	mode, err := w.ProcessTemplate(w.properties.Mode)
	if err != nil {
		return Error, fmt.Errorf("could not process file mode template: %w", err)
	}
	if mode == "" {
		return Error, fmt.Errorf("mode template result is empty")
	}

	wantMode, err := strconv.ParseUint(mode, 0, 32)
	if err != nil {
		return Error, fmt.Errorf("invalid mode, must be a string like 0700")
	}

	content, err := w.ProcessTemplateBytes(w.properties.Contents)
	if err != nil {
		return Error, fmt.Errorf("could not process template: %s", err)
	}

	stat, err := os.Stat(w.properties.Path)
	if err != nil {
		return w.createFile(content, os.FileMode(wantMode))
	}

	if stat.ModTime().After(w.mtime) {
		return w.createFile(content, os.FileMode(wantMode))
	}

	if stat.Mode() != os.FileMode(wantMode) {
		return w.createFile(content, os.FileMode(wantMode))
	}

	cHash, err := iu.Sha256HashBytes(content)
	if err != nil {
		return Error, fmt.Errorf("could not hash content: %s", err)
	}

	fHash, err := iu.Sha256HashFile(w.properties.Path)
	if err != nil {
		return Error, fmt.Errorf("could not hash file: %s", err)
	}

	if cHash != fHash {
		return w.createFile(content, os.FileMode(wantMode))
	}

	return Unchanged, nil
}

func (w *Watcher) watch() (state State, err error) {
	if !w.Watcher.ShouldWatch() {
		return Skipped, nil
	}

	if w.properties.Contents == "" {
		return w.watchFileStat()
	}

	return w.watchFileContent()
}

func (w *Watcher) validate() error {
	if w.properties.Path == "" {
		return fmt.Errorf("path is required")
	}

	if w.properties.Contents != "" {
		if w.properties.Owner == "" {
			return fmt.Errorf("owner is required when managing content")
		}
		if w.properties.Group == "" {
			return fmt.Errorf("group is required when managing content")
		}
		if w.properties.Mode == "" {
			return fmt.Errorf("mode is required when managing content")
		}

		_, err := strconv.ParseUint(w.properties.Mode, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid mode, must be a string like 0700")
		}
	}

	return nil
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
