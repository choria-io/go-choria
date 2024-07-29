// Copyright (c) 2017-2024, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package build

import (
	"strconv"
)

// Version the application version
var Version = "0.29.3"

// SHA is the git reference used to build this package
var SHA = "unknown"

// BuildDate is when it was build
var BuildDate = "unknown"

// License is the official Open Source Initiative license abbreviation
const License = "Apache-2.0"

// TLS controls the NATS protocol level TLS
var TLS = "true"

// maxBrokerClients defines the maximum clients a single choria broker will accept
var maxBrokerClients = "50000"

// ProvisionSecure when "false" will disable TLS provisioning mode
var ProvisionSecure = "true"

// ProvisionBrokerURLs defines where the daemon will connect when choria.server.provision is true
var ProvisionBrokerURLs = ""

// ProvisionBrokerSRVDomain defines a domain to query for provisioning brokers
var ProvisionBrokerSRVDomain = ""

// ProvisionModeDefault defines the value of plugin.choria.server.provision when it's not set
// in the configuration file at all.
var ProvisionModeDefault = "false"

// ProvisionAgent determines if the supplied provisioning agent should be started
// this lets you programmatically or via the additional agents system supply your own
// agent to perform the provisioning duties
var ProvisionAgent = "true"

// ProvisionRegistrationData is a file that will be published by the registration system
var ProvisionRegistrationData = ""

// ProvisionFacts is a facts file to use for discovery purposes during provisioning mode
var ProvisionFacts = ""

// ProvisionToken when not empty this token will be required interact with the provisioner agent
var ProvisionToken = ""

// ProvisionJWTFile is a file holding a JWT identifying the node to the provisioner
var ProvisionJWTFile = ""

// ProvisionStatusFile is the file where server status will be written to while in provisioning mode
var ProvisionStatusFile = ""

// ProvisioningBrokerUsername is the username used to connect to the middleware with
var ProvisioningBrokerUsername = ""

// ProvisioningBrokerPassword is the password used to connect to the middleware with
var ProvisioningBrokerPassword = ""

// ProvisioningUsesProtocolV2 indicates if provisioning should use v2 protocol
var ProvisioningUsesProtocolV2 = "false"

// ProvisionAllowServerUpdate allows over the air updates of the choria version from provisioner
var ProvisionAllowServerUpdate = "false"

// AgentProviders are registered systems capable of extending choria with new agents
var AgentProviders = []string{}

// MachineWatchers are registered Autonomous Agent watchers
var MachineWatchers = []string{}

// Machines are registered Autonomous Agents
var Machines = []string{}

// DataProviders are registered Data plugins
var DataProviders = []string{}

// DefaultCollectives is the names of the default list of collectives as comma separated strings
var DefaultCollectives = "mcollective"

// ClientIdentitySuffix is the string to be suffixed when creating client identities
var ClientIdentitySuffix = "mcollective"

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

// ProvisionSecurity determines if TLS should be enabled during provisioning
func ProvisionSecurity() bool {
	return ProvisionSecure == "true"
}
