// Copyright (c) 2019-2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package aagent

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	aahttp "github.com/choria-io/go-choria/aagent/http"
	"github.com/choria-io/go-choria/aagent/machine"
	"github.com/choria-io/go-choria/aagent/model"
	notifier "github.com/choria-io/go-choria/aagent/notifiers/choria"
	"github.com/choria-io/go-choria/aagent/watchers"
	"github.com/choria-io/go-choria/internal/util"
	"github.com/sirupsen/logrus"
)

type AAgent struct {
	fw          model.ChoriaProvider
	logger      *logrus.Entry
	machines    []*managedMachine
	notifier    *notifier.Notifier
	httpManager model.HttpManager

	source string

	sync.Mutex
}

type managedMachine struct {
	path       string
	loaded     time.Time
	machine    *machine.Machine
	loadedHash string
	plugin     bool
}

const (
	HTTPSwitchHandlerPattern = "/choria/machine/switch/v1/{machine}/{watcher}"
	HTTPMetricHandlerPattern = "/choria/machine/metric/v1/{machine}/{watcher}"
)

// New creates a new instance of the choria autonomous agent host
func New(dir string, fw model.ChoriaProvider) (aa *AAgent, err error) {
	n, err := notifier.New(fw)
	if err != nil {
		return nil, fmt.Errorf("could not create notifier: %s", err)
	}

	return &AAgent{
		fw:       fw,
		logger:   fw.Logger("aagent"),
		source:   dir,
		machines: []*managedMachine{},
		notifier: n,
	}, nil
}

func (a *AAgent) startHTTPListeners(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	httpPort := a.fw.MachineHTTPPort()

	if httpPort == 0 {
		return
	}

	a.logger.Infof("Starting Autonomous Agent HTTP listeners on port %d", httpPort)

	var err error
	a.httpManager, err = aahttp.NewHTTPServer(a.logger.WithField("port", httpPort))
	if err != nil {
		a.logger.Errorf("Could not start Autonomous Agent HTTP listeners: %s", err)
	}

	mux := http.NewServeMux()
	mux.Handle(HTTPSwitchHandlerPattern, aahttp.LoggingMiddleware(a.logger, http.HandlerFunc(a.httpManager.SwitchHandler)))
	mux.Handle(HTTPMetricHandlerPattern, aahttp.LoggingMiddleware(a.logger, http.HandlerFunc(a.httpManager.MetricHandler)))

	srv := &http.Server{
		BaseContext: func(_ net.Listener) context.Context { return ctx },
		Handler:     mux,
		Addr:        fmt.Sprintf(":%d", httpPort),
	}
	err = srv.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return
	}

	a.logger.Errorf("Could not start Autonomous Agent HTTP listeners: %s", err)
}

// ManageMachines start observing the source directories starting and stopping machines based on changes on disk
func (a *AAgent) ManageMachines(ctx context.Context, wg *sync.WaitGroup) error {
	wg.Add(1)
	go a.watchSource(ctx, wg)
	go a.startHTTPListeners(ctx, wg)

	return nil
}

// Transition transitions a running machine using a supplied transition event. Success is not guaranteed as the machine might be in a state that does not allow the transition
func (a *AAgent) Transition(name string, version string, path string, id string, transition string) error {
	m := a.findMachine(name, version, path, id)
	if m == nil {
		return fmt.Errorf("could not find machine matching criteria name='%s', version='%s', path='%s', id='%s'", name, version, path, id)
	}

	if !m.machine.Can(transition) {
		return fmt.Errorf("transition %s is not valid while in %v state", transition, m.machine.State())
	}

	err := m.machine.Transition(transition)
	if err != nil {
		return err
	}

	return nil
}

func (a *AAgent) configureMachine(aa *machine.Machine) {
	aa.SetFactSource(a.fw.Facts)
	aa.SetIdentity(a.fw.Identity())
	aa.SetMainCollective(a.fw.MainCollective())
	aa.RegisterNotifier(a.notifier)
	aa.SetTextFileDirectory(a.fw.PrometheusTextFileDir())
	aa.SetOverridesFile(a.fw.ScoutOverridesPath())
	aa.SetConnection(a.fw.Connector())
	aa.SetChoriaStatusFile(a.fw.ServerStatusFile())
	aa.SetSignerKey(a.fw.MachineSignerKey())
	aa.SetExternalMachineNotifier(a.notifyMachinesAfterTransition)
	aa.SetExternalMachineStateQuery(a.machineStateLookup)
}

func (a *AAgent) notifyMachinesAfterTransition(event *machine.TransitionNotification) {
	a.Lock()
	defer a.Unlock()

	for _, m := range a.machines {
		a.logger.Debugf("Notifying machine %s about transition %s#%s", m.machine.MachineName, event.Machine, event.Transition)
		go m.machine.ExternalEventNotify(event)
	}
}

func (a *AAgent) machineStateLookup(name string) (string, error) {
	m := a.findMachine(name, "", "", "")
	if m == nil {
		return "", fmt.Errorf("could not find machine matching name='%s'", name)
	}

	return m.machine.State(), nil
}

func (a *AAgent) loadMachine(ctx context.Context, path string) (err error) {
	aa, err := machine.FromDir(path, watchers.New(ctx))
	if err != nil {
		return err
	}

	sum, err := aa.Hash()
	if err != nil {
		return err
	}

	a.logger.Warnf("Loaded Autonomous Agent %s version %s from %s (%s)", aa.Name(), aa.Version(), path, sum)
	a.configureMachine(aa)

	managed := &managedMachine{
		loaded:     time.Now(),
		path:       path,
		machine:    aa,
		loadedHash: sum,
	}

	a.Lock()
	a.machines = append(a.machines, managed)
	a.Unlock()

	return nil
}

func (a *AAgent) startMachines(ctx context.Context, wg *sync.WaitGroup) error {
	a.Lock()
	machines := make([]*managedMachine, len(a.machines))
	copy(machines, a.machines)
	a.Unlock()

	for _, m := range machines {
		if m.machine.IsStarted() {
			continue
		}

		m.machine.Start(ctx, wg)
	}

	return nil
}

// LoadPlugin allows for runtime loading of plugins into a running autonomous agent subsystem.
func (a *AAgent) LoadPlugin(ctx context.Context, p model.MachineConstructor) error {
	if a == nil {
		return fmt.Errorf("autonomous agent subsystem not initialized")
	}

	a.Lock()
	defer a.Unlock()

	machine, err := machine.FromPlugin(p, watchers.New(ctx), a.logger.WithField("plugin", p.PluginName()))
	if err != nil {
		a.logger.Errorf("Could not load machine plugin from %s: %s", p.PluginName(), err)
		return err
	}

	err = machine.SetDirectory(filepath.Join(a.source, machine.MachineName), a.source)
	if err != nil {
		a.logger.Errorf("Could not set machine directory store: %v", err)
		return err
	}

	managed := &managedMachine{
		loaded:  time.Now(),
		machine: machine,
		path:    filepath.Join(a.source, machine.MachineName),
		plugin:  true,
	}

	if !util.FileIsDir(managed.path) {
		err = os.MkdirAll(managed.path, 0700)
		if err != nil {
			return err
		}
	}

	a.configureMachine(machine)

	a.machines = append(a.machines, managed)

	return nil
}

func (a *AAgent) loadPlugins(ctx context.Context) error {
	mu.Lock()
	compiledPlugins := plugins
	mu.Unlock()

	for _, p := range compiledPlugins {
		err := a.LoadPlugin(ctx, p)
		if err != nil {
			a.logger.Errorf("Could not load machine plugin from %s: %s", p.PluginName(), err)
			continue
		}
	}

	return nil
}

func (a *AAgent) loadFromSource(ctx context.Context) error {
	files, err := os.ReadDir(a.source)
	if err != nil {
		return fmt.Errorf("could not read machine source: %s", err)
	}

	for _, file := range files {
		name := file.Name()
		path := filepath.Join(a.source, name)

		if !file.IsDir() || name == "tmp" || strings.HasPrefix(name, ".") || strings.HasSuffix(name, "-temp") {
			continue
		}

		current := a.findMachine("", "", path, "")

		if current != nil && !current.machine.IsEmbedded() {
			hash, err := current.machine.Hash()
			if err != nil {
				a.logger.Errorf("could not determine hash for %s manifest in %v", current.machine.Name(), err)
			}

			if hash == current.loadedHash {
				continue
			}

			a.logger.Warnf("Loaded machine %s does not match current manifest (%s), stopping", current.machine.Name(), hash)
			current.machine.Stop()
			err = a.deleteByPath(path)
			if err != nil {
				a.logger.Errorf("could not delete machine for %s", path)
			}
			a.logger.Debugf("Sleeping 1 second to allow old machine to exit")
			util.InterruptibleSleep(ctx, time.Second)
		}

		if current != nil && current.machine.IsEmbedded() {
			continue
		}

		a.logger.Infof("Attempting to load Autonomous Agent from %s", path)

		err = a.loadMachine(ctx, path)
		if err != nil {
			a.logger.Errorf("Could not load Autonomous Agent from %s: %v", path, err)
		}
	}

	return nil
}

func (a *AAgent) watchSource(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	// loads what is found on disk
	loadf := func() {
		if a.source == "" {
			return
		}

		if !util.FileIsDir(a.source) {
			a.logger.Debugf("Autonomous Agent source directory %s does not exist, skipping", a.source)
			return
		}

		err := a.loadFromSource(ctx)
		if err != nil {
			a.logger.Errorf("Could not load Autonomous Agents from %s: %s", a.source, err)
		}
	}

	// deletes machines not on disk anymore
	cleanf := func() {
		targets := []string{}

		a.Lock()
		for _, m := range a.machines {
			// these are compiled in, cannot be removed
			if m.plugin {
				continue
			}

			if !util.FileExist(m.path) {
				a.logger.Infof("Machine %s does not exist anymore in %s, terminating", m.machine.Name(), m.path)
				targets = append(targets, m.path)
				m.machine.Delete()
			}
		}
		a.Unlock()

		for _, p := range targets {
			err := a.deleteByPath(p)
			if err != nil {
				a.logger.Errorf("Could not terminate machine previously in %s: %s", p, err)
			}
		}
	}

	startf := func() {
		err := a.startMachines(ctx, wg)
		if err != nil {
			a.logger.Errorf("Could not start Autonomous Agents: %s", err)
		}
	}

	err := a.loadPlugins(ctx)
	if err != nil {
		a.logger.Errorf("Could not load Autonomous Agents plugins: %s", err)
	}

	loadf()
	startf()

	tick := time.NewTicker(10 * time.Second)

	for {
		select {
		case <-tick.C:
			cleanf()
			loadf()
			startf()
		case <-ctx.Done():
			return
		}
	}
}

func (a *AAgent) deleteByPath(path string) error {
	a.Lock()
	defer a.Unlock()

	match := -1

	for i, m := range a.machines {
		if m.path == path {
			match = i
		}
	}

	if match >= 0 {
		// delete without memleaks, apparently, https://github.com/golang/go/wiki/SliceTricks
		a.machines[match] = a.machines[len(a.machines)-1]
		a.machines[len(a.machines)-1] = nil
		a.machines = a.machines[:len(a.machines)-1]

		return nil
	}

	return fmt.Errorf("could not find a machine from %s", path)
}

func (a *AAgent) findMachine(name string, version string, path string, id string) *managedMachine {
	a.Lock()
	defer a.Unlock()

	if name == "" && version == "" && path == "" && id == "" {
		return nil
	}

	for _, m := range a.machines {
		nameMatch := name == ""
		versionMatch := version == ""
		pathMatch := path == ""
		idMatch := id == ""

		if name != "" {
			nameMatch = m.machine.Name() == name
		}

		if path != "" {
			pathMatch = m.path == path
		}

		if version != "" {
			versionMatch = m.machine.Version() == version
		}

		if id != "" {
			idMatch = m.machine.InstanceID() == id
		}

		if nameMatch && versionMatch && pathMatch && idMatch {
			return m
		}

	}

	return nil
}
