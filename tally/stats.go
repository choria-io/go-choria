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
		Name: fmt.Sprintf("%s_bad_events", r.options.StatPrefix),
		Help: "The number of unparsable or wrong type of events received",
	}, []string{"component"})

	r.eventsTally = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: fmt.Sprintf("%s_versions", r.options.StatPrefix),
		Help: "The number of observations for a specific version",
	}, []string{"component", "version"})

	r.maintTime = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name: fmt.Sprintf("%s_maintenance_time", r.options.StatPrefix),
		Help: "The time taken to perform maintenance",
	}, []string{"component"})

	if registerStats {
		prometheus.MustRegister(r.okEvents)
		prometheus.MustRegister(r.badEvents)
		prometheus.MustRegister(r.eventsTally)
		prometheus.MustRegister(r.maintTime)
	}
}
