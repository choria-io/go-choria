// Copyright (c) 2018-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"time"

	"github.com/choria-io/go-choria/confkey"
	log "github.com/sirupsen/logrus"
)

// ChoriaPluginConfig settings
//
// NOTE: When adding or updating doc strings please run `go generate` in the root of the repository
type ChoriaPluginConfig struct {
	PuppetServerHost string `confkey:"plugin.choria.puppetserver_host" default:"puppet"`                                              // The hostname where your Puppet Server can be found
	PuppetServerPort int    `confkey:"plugin.choria.puppetserver_port" default:"8140"`                                                // The port your Puppet Server listens on
	PuppetCAHost     string `confkey:"plugin.choria.puppetca_host" default:"puppet"`                                                  // The hostname where your Puppet Certificate Authority can be found
	PuppetCAPort     int    `confkey:"plugin.choria.puppetca_port" default:"8140"`                                                    // The port your Puppet Certificate Authority listens on
	PuppetDBHost     string `confkey:"plugin.choria.puppetdb_host" default:""`                                                        // The host hosting your PuppetDB, used by the "choria" discovery plugin
	PuppetDBPort     int    `confkey:"plugin.choria.puppetdb_port" default:"8081"`                                                    // The port your PuppetDB listens on
	UseSRVRecords    bool   `confkey:"plugin.choria.use_srv" default:"true" url:"https://choria.io/docs/deployment/dns/"`             // If SRV record lookups should be attempted to find Puppet, PuppetDB, Brokers etc
	SRVDomain        string `confkey:"plugin.choria.srv_domain" url:"https://choria.io/docs/deployment/dns/"`                         // The domain to use for SRV records, defaults to the domain the server FQDN is in
	Provision        bool   `confkey:"plugin.choria.server.provision" default:"false" url:"https://github.com/choria-io/provisioner"` // Specifically enable or disable provisioning

	ExternalDiscoveryCommand         string `confkey:"plugin.choria.discovery.external.command" type:"path_string"` // The command to use for external discovery
	InventoryDiscoverySource         string `confkey:"plugin.choria.discovery.inventory.source" type:"path_string"` // The file to read for inventory discovery
	BroadcastDiscoveryDynamicTimeout bool   `confkey:"plugin.choria.discovery.broadcast.windowed_timeout"`          // Enables the experimental dynamic timeout for choria/mc discovery

	FederationCollectives     []string `confkey:"plugin.choria.federation.collectives" type:"comma_split" environment:"CHORIA_FED_COLLECTIVE" url:"https://choria.io/docs/federation/"` // List of known remote collectives accessible via Federation Brokers
	FederationMiddlewareHosts []string `confkey:"plugin.choria.federation_middleware_hosts" type:"comma_split" url:"https://choria.io/docs/federation/"`                                // Middleware brokers used by the Federation Broker, if unset uses SRV
	FederationCluster         string   `confkey:"plugin.choria.federation.cluster" default:"mcollective" url:"https://choria.io/docs/federation/"`                                      // The cluster name a Federation Broker serves

	StatsListenAddress    string `confkey:"plugin.choria.stats_address" default:"127.0.0.1"`   // The address to listen on for statistics
	StatsPort             int    `confkey:"plugin.choria.stats_port" default:"0"`              // The port to listen on for HTTP requests for statistics, setting to 0 disables it
	LegacyLifeCycleFormat bool   `confkey:"plugin.choria.legacy_lifecycle_format" default:"0"` // When enabled will publish lifecycle events in the legacy format, else Cloud Events format is used

	NatsUser                 string   `confkey:"plugin.nats.user" environment:"MCOLLECTIVE_NATS_USERNAME"`           // The user to connect to the NATS server as. When unset no username is used.
	NatsPass                 string   `confkey:"plugin.nats.pass" environment:"MCOLLECTIVE_NATS_PASSWORD"`           // The password to use when connecting to the NATS server
	NatsCredentials          string   `confkey:"plugin.nats.credentials" environment:"MCOLLECTIVE_NATS_CREDENTIALS"` // The NATS 2.0 credentials to use, required for accessing NGS
	NatsNGS                  bool     `confkey:"plugin.nats.ngs" environment:"MCOLLECTIVE_NATS_NGS"`                 // Uses NATS NGS global managed network as middleware, overrides broker names to "connect.ngs.global"
	MiddlewareHosts          []string `confkey:"plugin.choria.middleware_hosts" type:"comma_split"`                  // Set specific middleware hosts in the format host:port, if unset uses SRV
	RandomizeMiddlewareHosts bool     `confkey:"plugin.choria.randomize_middleware_hosts" default:"true"`            // Shuffle middleware hosts before connecting to spread traffic of initial connections

	NetworkListenAddress               string        `confkey:"plugin.choria.network.listen_address" default:"::" url:"https://choria.io/docs/deployment/broker/"` // Address the Network Broker will listen on
	NetworkWebSocketPort               int           `confkey:"plugin.choria.network.websocket_port" url:"https://choria.io/docs/deployment/broker/"`              // Port to listen on for websocket connections
	NetworkWebSocketAdvertise          string        `confkey:"plugin.choria.network.websocket_advertise" url:"https://choria.io/docs/deployment/broker/"`         // The URL to advertise for websocket connections
	NetworkClientPort                  int           `confkey:"plugin.choria.network.client_port" default:"4222" url:"https://choria.io/docs/deployment/broker/"`  // Port the Network Broker will accept client connections on
	NetworkClientTLSForce              bool          `confkey:"plugin.choria.network.client_tls_force_required"`                                                   // Force requiring/not requiring TLS for all clients
	NetworkClientTLSAnon               bool          `confkey:"plugin.choria.network.client_anon_tls"  deprecated:"1"`                                             // Now enabled using plugin.choria.network.client_signer_cert
	NetworkPeerPort                    int           `confkey:"plugin.choria.network.peer_port" url:"https://choria.io/docs/deployment/broker/"`                   // Port used to communicate with other local cluster peers
	NetworkPeerUser                    string        `confkey:"plugin.choria.network.peer_user"`                                                                   // Username to use when connecting to cluster peers
	NetworkPeerPassword                string        `confkey:"plugin.choria.network.peer_password"`                                                               // Password to use when connecting to cluster peers
	NetworkPeers                       []string      `confkey:"plugin.choria.network.peers" type:"comma_split" url:"https://choria.io/docs/deployment/broker/"`    // List of cluster peers in host:port format
	NetworkLeafPort                    int           `confkey:"plugin.choria.network.leafnode_port" default:"0"`                                                   // Port to listen on for Leafnode connections, disabled with 0
	NetworkLeafRemotes                 []string      `confkey:"plugin.choria.network.leafnode_remotes" type:"comma_split"`                                         // Remote networks to connect to as a Leafnode
	NetworkGatewayPort                 int           `confkey:"plugin.choria.network.gateway_port" default:"0"`                                                    // Port to listen on for Super Cluster connections
	NetworkGatewayName                 string        `confkey:"plugin.choria.network.gateway_name" default:"CHORIA"`                                               // Name for the Super Cluster
	NetworkGatewayRemotes              []string      `confkey:"plugin.choria.network.gateway_remotes" type:"comma_split"`                                          // List of remote Super Clusters to connect to
	NetworkWriteDeadline               time.Duration `confkey:"plugin.choria.network.write_deadline" type:"duration" default:"10s"`                                // How long to allow clients to process traffic before treating them as slow, increase this on large networks or slow networks
	NetworkAllowedClientHosts          []string      `confkey:"plugin.choria.network.client_hosts" type:"comma_split"`                                             // CIDRs to limit client connections from, appropriate ACLs are added based on this
	NetworkClientTokenSignerFile       string        `confkey:"plugin.choria.network.client_signer_cert" type:"path_string"`                                       // Path to the public certificate used by the AAA Service to sign client JWT tokens. This enables users with signed JWTs to use unverified TLS to connect
	NetworkDenyServers                 bool          `confkey:"plugin.choria.network.deny_server_connections"`                                                     // Set ACLs denying server connections to this broker
	NetworkTLSTimeout                  int           `confkey:"plugin.choria.network.tls_timeout" default:"2"`                                                     // Time to allow for TLS connections to establish, increase on slow or very large networks
	NetworkClientAdvertiseName         string        `confkey:"plugin.choria.network.public_url"`                                                                  // Name:Port to advertise to clients, useful when fronted by a proxy
	NetworkStreamStore                 string        `confkey:"plugin.choria.network.stream.store" type:"path_string"`                                             // Enables Streaming data persistence stored in this path
	NetworkEventStoreDuration          time.Duration `confkey:"plugin.choria.network.stream.event_retention" type:"duration" default:"24h"`                        // When not zero enables retaining Lifecycle events in the Stream Store
	NetworkEventStoreReplicas          int           `confkey:"plugin.choria.network.stream.event_replicas" default:"-1"`                                          // When configuring LifeCycle events ensure data is replicated in the cluster over this many servers, -1 means count of peers
	NetworkMachineStoreDuration        time.Duration `confkey:"plugin.choria.network.stream.machine_retention" type:"duration" default:"24h"`                      // When not zero enables retaining Autonomous Agent events in the Stream Store
	NetworkMachineStoreReplicas        int           `confkey:"plugin.choria.network.stream.machine_replicas" default:"-1"`                                        // When configuring Autonomous Agent event storage ensure data is replicated in the cluster over this many servers, -1 means count of peers
	NetworkStreamAdvisoryDuration      time.Duration `confkey:"plugin.choria.network.stream.advisory_retention" type:"duration" default:"168h"`                    // When not zero enables retaining Stream advisories in the Stream Store
	NetworkStreamAdvisoryReplicas      int           `confkey:"plugin.choria.network.stream.advisory_replicas" default:"-1"`                                       // When configuring Stream advisories storage ensure data is replicated in the cluster over this many servers, -1 means count of peers
	NetworkLeaderElectionReplicas      int           `confkey:"plugin.choria.network.stream.leader_election_replicas" default:"-1"`                                // When configuring Stream based Leader Election storage ensure data is replicated in the cluster over this many servers, -1 means count of peers
	NetworkLeaderElectionTTL           time.Duration `confkey:"plugin.choria.network.stream.leader_election_ttl" type:"duration" default:"1m"`                     // The TTL for leader election, leaders must vote at least this frequently to remain leader
	NetworkSystemUsername              string        `confkey:"plugin.choria.network.system.user"`                                                                 // Username used to access the Choria system account
	NetworkSystemPassword              string        `confkey:"plugin.choria.network.system.password"`                                                             // Password used to access the Choria system account
	NetworkProfilePort                 int           `confkey:"plugin.choria.network.pprof_port" default:"0"`                                                      // The port the network broker will listen on for pprof requests
	NetworkProvisioningTokenSignerFile string        `confkey:"plugin.choria.network.provisioning.signer_cert" type:"path_string"`                                 // Path to the public cert that signs provisioning tokens, enables accepting provisioning connections into the provisioning account
	NetworkProvisioningClientPassword  string        `confkey:"plugin.choria.network.provisioning.client_password"`                                                // Password the provisioned clients should use to connect

	BrokerNetwork    bool `confkey:"plugin.choria.broker_network" default:"false" url:"https://choria.io/docs/deployment/broker/"` // Enables the Network Broker
	BrokerDiscovery  bool `confkey:"plugin.choria.broker_discovery" default:"false" deprecated:"1"`
	BrokerFederation bool `confkey:"plugin.choria.broker_federation" default:"false" url:"https://choria.io/docs/federation/"` // Enables the Federation Broker

	FileContentRegistrationData        string `confkey:"plugin.choria.registration.file_content.data" default:""`                 // YAML or JSON file to use as data source for registration
	FileContentRegistrationTarget      string `confkey:"plugin.choria.registration.file_content.target" default:""`               // NATS Subject to publish registration data to
	FileContentCompression             bool   `confkey:"plugin.choria.registration.file_content.compression" default:"true"`      // Enables gzip compression of registration data
	InventoryContentCompression        bool   `confkey:"plugin.choria.registration.inventory_content.compression" default:"true"` // Enables gzip compression of registration data
	InventoryContentRegistrationTarget string `confkey:"plugin.choria.registration.inventory_content.target" default:""`          // NATS Subject to publish registration data to

	RubyAgentShim   string   `confkey:"plugin.choria.agent_provider.mcorpc.agent_shim"`               // Path to the helper used to call MCollective Ruby agents
	RubyAgentConfig string   `confkey:"plugin.choria.agent_provider.mcorpc.config"`                   // Path to the MCollective configuration file used when running MCollective Ruby agents
	RubyLibdir      []string `confkey:"plugin.choria.agent_provider.mcorpc.libdir" type:"path_split"` // Path to the libdir MCollective Ruby agents should have

	SSLDir                       string   `confkey:"plugin.choria.ssldir" type:"path_string"`                                                                                                                               // The SSL directory, auto detected via Puppet, when specifically set Puppet will not be consulted
	PrivilegedUsers              []string `confkey:"plugin.choria.security.privileged_users" type:"comma_split" default:"\\.privileged.mcollective$,\\.privileged.choria$" url:"https://choria.io/docs/configuration/aaa/"` // Patterns of certificate names that would be considered privileged and able to set custom callers
	CertnameWhitelist            []string `confkey:"plugin.choria.security.certname_whitelist" type:"comma_split" default:"\\.mcollective$,\\.choria$"`                                                                     // Patterns of certificate names that are allowed to be clients
	Serializer                   string   `confkey:"plugin.choria.security.serializer" validate:"enum=json,yaml" default:"json" deprecated:"1"`
	SecurityProvider             string   `confkey:"plugin.security.provider" default:"puppet" validate:"enum=puppet,file,pkcs11,certmanager"`                      // The Security Provider to use
	SecurityAlwaysOverwriteCache bool     `confkey:"plugin.security.always_overwrite_cache" default:"false"`                                                        // Always store new Public Keys to the cache overwriting existing ones
	SecurityAllowLegacyCerts     bool     `confkey:"plugin.security.support_legacy_certificates" default:"false"`                                                   // Allow certificates without SANs to be used
	RemoteSignerTokenSeedFile    string   `confkey:"plugin.choria.security.request_signer.seed_file" type:"path_string" url:"https://github.com/choria-io/aaasvc"`  // Path to the seed file used to access a Central Authenticator
	RemoteSignerTokenFile        string   `confkey:"plugin.choria.security.request_signer.token_file" type:"path_string" url:"https://github.com/choria-io/aaasvc"` // Path to the token used to access a Central Authenticator
	RemoteSignerSigningCertFile  string   `confkey:"plugin.choria.security.request_signing_certificate" type:"path_string"`                                         // Path to the public certificate of the key used to sign the JWTs in the Signing Service
	RemoteSignerURL              string   `confkey:"plugin.choria.security.request_signer.url" url:"https://github.com/choria-io/aaasvc"`                           // URL to the Signing Service
	RemoteSignerService          bool     `confkey:"plugin.choria.security.request_signer.service" url:"https://github.com/choria-io/aaasvc"`                       // Enables signing requests via Choria RPC requests
	AAAServiceLoginURLs          []string `confkey:"plugin.login.aaasvc.login.url" url:"https://github.com/choria-io/aaasvc"`                                       // List of URLs to attempt to login against when the remote signer is enabled
	ClientAnonTLS                bool     `confkey:"plugin.security.client_anon_tls" default:"false"`                                                               // Use anonymous TLS to the Choria brokers from a client, also disables security provider verification - only when a remote signer is set

	FileSecurityCertificate string `confkey:"plugin.security.file.certificate" type:"path_string"` // When using file security provider, the path to the public certificate
	FileSecurityKey         string `confkey:"plugin.security.file.key" type:"path_string"`         // When using file security provider, the path to the private key
	FileSecurityCA          string `confkey:"plugin.security.file.ca" type:"path_string"`          // When using file security provider, the path to the Certificate Authority public certificate
	FileSecurityCache       string `confkey:"plugin.security.file.cache" type:"path_string"`       // When using file security provider, the path to the client cache

	CertManagerSecurityNamespace  string   `confkey:"plugin.security.certmanager.namespace" default:"choria"`   // When using Cert Manager security provider, the namespace the issuer is in
	CertManagerSecurityIssuer     string   `confkey:"plugin.security.certmanager.issuer"`                       // When using Cert Manager security provider, the name of the issuer
	CertManagerSecurityReplaceCSR bool     `confkey:"plugin.security.certmanager.replace" default:"true"`       // when using Cert Manager security provider, replace existing CSRs with new ones
	CertManagerSecurityAltNames   []string `confkey:"plugin.security.certmanager.alt_names" type:"comma_split"` // when using Cert Manager security provider, add these additional names to the CSR
	CertManagerAPIVersion         string   `confkey:"plugin.security.certmanager.api_version" default:"v1"`     // the API version to call in cert manager

	CipherSuites []string `confkey:"plugin.security.cipher_suites" type:"comma_split"` // List of allowed cipher suites
	ECCCurves    []string `confkey:"plugin.security.ecc_curves" type:"comma_split"`    // List of allowed ECC curves

	PKCS11DriverFile string `confkey:"plugin.security.pkcs11.driver_file" type:"path_string" url:"https://choria.io/blog/post/2019/09/09/pkcs11/"` // When using the pkcs11 security provider, the path to the PCS11 driver file
	PKCS11Slot       int    `confkey:"plugin.security.pkcs11.slot" url:"https://choria.io/blog/post/2019/09/09/pkcs11/"`                           // When using the pkcs11 security provider, the slot to use in the device

	Adapters []string `confkey:"plugin.choria.adapters" type:"comma_split" url:"https://choria.io/docs/adapters/"` // The list of Data Adapters to activate

	StatusFilePath      string `confkey:"plugin.choria.status_file_path" type:"path_string"` // Path to a JSON file to write server health information to regularly
	StatusUpdateSeconds int    `confkey:"plugin.choria.status_update_interval" default:"30"` // How frequently to write to the status_file_path

	MachineSourceDir string `confkey:"plugin.choria.machine.store" url:"https://choria.io/docs/autoagents/"` // Directory where Autonomous Agents are stored

	PrometheusTextFileDir string `confkey:"plugin.choria.prometheus_textfile_directory" type:"path_string"` // Directory where Prometheus Node Exporter textfile collector reads data
	ScoutOverrides        string `confkey:"plugin.scout.overrides" type:"path_string"`                      // Path to a file holding overrides for Scout checks
	ScoutTags             string `confkey:"plugin.scout.tags" type:"path_string"`                           // Path to a file holding tags for a Scout entity
	ScoutAgentDisabled    bool   `confkey:"plugin.scout.agent_disabled"`                                    // Disables the scout agent

	RequireClientFilter bool `confkey:"plugin.choria.require_client_filter" default:"false"` // If a client filter should always be required, only used in Go clients

	RegistryServiceStore string `confkey:"plugin.choria.services.registry.store" type:"path_string"`                                // Directory where the Registry service finds DDLs to read
	RegistryClientCache  string `confkey:"plugin.choria.services.registry.cache" type:"path_string"  environment:"CHORIA_REGISTRY"` // Directory where the Registry client stores DDLs found in the registry

	SubmissionSpool        string `confkey:"plugin.choria.submission.spool" type:"path_string"`     // Path to a directory holding messages to submit to the middleware
	SubmissionSpoolMaxSize int    `confkey:"plugin.choria.submission.max_spool_size" default:"500"` // Maximum amount of messages allowed into each priority
}

func newChoria() *ChoriaPluginConfig {
	c := &ChoriaPluginConfig{}

	err := confkey.SetStructDefaults(c)
	if err != nil {
		log.Errorf("Choria config creation failed: %s", err)
	}

	return c
}
