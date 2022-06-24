// Copyright (c) 2017-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package stats

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	BadJsonCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "choria_protocol_json_parse_failures",
		Help: "Amount of times un-parsable JSON was received",
	}, []string{"version"})

	ValidCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "choria_protocol_valid",
		Help: "Number of messages with valid signatures",
	}, []string{"version"})

	InvalidCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "choria_protocol_invalid",
		Help: "Number of messages with invalid signatures",
	}, []string{"version"})

	ProtocolErrorCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "choria_protocol_error",
		Help: "Number of protocol errors such as missing certificate, un-parsable files and other unrecoverable errors",
	}, []string{"version"})
)

func init() {
	prometheus.MustRegister(BadJsonCtr)
	prometheus.MustRegister(ValidCtr)
	prometheus.MustRegister(InvalidCtr)
	prometheus.MustRegister(ProtocolErrorCtr)
}
