package server

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	totalCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "choria_server_total",
		Help: "Total number of messages received from the network",
	}, []string{"identity"})

	validatedCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "choria_server_validated",
		Help: "Number of messages that were received and validated succesfully",
	}, []string{"identity"})

	unvalidatedCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "choria_server_unvalidated",
		Help: "Number of messages that were received but did not pass validation",
	}, []string{"identity"})

	passedCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "choria_server_passed",
		Help: "Number of messages where this instance matched the filter expression",
	}, []string{"identity"})

	filteredCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "choria_server_filtered",
		Help: "Number of messages where this instance did not match the filter expression",
	}, []string{"identity"})

	repliesCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "choria_server_replies",
		Help: "Number of reply messages that were produced",
	}, []string{"identity"})

	ttlExpiredCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "choria_server_ttlexpired",
		Help: "Number of messages received that were too old and dropped",
	}, []string{"identity"})
)

func init() {
	prometheus.MustRegister(validatedCtr)
	prometheus.MustRegister(unvalidatedCtr)
	prometheus.MustRegister(passedCtr)
	prometheus.MustRegister(filteredCtr)
	prometheus.MustRegister(repliesCtr)
	prometheus.MustRegister(ttlExpiredCtr)
	prometheus.MustRegister(totalCtr)
}
