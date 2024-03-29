// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package srvcache

import (
	"github.com/prometheus/client_golang/prometheus"
)

var srvctr = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "choria_dns_srv_lookups",
	Help: "Number of SRV queries performed",
}, []string{"identity"})

var srvhit = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "choria_dns_srv_cachehits",
	Help: "Number of SRV lookups served from the cache",
}, []string{"identity"})

var srvmiss = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "choria_dns_srv_cachemiss",
	Help: "Number of SRV cache lookup misses",
}, []string{"identity"})

func init() {
	prometheus.MustRegister(srvctr)
	prometheus.MustRegister(srvhit)
	prometheus.MustRegister(srvmiss)
}
