// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package tally

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/choria-io/go-choria/aagent/watchers/execwatcher"
	"strings"
	"sync"
	"time"

	"github.com/choria-io/go-choria/aagent/machine"
	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/internal/util"
	"github.com/choria-io/go-choria/lifecycle"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/prometheus/client_golang/prometheus"
)

// Connector is a connection to the middleware
type Connector interface {
	QueueSubscribe(ctx context.Context, name string, subject string, group string, output chan inter.ConnectorMessage) error
}

// Recorder listens for alive events and records the versions and expose the results to Prometheus
type Recorder struct {
	sync.Mutex

	options  *options
	observed map[string]*observations

	// lifecycle
	okEvents       *prometheus.CounterVec
	badEvents      *prometheus.CounterVec
	versionsTally  *prometheus.GaugeVec
	processTime    *prometheus.SummaryVec
	eventTypes     *prometheus.CounterVec
	nodesExpired   *prometheus.CounterVec
	governorEvents *prometheus.CounterVec

	// transitions
	transitionEvent *prometheus.CounterVec

	// exec watch events
	execWatchSuccess *prometheus.CounterVec
	execWatchFail    *prometheus.CounterVec
	execWatchRuntime *prometheus.SummaryVec
}

type observations struct {
	component string
	hosts     map[string]*observation
}

type observation struct {
	ts        time.Time
	component string
	version   string
}

// New creates a new Recorder
func New(opts ...Option) (recorder *Recorder, err error) {
	recorder = &Recorder{
		options:  &options{},
		observed: make(map[string]*observations),
	}

	for _, opt := range opts {
		opt(recorder.options)
	}

	err = recorder.options.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid options supplied: %s", err)
	}

	recorder.createStats()

	return recorder, nil
}

func (r *Recorder) processAlive(e lifecycle.Event) error {
	alive := e.(*lifecycle.AliveEvent)

	r.Lock()
	defer r.Unlock()

	hname := alive.Identity()

	cobs, ok := r.observed[alive.Component()]
	if !ok {
		cobs = &observations{
			component: alive.Component(),
			hosts:     make(map[string]*observation),
		}
		r.observed[alive.Component()] = cobs
	}

	obs, ok := cobs.hosts[hname]
	if !ok {
		cobs.hosts[hname] = &observation{
			version:   alive.Version,
			component: alive.Component(),
			ts:        time.Now(),
		}

		r.versionsTally.WithLabelValues(alive.Component(), alive.Version).Inc()

		return nil
	}

	cobs.hosts[hname].ts = time.Now()

	if obs.version != alive.Version {
		r.versionsTally.WithLabelValues(alive.Component(), obs.version).Dec()
		obs.version = alive.Version
		r.versionsTally.WithLabelValues(alive.Component(), obs.version).Inc()
	}

	return nil
}

func (r *Recorder) processStartup(e lifecycle.Event) error {
	startup := e.(*lifecycle.StartupEvent)

	r.Lock()
	defer r.Unlock()

	hname := startup.Identity()
	cobs, ok := r.observed[startup.Component()]
	if !ok {
		cobs = &observations{
			component: startup.Component(),
			hosts:     make(map[string]*observation),
		}
		r.observed[startup.Component()] = cobs
	}

	obs, ok := cobs.hosts[hname]
	if ok {
		r.versionsTally.WithLabelValues(startup.Component(), obs.version).Dec()
	}

	cobs.hosts[hname] = &observation{
		ts:        time.Now(),
		version:   startup.Version,
		component: startup.Component(),
	}

	r.versionsTally.WithLabelValues(startup.Component(), startup.Version).Inc()

	return nil
}

func (r *Recorder) processShutdown(e lifecycle.Event) error {
	shutdown := e.(*lifecycle.ShutdownEvent)

	r.Lock()
	defer r.Unlock()

	cobs, ok := r.observed[shutdown.Component()]
	if !ok {
		return nil
	}

	hname := shutdown.Identity()
	obs, ok := cobs.hosts[hname]
	if !ok {
		return nil
	}

	delete(cobs.hosts, hname)
	r.versionsTally.WithLabelValues(shutdown.Component(), obs.version).Dec()

	return nil
}

func (r *Recorder) processGovernor(e lifecycle.Event) error {
	governor := e.(*lifecycle.GovernorEvent)

	r.governorEvents.WithLabelValues(governor.Component(), governor.Governor, string(governor.EventType)).Inc()

	return nil
}

func (r *Recorder) process(e lifecycle.Event) (err error) {
	r.options.Log.Debugf("Processing %s event from %s %s", e.TypeString(), e.Component(), e.Identity())

	timer := r.processTime.WithLabelValues(e.Component())
	obs := prometheus.NewTimer(timer)
	defer obs.ObserveDuration()

	r.eventTypes.WithLabelValues(e.Component(), e.TypeString()).Inc()

	switch e.Type() {
	case lifecycle.Alive:
		err = r.processAlive(e)

	case lifecycle.Startup:
		err = r.processStartup(e)

	case lifecycle.Shutdown:
		err = r.processShutdown(e)

	case lifecycle.Governor:
		err = r.processGovernor(e)
	}

	if err == nil {
		r.okEvents.WithLabelValues(e.Component()).Inc()
	} else {
		r.badEvents.WithLabelValues(e.Component()).Inc()
	}

	return err
}

func (r *Recorder) maintenance() {
	r.Lock()
	defer r.Unlock()

	if len(r.observed) == 0 {
		return
	}

	oldest := time.Now().Add(-80 * time.Minute)

	for component := range r.observed {
		older := map[string]*observation{}

		for host, obs := range r.observed[component].hosts {
			if obs.ts.Before(oldest) {
				r.versionsTally.WithLabelValues(obs.component, obs.version).Dec()
				older[host] = obs
			}
		}

		for host, obs := range older {
			r.options.Log.Debugf("Removing node %s, last seen %v", host, obs.ts)

			delete(r.observed[component].hosts, host)
			r.nodesExpired.WithLabelValues(obs.component).Inc()
		}

		if len(older) > 0 {
			r.options.Log.Infof("Removed %d '%s' hosts that have not been seen in over an hour", len(older), component)
		}
	}
}

func (r *Recorder) processStateTransition(m inter.ConnectorMessage) (err error) {
	ce := cloudevents.NewEvent("1.0")
	event := &machine.TransitionNotification{}

	err = json.Unmarshal(m.Data(), &ce)
	if err != nil {
		return fmt.Errorf("could not parse cloudevent: %s", err)
	}

	err = ce.DataAs(event)
	if err != nil {
		return fmt.Errorf("could not parse transition event: %s", err)
	}

	if event.Protocol != "io.choria.machine.v1.transition" {
		return fmt.Errorf("unknown notification protocol %s", event.Protocol)
	}

	r.transitionEvent.WithLabelValues(event.Machine, event.Version, event.Transition, event.FromState, event.ToState).Inc()

	return nil
}

func (r *Recorder) componentFromSubject(s string) string {
	parts := strings.Split(s, ".")
	if len(parts) == 0 {
		return "unknown"
	}

	return parts[len(parts)-1]
}

// Run starts listening for events and record statistics about it in prometheus
func (r *Recorder) Run(ctx context.Context) (err error) {
	lifeEvents := make(chan inter.ConnectorMessage, 100)
	machineTransitions := make(chan inter.ConnectorMessage, 100)
	execWatcherStates := make(chan inter.ConnectorMessage, 100)

	maintSched := time.NewTicker(time.Minute)
	subid := util.UniqueID()

	if r.options.Component == "" {
		r.options.Log.Warn("Component was not specified, disabling lifecycle tallies")
	} else {
		err = r.options.Connector.QueueSubscribe(ctx, fmt.Sprintf("tally_%s_%s", r.options.Component, subid), fmt.Sprintf("choria.lifecycle.event.*.%s", r.options.Component), "", lifeEvents)
		if err != nil {
			return fmt.Errorf("could not subscribe to lifecycle events: %s", err)
		}
	}

	err = r.options.Connector.QueueSubscribe(ctx, fmt.Sprintf("tally_transitions_%s", subid), "choria.machine.transition", "", machineTransitions)
	if err != nil {
		return fmt.Errorf("could not subscribe to machine transition events: %s", err)
	}

	err = r.options.Connector.QueueSubscribe(ctx, "tally_exec_watcher_states", "choria.machine.watcher.exec.state", "", execWatcherStates)
	if err != nil {
		return err
	}

	for {
		select {
		case e := <-lifeEvents:
			event, err := lifecycle.NewFromJSON(e.Data())
			if err != nil {
				r.options.Log.Errorf("could not process event: %s", err)
				r.badEvents.WithLabelValues(r.componentFromSubject(e.Subject())).Inc()
				continue
			}

			err = r.process(event)
			if err != nil {
				r.options.Log.Errorf("could not process event from %s: %s", event.Identity(), err)
			}

		case t := <-machineTransitions:
			err = r.processStateTransition(t)
			if err != nil {
				r.options.Log.Errorf("could not process transition event: %s", err)
				r.badEvents.WithLabelValues("transition").Inc()
			}

		case t := <-execWatcherStates:
			err = r.processExecWatcherState(t)
			if err != nil {
				r.options.Log.Errorf("could not process exec watcher event: %s", err)
				r.badEvents.WithLabelValues("exec_watcher").Inc()
			}

		case <-maintSched.C:
			r.maintenance()

		case <-ctx.Done():
			return nil
		}
	}
}

func (r *Recorder) processExecWatcherState(m inter.ConnectorMessage) error {
	ce := cloudevents.NewEvent("1.0")
	event := &execwatcher.StateNotification{}

	err := json.Unmarshal(m.Data(), &ce)
	if err != nil {
		return fmt.Errorf("could not parse cloudevent: %w", err)
	}

	err = ce.DataAs(event)
	if err != nil {
		return fmt.Errorf("could not parse state notification: %w", err)
	}

	switch event.PreviousOutcome {
	case "success":
		r.execWatchSuccess.WithLabelValues(event.Machine, event.Version, event.Name).Inc()
	case "error":
		r.execWatchFail.WithLabelValues(event.Machine, event.Version, event.Name).Inc()
	default:
		return nil
	}

	r.execWatchRuntime.WithLabelValues(event.Machine, event.Version, event.Name).Observe(time.Duration(event.PreviousRunTime).Seconds())

	return nil
}
