package tally

import (
	"context"
	"encoding/json"
	"fmt"
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

// Recorder listens for alive events and records the versions
// and expose the results to Prometheus
type Recorder struct {
	sync.Mutex

	options  *options
	observed map[string]*observation

	// lifecycle
	okEvents     *prometheus.CounterVec
	badEvents    *prometheus.CounterVec
	eventsTally  *prometheus.GaugeVec
	maintTime    *prometheus.SummaryVec
	processTime  *prometheus.SummaryVec
	eventTypes   *prometheus.CounterVec
	nodesExpired *prometheus.CounterVec

	// transitions
	transitionEvent *prometheus.CounterVec
}

type observation struct {
	ts      time.Time
	version string
}

// New creates a new Recorder
func New(opts ...Option) (recorder *Recorder, err error) {
	recorder = &Recorder{
		options:  &options{},
		observed: make(map[string]*observation),
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

	obs, ok := r.observed[hname]
	if !ok {
		r.observed[hname] = &observation{
			ts:      time.Now(),
			version: alive.Version,
		}

		r.eventsTally.WithLabelValues(alive.Component(), alive.Version).Inc()

		return nil
	}

	r.observed[hname].ts = time.Now()

	if obs.version != alive.Version {
		r.eventsTally.WithLabelValues(alive.Component(), obs.version).Dec()
		obs.version = alive.Version
		r.eventsTally.WithLabelValues(alive.Component(), obs.version).Inc()
	}

	return nil
}

func (r *Recorder) processStartup(e lifecycle.Event) error {
	startup := e.(*lifecycle.StartupEvent)

	r.Lock()
	defer r.Unlock()

	hname := startup.Identity()
	obs, ok := r.observed[hname]
	if ok {
		r.eventsTally.WithLabelValues(startup.Component(), obs.version).Dec()
	}

	r.observed[hname] = &observation{
		ts:      time.Now(),
		version: startup.Version,
	}

	r.eventsTally.WithLabelValues(startup.Component(), startup.Version).Inc()

	return nil
}

func (r *Recorder) processShutdown(e lifecycle.Event) error {
	shutdown := e.(*lifecycle.ShutdownEvent)

	r.Lock()
	defer r.Unlock()

	hname := shutdown.Identity()
	obs, ok := r.observed[hname]
	if ok {
		r.eventsTally.WithLabelValues(shutdown.Component(), obs.version).Dec()
		delete(r.observed, hname)
	}

	return nil
}

func (r *Recorder) process(e lifecycle.Event) (err error) {
	r.options.Log.Debugf("Processing %s event from %s %s", e.TypeString(), e.Component(), e.Identity())

	timer := r.processTime.WithLabelValues(r.options.Component)
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
	}

	if err == nil {
		r.okEvents.WithLabelValues(r.options.Component).Inc()
	} else {
		r.badEvents.WithLabelValues(r.options.Component).Inc()
	}

	return err
}

func (r *Recorder) maintenance() {
	if r.options.Component == "" {
		return
	}

	r.Lock()
	defer r.Unlock()

	timer := r.maintTime.WithLabelValues(r.options.Component)
	obs := prometheus.NewTimer(timer)
	defer obs.ObserveDuration()

	oldest := time.Now().Add(-80 * time.Minute)
	older := []string{}

	for host, obs := range r.observed {
		if obs.ts.Before(oldest) {
			r.eventsTally.WithLabelValues(r.options.Component, obs.version).Dec()
			older = append(older, host)
		}
	}

	for _, host := range older {
		r.options.Log.Debugf("Removing node %s, last seen %v", host, r.observed[host].ts)

		delete(r.observed, host)
		r.nodesExpired.WithLabelValues(r.options.Component).Inc()
	}

	if len(older) > 0 {
		r.options.Log.Infof("Removed %d hosts that have not been seen in over an hour", len(older))
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

// Run starts listening for events and record statistics about it in prometheus
func (r *Recorder) Run(ctx context.Context) (err error) {
	lifeEvents := make(chan inter.ConnectorMessage, 100)
	machineTransitions := make(chan inter.ConnectorMessage, 100)

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

	for {
		select {
		case e := <-lifeEvents:
			event, err := lifecycle.NewFromJSON(e.Data())
			if err != nil {
				r.options.Log.Errorf("could not process event: %s", err)
				r.badEvents.WithLabelValues(r.options.Component).Inc()
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
				r.badEvents.WithLabelValues(r.options.Component).Inc()
			}

		case <-maintSched.C:
			r.maintenance()

		case <-ctx.Done():
			return nil
		}
	}
}
