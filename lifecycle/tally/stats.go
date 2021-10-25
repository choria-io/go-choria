// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

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

	r.versionsTally = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: fmt.Sprintf("%s_versions", r.options.StatPrefix),
		Help: "The number of observations for a specific version and component",
	}, []string{"component", "version"})

	r.processTime = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name: fmt.Sprintf("%s_processing_time", r.options.StatPrefix),
		Help: "The time taken to process events",
	}, []string{"component"})

	r.transitionEvent = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: fmt.Sprintf("%s_machine_transition", r.options.StatPrefix),
		Help: "Machine state transition",
	}, []string{"machine", "version", "transition", "from", "to"})

	r.nodesExpired = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: fmt.Sprintf("%s_nodes_expired", r.options.StatPrefix),
		Help: "The number of nodes that were expired after not receiving alive events",
	}, []string{"component"})

	r.governorEvents = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: fmt.Sprintf("%s_governor", r.options.StatPrefix),
		Help: "Choria Governor events",
	}, []string{"component", "governor", "event"})

	if registerStats {
		prometheus.MustRegister(r.okEvents)
		prometheus.MustRegister(r.badEvents)
		prometheus.MustRegister(r.eventTypes)
		prometheus.MustRegister(r.versionsTally)
		prometheus.MustRegister(r.processTime)
		prometheus.MustRegister(r.transitionEvent)
		prometheus.MustRegister(r.nodesExpired)
		prometheus.MustRegister(r.governorEvents)
	}
}
