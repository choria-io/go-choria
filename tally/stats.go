package tally

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

var registerStats = true

func (r *Recorder) createStats() {
	r.okEvents = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: fmt.Sprintf("%s_good_events", r.options.StatPrefix),
		Help: "The number of successfully parsed events received",
	}, []string{"component"})

	r.badEvents = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: fmt.Sprintf("%s_process_errors", r.options.StatPrefix),
		Help: "The number of events received that failed to process",
	}, []string{"component"})

	r.eventTypes = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: fmt.Sprintf("%s_event_types", r.options.StatPrefix),
		Help: "The number events received by type",
	}, []string{"component", "type"})

	r.eventsTally = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: fmt.Sprintf("%s_versions", r.options.StatPrefix),
		Help: "The number of observations for a specific version",
	}, []string{"component", "version"})

	r.maintTime = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name: fmt.Sprintf("%s_maintenance_time", r.options.StatPrefix),
		Help: "The time taken to perform maintenance",
	}, []string{"component"})

	r.processTime = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name: fmt.Sprintf("%s_processing_time", r.options.StatPrefix),
		Help: "The time taken to process events",
	}, []string{"component"})

	r.transitionEvent = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: fmt.Sprintf("%s_machine_transition", r.options.StatPrefix),
		Help: "Machine state transition",
	}, []string{"machine", "version", "transition", "from", "to"})

	if registerStats {
		prometheus.MustRegister(r.okEvents)
		prometheus.MustRegister(r.badEvents)
		prometheus.MustRegister(r.eventTypes)
		prometheus.MustRegister(r.eventsTally)
		prometheus.MustRegister(r.maintTime)
		prometheus.MustRegister(r.processTime)
		prometheus.MustRegister(r.transitionEvent)
	}
}
