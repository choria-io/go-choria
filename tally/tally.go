package tally

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/choria-io/go-choria/choria"
	lifecycle "github.com/choria-io/go-lifecycle"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

// Connector is a connection to the middleware
type Connector interface {
	QueueSubscribe(ctx context.Context, name string, subject string, group string, output chan *choria.ConnectorMessage) error
}

// Recorder listens for alive events and records the versions
// and expose the results to Prometheus
type Recorder struct {
	sync.Mutex

	options  *options
	observed map[string]*observation

	okEvents    *prometheus.CounterVec
	badEvents   *prometheus.CounterVec
	eventsTally *prometheus.GaugeVec
	maintTime   *prometheus.SummaryVec
	processTime *prometheus.SummaryVec
	eventTypes  *prometheus.CounterVec
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
		return nil, errors.Wrap(err, "invalid options supplied")
	}

	recorder.createStats()

	return recorder, nil
}

func (r *Recorder) processAlive(e lifecycle.Event) error {
	alive := e.(*lifecycle.AliveEvent)

	r.Lock()
	defer r.Unlock()

	hname := alive.Identity()

	obs, ok := r.observed[alive.Identity()]
	if !ok {
		r.observed[hname] = &observation{
			ts:      time.Now(),
			version: alive.Version,
		}

		r.eventsTally.WithLabelValues(alive.Component(), alive.Version).Inc()

		return nil
	}

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
	}

	if len(older) > 0 {
		r.options.Log.Infof("Removed %d hosts that have not been seen in over an hour", len(older))
	}
}

// Run starts listening for events and record statistics about it in prometheus
func (r *Recorder) Run(ctx context.Context) error {
	events := make(chan *choria.ConnectorMessage, 100)
	maintSched := time.NewTicker(time.Minute)
	subid, err := uuid.NewV4()
	if err != nil {
		return errors.Wrap(err, "could not create random subscription id")
	}

	err = r.options.Connector.QueueSubscribe(ctx, fmt.Sprintf("tally_%s_%s", r.options.Component, subid.String()), fmt.Sprintf("choria.lifecycle.event.*.%s", r.options.Component), "", events)
	if err != nil {
		return errors.Wrap(err, "could not subscribe")
	}

	for {
		select {
		case e := <-events:
			event, err := lifecycle.NewFromJSON(e.Data)
			if err != nil {
				r.options.Log.Errorf("could not process event: %s", err)
				continue
			}

			err = r.process(event)
			if err != nil {
				r.options.Log.Errorf("could not process event from %s: %s", event.Identity(), err)
				continue
			}

		case <-maintSched.C:
			r.maintenance()

		case <-ctx.Done():
			return nil
		}
	}
}
