package stats

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	ReceivedMsgsCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "choria_adapter_received_msgs",
		Help: "Number of messages received by adapters on their inputs",
	}, []string{"name", "role", "identity"})

	PublishedMsgsCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "choria_adapter_published_msgs",
		Help: "Number of messages published by adapters on their outputs",
	}, []string{"name", "role", "identity"})

	ErrorCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "choria_adapter_errors",
		Help: "Messages that could not be handled",
	}, []string{"name", "role", "identity"})

	BytesCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "choria_adapter_bytes",
		Help: "Bytes processed by the adapter",
	}, []string{"name", "role", "identity"})

	ProcessTime = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name: "choria_adapter_time",
		Help: "Time taken to process messages",
	}, []string{"name", "role", "identity"})

	WorkQueueLengthGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "choria_adapter_queue_length",
		Help: "How many messages are in the work queue to process",
	}, []string{"name", "identity"})

	WorkQueueCapacityGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "choria_adapter_queue_capacity",
		Help: "How many messages can be the work queue to process",
	}, []string{"name", "identity"})
)

func init() {
	prometheus.MustRegister(ReceivedMsgsCtr)
	prometheus.MustRegister(PublishedMsgsCtr)
	prometheus.MustRegister(ErrorCtr)
	prometheus.MustRegister(BytesCtr)
	prometheus.MustRegister(ProcessTime)
	prometheus.MustRegister(WorkQueueLengthGauge)
	prometheus.MustRegister(WorkQueueCapacityGauge)
}
