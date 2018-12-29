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

var stubSource chan *choria.ConnectorMessage

// Connector is a connection to the middleware
type Connector interface {
	QueueSubscribe(ctx context.Context, name string, subject string, group string, output chan *choria.ConnectorMessage) error
}

// Recorder listens for alive events and records the versions
// and expose the results to Prometheus
type Recorder struct {
	sync.Mutex

	options  *options
	observed map[uint64]*observation

	okEvents    *prometheus.CounterVec
	badEvents   *prometheus.CounterVec
	eventsTally *prometheus.GaugeVec
	maintTime   *prometheus.SummaryVec
}

type observation struct {
	ts      time.Time
	version string
}

// New creates a new Recorder
func New(opts ...Option) (recorder *Recorder, err error) {
	recorder = &Recorder{
		options:  &options{},
		observed: make(map[uint64]*observation),
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

func (r *Recorder) process(e lifecycle.Event) error {
	if e.Type() != lifecycle.Alive {
		r.badEvents.WithLabelValues(r.options.Component).Inc()

		return fmt.Errorf("can only process Alive events, received %s", e.TypeString())
	}

	alive := e.(*lifecycle.AliveEvent)

	r.okEvents.WithLabelValues(r.options.Component).Inc()

	r.Lock()
	defer r.Unlock()

	hname := hostHash(alive.Identity())

	obs, ok := r.observed[hname]
	if !ok {
		r.observed[hname] = &observation{
			ts:      time.Now(),
			version: alive.Version,
		}

		r.eventsTally.WithLabelValues(alive.Component(), alive.Version).Inc()

		return nil
	}

	if obs.version != alive.Version {
		r.eventsTally.WithLabelValues(r.options.Component, obs.version).Dec()
		obs.version = alive.Version
		r.eventsTally.WithLabelValues(r.options.Component, obs.version).Inc()
	}

	return nil
}

func (r *Recorder) maintenance() {
	r.Lock()
	defer r.Unlock()

	timer := r.maintTime.WithLabelValues(r.options.Component)
	obs := prometheus.NewTimer(timer)
	defer obs.ObserveDuration()

	oldest := time.Now().Add(-1*time.Hour + time.Minute)
	older := []uint64{}

	for host, obs := range r.observed {
		if obs.ts.Before(oldest) {
			r.eventsTally.WithLabelValues(r.options.Component, obs.version).Dec()
			older = append(older, host)
		}
	}

	for _, host := range older {
		delete(r.observed, host)
	}
}

// Run starts listening for events and record statistics about it in prometheus
func (r *Recorder) Run(ctx context.Context) error {
	var events chan *choria.ConnectorMessage

	if stubSource == nil {
		events = stubSource
	} else {
		events = make(chan *choria.ConnectorMessage, 100)
	}

	maintSched := time.NewTicker(time.Minute)
	subid, err := uuid.NewV4()
	if err != nil {
		return errors.Wrap(err, "could not create random subscription id")
	}

	r.options.Connector.QueueSubscribe(ctx, fmt.Sprintf("tally_%s_%s", r.options.Component, subid.String()), fmt.Sprintf("choria.lifecycle.event.alive.%s", r.options.Component), "", events)

	for {
		select {
		case e := <-events:
			event, err := lifecycle.NewFromJSON(e.Data)
			if err != nil {
				r.options.Log.Printf("could not process event: %s", err)
				continue
			}

			r.process(event)

		case <-maintSched.C:
			r.maintenance()

		case <-ctx.Done():
			return nil
		}
	}
}
