// Copyright (c) 2017-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package election

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	campaignsCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "choria_election_campaigns",
		Help: "The number of campaigns a specific candidate voted in",
	}, []string{"election", "identity", "state"})

	leaderGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "choria_election_leader",
		Help: "Indicates if a specific instance is the current leader",
	}, []string{"election", "identity"})

	campaignIntervalGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "choria_election_interval_seconds",
		Help: "The number of seconds between campaigns",
	}, []string{"election", "identity"})
)

func init() {
	prometheus.MustRegister(campaignsCounter)
	prometheus.MustRegister(leaderGauge)
	prometheus.MustRegister(campaignIntervalGauge)
}
