package stats

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	ReceivedMsgsCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "choria_federation_received_msgs",
		Help: "Number of messages received by a Federation Broker worker",
	}, []string{"name", "worker", "connected_to", "cluster", "identity"})

	PublishedMsgsCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "choria_federation_published_msgs",
		Help: "Number of messages published by a Federation Broker worker",
	}, []string{"name", "worker", "connected_to", "cluster", "identity"})

	ErrorCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "choria_federation_errors",
		Help: "Messages that could not be handled",
	}, []string{"name", "worker", "connected_to", "cluster", "identity"})

	ProcessTime = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name: "choria_federation_time",
		Help: "Time taken to process messages",
	}, []string{"name", "worker", "connected_to", "cluster", "identity"})
)

func init() {
	prometheus.MustRegister(ReceivedMsgsCtr)
	prometheus.MustRegister(PublishedMsgsCtr)
	prometheus.MustRegister(ErrorCtr)
	prometheus.MustRegister(ProcessTime)
}
