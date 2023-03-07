// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package tally

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/choria-io/go-choria/aagent/watchers/execwatcher"
	"github.com/choria-io/go-choria/backoff"
	election "github.com/choria-io/go-choria/providers/election/streams"
	"github.com/nats-io/nats.go"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
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
	Nats() *nats.Conn
}

// Recorder listens for alive events and records the versions and expose the results to Prometheus
type Recorder struct {
	sync.Mutex

	options  *options
	active   int32
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

	if recorder.options.Election == "" {
		recorder.active = 1
	}

	recorder.createStats()

	return recorder, nil
}

func (r *Recorder) wonCb() {
	atomic.StoreInt32(&r.active, 1)
	r.options.Log.Infof("Became leader")
}

func (r *Recorder) lostCb() {
	atomic.StoreInt32(&r.active, 0)
	r.options.Log.Infof("Lost leadership")
}

func (r *Recorder) activeLabel() string {
	return strconv.Itoa(int(atomic.LoadInt32(&r.active)))
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

		r.versionsTally.WithLabelValues(alive.Component(), alive.Version, r.activeLabel()).Inc()

		return nil
	}

	cobs.hosts[hname].ts = time.Now()

	if obs.version != alive.Version {
		r.versionsTally.WithLabelValues(alive.Component(), obs.version, r.activeLabel()).Dec()
		obs.version = alive.Version
		r.versionsTally.WithLabelValues(alive.Component(), obs.version, r.activeLabel()).Inc()
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
		r.versionsTally.WithLabelValues(startup.Component(), obs.version, r.activeLabel()).Dec()
	}

	cobs.hosts[hname] = &observation{
		ts:        time.Now(),
		version:   startup.Version,
		component: startup.Component(),
	}

	r.versionsTally.WithLabelValues(startup.Component(), startup.Version, r.activeLabel()).Inc()

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
	r.versionsTally.WithLabelValues(shutdown.Component(), obs.version, r.activeLabel()).Dec()

	return nil
}

func (r *Recorder) processGovernor(e lifecycle.Event) error {
	governor := e.(*lifecycle.GovernorEvent)

	r.governorEvents.WithLabelValues(governor.Component(), governor.Governor, string(governor.EventType), r.activeLabel()).Inc()

	return nil
}

func (r *Recorder) process(e lifecycle.Event) (err error) {
	r.options.Log.Debugf("Processing %s event from %s %s", e.TypeString(), e.Component(), e.Identity())

	timer := r.processTime.WithLabelValues(e.Component(), r.activeLabel())
	obs := prometheus.NewTimer(timer)
	defer obs.ObserveDuration()

	r.eventTypes.WithLabelValues(e.Component(), e.TypeString(), r.activeLabel()).Inc()

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
		r.okEvents.WithLabelValues(e.Component(), r.activeLabel()).Inc()
	} else {
		r.badEvents.WithLabelValues(e.Component(), r.activeLabel()).Inc()
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
				r.versionsTally.WithLabelValues(obs.component, obs.version, r.activeLabel()).Dec()
				older[host] = obs
			}
		}

		for host, obs := range older {
			r.options.Log.Debugf("Removing node %s, last seen %v", host, obs.ts)

			delete(r.observed[component].hosts, host)
			r.nodesExpired.WithLabelValues(obs.component, r.activeLabel()).Inc()
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

	r.transitionEvent.WithLabelValues(event.Machine, event.Version, event.Transition, event.FromState, event.ToState, r.activeLabel()).Inc()

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

	if r.options.Election != "" {
		r.options.Log.Warnf("Starting leader election in campaign %s", r.options.Election)

		name, err := os.Hostname()
		if err != nil {
			return err
		}

		js, err := r.options.Connector.Nats().JetStream()
		if err != nil {
			return err
		}

		kv, err := js.KeyValue("CHORIA_LEADER_ELECTION")
		if err != nil {
			return fmt.Errorf("cannot access KV Bucket CHORIA_LEADER_ELECTION: %v", err)
		}

		e, err := election.NewElection(name, r.options.Election, kv, election.WithBackoff(backoff.FiveSec), election.OnWon(r.wonCb), election.OnLost(r.lostCb))
		if err != nil {
			return err
		}

		go e.Start(ctx)
	}

	for {
		select {
		case e := <-lifeEvents:
			event, err := lifecycle.NewFromJSON(e.Data())
			if err != nil {
				r.options.Log.Errorf("could not process event: %s", err)
				r.badEvents.WithLabelValues(r.componentFromSubject(e.Subject()), r.activeLabel()).Inc()
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
				r.badEvents.WithLabelValues("transition", r.activeLabel()).Inc()
			}

		case t := <-execWatcherStates:
			err = r.processExecWatcherState(t)
			if err != nil {
				r.options.Log.Errorf("could not process exec watcher event: %s", err)
				r.badEvents.WithLabelValues("exec_watcher", r.activeLabel()).Inc()
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
		r.execWatchSuccess.WithLabelValues(event.Machine, event.Version, event.Name, r.activeLabel()).Inc()
	case "error":
		r.execWatchFail.WithLabelValues(event.Machine, event.Version, event.Name, r.activeLabel()).Inc()
	default:
		return nil
	}

	r.execWatchRuntime.WithLabelValues(event.Machine, event.Version, event.Name, r.activeLabel()).Observe(time.Duration(event.PreviousRunTime).Seconds())

	return nil
}
