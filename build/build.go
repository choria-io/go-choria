package build

import (
	"strconv"
)

// Version the application version
var Version = "development"

// SHA is the git reference used to build this package
var SHA = "unknown"

// BuildDate is when it was build
var BuildDate = "unknown"

// License is the official Open Source Initiave license abbreciation
var License = "Apache-2.0"

// TLS controls the NATS protocol level TLS
var TLS = "true"

// maxBrokerClients defines the maximum clients a single choria broker will accept
var maxBrokerClients = "50000"

// ProvisionBrokerURLs defines where the daemon will connect when choria.server.provision is true
var ProvisionBrokerURLs = ""

// ProvisionModeDefault defines the value of plugin.choria.server.provision when it's not set
// in the configuration file at all.
var ProvisionModeDefault = "false"

// ProvisionAgent determines if the supplied provisioning agent should be started
// this lets you programatically or via the additional agents system supply your own
// agent to perform the provisioning duties
var ProvisionAgent = "true"

// HasTLS determines if TLS should be used on the wire
func HasTLS() bool {
	return TLS == "true"
}

// MaxBrokerClients is the maximum number of clients the network broker may handle
func MaxBrokerClients() int {
	c, err := strconv.Atoi(maxBrokerClients)
	if err != nil {
		return 50000
	}

	return c
}

// ProvisionDefault defines the value of plugin.choria.server.provision when it's not set
// in the configuration file at all.
func ProvisionDefault() bool {
	return ProvisionModeDefault == "true"
}
