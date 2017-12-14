package v1

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	badJsonCtr = prometheus.NewCounter(prometheus.CounterOpts{
		Name:        "choria_protocol_json_parse_failures",
		Help:        "Amount of times unparsable JSON was received",
		ConstLabels: prometheus.Labels{"version": "1"},
	})

	invalidJsonCtr = prometheus.NewCounter(prometheus.CounterOpts{
		Name:        "choria_protocol_json_validation_failures",
		Help:        "Amount of times parsable JSON did not match the expected schema",
		ConstLabels: prometheus.Labels{"version": "1"},
	})

	signFailCtr = prometheus.NewCounter(prometheus.CounterOpts{
		Name:        "choria_protocol_signing_failures",
		Help:        "Amount of times signing a message failed",
		ConstLabels: prometheus.Labels{"version": "1"},
	})

	validCtr = prometheus.NewCounter(prometheus.CounterOpts{
		Name:        "choria_protocol_valid",
		Help:        "Number of messages with valid signatures",
		ConstLabels: prometheus.Labels{"version": "1"},
	})

	invalidCtr = prometheus.NewCounter(prometheus.CounterOpts{
		Name:        "choria_protocol_invalid",
		Help:        "Number of messages with invalid signatures",
		ConstLabels: prometheus.Labels{"version": "1"},
	})

	invalidCertificateCtr = prometheus.NewCounter(prometheus.CounterOpts{
		Name:        "choria_protocol_security_invalid_certificate",
		Help:        "Number of messages with unverifiable certificates",
		ConstLabels: prometheus.Labels{"version": "1"},
	})

	protocolErrorCtr = prometheus.NewCounter(prometheus.CounterOpts{
		Name:        "choria_protocol_error",
		Help:        "Number of protocol errors such as missing certificate, unparsable files and other unrecoverable errors",
		ConstLabels: prometheus.Labels{"version": "1"},
	})
)

func init() {
	prometheus.MustRegister(badJsonCtr)
	prometheus.MustRegister(invalidJsonCtr)
	prometheus.MustRegister(signFailCtr)
	prometheus.MustRegister(validCtr)
	prometheus.MustRegister(invalidCtr)
	prometheus.MustRegister(invalidCertificateCtr)
	prometheus.MustRegister(protocolErrorCtr)
}
