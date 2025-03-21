// generated code; DO NOT EDIT

package config

var docStrings = map[string]string{
	"registration":                    "The plugins used when publishing Registration data, when this is unset or empty sending registration data is disabled",
	"registration_collective":         "The Sub Collective to publish registration data to",
	"registerinterval":                "How often to publish registration data",
	"registration_splay":              "When true delays initial registration publish by a random period up to registerinterval following registration publishes will be at registerinterval without further splay",
	"collectives":                     "The list of known Sub Collectives this node will join or communicate with, Servers will subscribe the node and each agent to each sub collective and Clients will publish to a chosen sub collective. Defaults to the build settin build.DefaultCollectives",
	"main_collective":                 "The Sub Collective where a Client will publish to when no specific Sub Collective is configured",
	"logfile":                         "The file to write logs to, when set to 'discard' logging will be disabled. Also supports 'stdout' and 'stderr' as special log destinations.",
	"loglevel":                        "The lowest level log to add to the logfile",
	"libdir":                          "The directory where Agents, DDLs and other plugins are found",
	"identity":                        "The identity this machine is known as, when empty it's derived based on the operating system hostname or by calling facter fqdn",
	"color":                           "Disables or enable CLI color",
	"classesfile":                     "Path to a file listing configuration classes applied to a node, used in matches using Class filters",
	"discovery_timeout":               "How long to wait for responses while doing broadcast discovery",
	"rpcaudit":                        "When enabled uses rpcauditprovider to audit RPC requests processed by the server",
	"rpcauthorization":                "When enables authorization is performed on every RPC request based on rpcauthprovider",
	"rpcauthprovider":                 "The Authorization system to use",
	"rpclimitmethod":                  "When limiting nodes to a subset of discovered nodes this is the method to use, random is influenced by",
	"ttl":                             "How long published messages are allowed to linger on the network, lower numbers have a higher reliance on clocks being in sync",
	"default_discovery_method":        "The default discovery plugin to use. The default \"mc\" uses a network broadcast, \"choria\" uses PuppetDB, external calls external commands",
	"plugin.yaml":                     "Where to look for YAML or JSON based facts",
	"default_discovery_options":       "Default options to pass to the discovery plugin",
	"soft_shutdown_timeout":           "The amount of time to allow the server to exit, after this memory and thread dumps will be performed and a force exit will be done",
	"plugin.choria.puppetserver_host": "The hostname where your Puppet Server can be found",
	"plugin.choria.puppetserver_port": "The port your Puppet Server listens on",
	"plugin.choria.puppetca_host":     "The hostname where your Puppet Certificate Authority can be found",
	"plugin.choria.puppetca_port":     "The port your Puppet Certificate Authority listens on",
	"plugin.choria.puppetdb_host":     "The host hosting your PuppetDB, used by the \"choria\" discovery plugin",
	"plugin.choria.puppetdb_port":     "The port your PuppetDB listens on",
	"plugin.choria.use_srv":           "If SRV record lookups should be attempted to find Puppet, PuppetDB, Brokers etc",
	"plugin.choria.srv_domain":        "The domain to use for SRV records, defaults to the domain the server FQDN is in",
	"plugin.choria.server.provision":  "Specifically enable or disable provisioning",
	"plugin.choria.server.provision.allow_update":                  "Allows the provisioner to perform in-place version updates",
	"plugin.choria.discovery.external.command":                     "The command to use for external discovery",
	"plugin.choria.discovery.inventory.source":                     "The file to read for inventory discovery",
	"plugin.choria.discovery.broadcast.windowed_timeout":           "Enables the experimental dynamic timeout for choria/mc discovery",
	"plugin.choria.federation.collectives":                         "List of known remote collectives accessible via Federation Brokers",
	"plugin.choria.federation_middleware_hosts":                    "Middleware brokers used by the Federation Broker, if unset uses SRV",
	"plugin.choria.federation.cluster":                             "The cluster name a Federation Broker serves",
	"plugin.choria.stats_address":                                  "The address to listen on for statistics",
	"plugin.choria.stats_port":                                     "The port to listen on for HTTP requests for statistics, setting to 0 disables it",
	"plugin.choria.legacy_lifecycle_format":                        "When enabled will publish lifecycle events in the legacy format, else Cloud Events format is used",
	"plugin.nats.user":                                             "The user to connect to the NATS server as. When unset no username is used.",
	"plugin.nats.pass":                                             "The password to use when connecting to the NATS server",
	"plugin.nats.credentials":                                      "The NATS 2.0 credentials to use, required for accessing NGS",
	"plugin.choria.middleware_hosts":                               "Set specific middleware hosts in the format host:port, if unset uses SRV",
	"plugin.choria.network.client_hosts":                           "CIDRs to limit client connections from, appropriate ACLs are added based on this",
	"plugin.choria.network.public_url":                             "Name:Port to advertise to clients, useful when fronted by a proxy",
	"plugin.choria.network.client_port":                            "Port the Network Broker will accept client connections on",
	"plugin.choria.network.client_tls_force_required":              "Force requiring/not requiring TLS for all clients",
	"plugin.choria.network.client_signer_cert":                     "Fully qualified paths to the public certificates used by the AAA Service to sign client JWT tokens. This enables users with signed JWTs to use unverified TLS to connect. Can also be a list of ed25519 public keys.",
	"plugin.choria.network.deny_server_connections":                "Set ACLs denying server connections to this broker",
	"plugin.choria.network.stream.executor_retention":              "When not zero enables retaining Executor events in the Stream Store",
	"plugin.choria.network.stream.executor_replicas":               "When configuring Executor events ensure data is replicated in the cluster over this many servers, -1 means count of peers",
	"plugin.choria.network.stream.event_retention":                 "When not zero enables retaining Lifecycle events in the Stream Store",
	"plugin.choria.network.stream.event_replicas":                  "When configuring LifeCycle events ensure data is replicated in the cluster over this many servers, -1 means count of peers",
	"plugin.choria.network.gateway_name":                           "Name for the Super Cluster",
	"plugin.choria.network.gateway_port":                           "Port to listen on for Super Cluster connections",
	"plugin.choria.network.gateway_remotes":                        "List of remote Super Clusters to connect to",
	"plugin.choria.network.stream.leader_election_replicas":        "When configuring Stream based Leader Election storage ensure data is replicated in the cluster over this many servers, -1 means count of peers",
	"plugin.choria.network.stream.leader_election_ttl":             "The TTL for leader election, leaders must vote at least this frequently to remain leader",
	"plugin.choria.network.leafnode_port":                          "Port to listen on for Leafnode connections, disabled with 0",
	"plugin.choria.network.leafnode_remotes":                       "Remote networks to connect to as a Leafnode",
	"plugin.choria.network.listen_address":                         "Address the Network Broker will listen on",
	"plugin.choria.network.stream.machine_retention":               "When not zero enables retaining Autonomous Agent events in the Stream Store",
	"plugin.choria.network.stream.machine_replicas":                "When configuring Autonomous Agent event storage ensure data is replicated in the cluster over this many servers, -1 means count of peers",
	"plugin.choria.network.mapping.names":                          "List of subject remappings to apply",
	"plugin.choria.network.peer_password":                          "Password to use when connecting to cluster peers",
	"plugin.choria.network.peer_port":                              "Port used to communicate with other local cluster peers",
	"plugin.choria.network.peer_user":                              "Username to use when connecting to cluster peers",
	"plugin.choria.network.peers":                                  "List of cluster peers in host:port format",
	"plugin.choria.network.pprof_port":                             "The port the network broker will listen on for pprof requests",
	"plugin.choria.network.provisioning.client_password":           "Password the provisioned clients should use to connect",
	"plugin.choria.network.provisioning.provisioner_without_token": "Allows a provisioner without a token to connect over TLS using username and password.  This facilitates v1 provisioning on an Issuer based network",
	"plugin.choria.network.provisioning.signer_cert":               "Path to the public cert that signs provisioning tokens, enables accepting provisioning connections into the provisioning account",
	"plugin.choria.network.server_signer_cert":                     "Fully qualified Paths to the public certificates used by the Provisioner Service to sign server JWT tokens. This enables servers with signed JWTs to use unverified TLS to connect. Can also be a list of ed25519 public keys.",
	"plugin.choria.network.stream.advisory_retention":              "When not zero enables retaining Stream advisories in the Stream Store",
	"plugin.choria.network.stream.advisory_replicas":               "When configuring Stream advisories storage ensure data is replicated in the cluster over this many servers, -1 means count of peers",
	"plugin.choria.network.stream.manage_streams":                  "When set to zero will disable managing the standard streams on this node",
	"plugin.choria.network.stream.store":                           "Enables Streaming data persistence stored in this path",
	"plugin.choria.network.system.password":                        "Password used to access the Choria system account",
	"plugin.choria.network.system.user":                            "Username used to access the Choria system account",
	"plugin.choria.network.tls_timeout":                            "Time to allow for TLS connections to establish, increase on slow or very large networks",
	"plugin.choria.network.websocket_advertise":                    "The URL to advertise for websocket connections",
	"plugin.choria.network.websocket_port":                         "Port to listen on for websocket connections",
	"plugin.choria.network.write_deadline":                         "How long to allow clients to process traffic before treating them as slow, increase this on large networks or slow networks",
	"plugin.choria.network.soft_shutdown_timeout":                  "The amount of time to allow the broker to exit, after this memory and thread dumps will be performed and a force exit will be done",
	"plugin.choria.broker_network":                                 "Enables the Network Broker",
	"plugin.choria.broker_federation":                              "Enables the Federation Broker",
	"plugin.choria.adapters":                                       "The list of Data Adapters to activate",
	"plugin.choria.registration.file_content.data":                 "YAML or JSON file to use as data source for registration",
	"plugin.choria.registration.file_content.target":               "NATS Subject to publish registration data to",
	"plugin.choria.registration.file_content.compression":          "Enables gzip compression of registration data",
	"plugin.choria.registration.inventory_content.compression":     "Enables gzip compression of registration data",
	"plugin.choria.registration.inventory_content.target":          "NATS Subject to publish registration data to",
	"plugin.choria.agent_provider.mcorpc.agent_shim":               "Path to the helper used to call MCollective Ruby agents",
	"plugin.choria.agent_provider.mcorpc.config":                   "Path to the MCollective configuration file used when running MCollective Ruby agents",
	"plugin.choria.agent_provider.mcorpc.libdir":                   "Path to the libdir MCollective Ruby agents should have",
	"plugin.security.provider":                                     "The Security Provider to use",
	"plugin.security.server_anon_tls":                              "Use anonymous TLS to the Choria brokers from a server",
	"plugin.security.client_anon_tls":                              "Use anonymous TLS to the Choria brokers from a client, also disables security provider verification - only when a remote signer is set",
	"plugin.login.aaasvc.login.url":                                "List of URLs to attempt to login against when the remote signer is enabled",
	"plugin.security.cipher_suites":                                "List of allowed cipher suites",
	"plugin.security.ecc_curves":                                   "List of allowed ECC curves",
	"plugin.security.issuer.names":                                 "List of names of valid issuers this server will accept, set indvidiaul issuer data using plugin.security.issuer.<name>.public",
	"plugin.choria.security.server.token_file":                     "The server token file to use for authentication, defaults to serer.jwt in the same location as server.conf",
	"plugin.choria.security.server.seed_file":                      "The server token seed to use for authentication, defaults to server.seed in the same location as server.conf",
	"plugin.choria.ssldir":                                         "The SSL directory, auto detected via Puppet, when specifically set Puppet will not be consulted",
	"plugin.choria.security.privileged_users":                      "Patterns of certificate names that would be considered privileged and able to set custom callers",
	"plugin.choria.security.certname_whitelist":                    "Patterns of certificate names that are allowed to be clients",
	"plugin.security.support_legacy_certificates":                  "Allow certificates without SANs to be used",
	"plugin.choria.security.request_signer.seed_file":              "Path to the seed file used to access a Central Authenticator",
	"plugin.choria.security.request_signer.token_file":             "Path to the token used to access a Central Authenticator",
	"plugin.choria.security.request_signer.url":                    "URL to the Signing Service",
	"plugin.choria.security.request_signer.service":                "Enables signing requests via Choria RPC requests",
	"plugin.security.choria.trusted_signers":                       "Ed25119 public keys of entities allowed to sign client and server JWT tokens in hex encoded format",
	"plugin.security.choria.certificate":                           "When using choria security provider, the path to the optional public certificate",
	"plugin.security.choria.key":                                   "When using choria security provider, the path to the optional private key",
	"plugin.security.choria.ca":                                    "When using choria security provider, the path to the optional Certificate Authority public certificate",
	"plugin.security.choria.token_file":                            "The path to the JWT token file",
	"plugin.security.choria.seed_file":                             "The path to the seed file",
	"plugin.security.choria.sign_replies":                          "Disables signing replies which would significantly trim down the size of replies but would remove the ability to verify signatures or verify message origin",
	"plugin.security.file.certificate":                             "When using file security provider, the path to the public certificate",
	"plugin.security.file.key":                                     "When using file security provider, the path to the private key",
	"plugin.security.file.ca":                                      "When using file security provider, the path to the Certificate Authority public certificate",
	"plugin.security.certmanager.namespace":                        "When using Cert Manager security provider, the namespace the issuer is in",
	"plugin.security.certmanager.issuer":                           "When using Cert Manager security provider, the name of the issuer",
	"plugin.security.certmanager.replace":                          "when using Cert Manager security provider, replace existing CSRs with new ones",
	"plugin.security.certmanager.alt_names":                        "when using Cert Manager security provider, add these additional names to the CSR",
	"plugin.security.certmanager.api_version":                      "the API version to call in cert manager",
	"plugin.security.pkcs11.driver_file":                           "When using the pkcs11 security provider, the path to the PCS11 driver file",
	"plugin.security.pkcs11.slot":                                  "When using the pkcs11 security provider, the slot to use in the device",
	"plugin.choria.machine.store":                                  "Directory where Autonomous Agents are stored",
	"plugin.choria.machine.signing_key":                            "Public key used to sign data for watchers like machines watcher. Will override the value compiled in or in the watcher definitions if set here. This is primarily to allow development environments to use different private keys.",
	"plugin.choria.status_file_path":                               "Path to a JSON file to write server health information to regularly",
	"plugin.choria.status_update_interval":                         "How frequently to write to the status_file_path",
	"plugin.choria.prometheus_textfile_directory":                  "Directory where Prometheus Node Exporter textfile collector reads data",
	"plugin.scout.overrides":                                       "Path to a file holding overrides for Scout checks",
	"plugin.scout.tags":                                            "Path to a file holding tags for a Scout entity",
	"plugin.scout.agent_disabled":                                  "Disables the scout agent",
	"plugin.scout.goss.denied_local_resources":                     "List of resource types to deny for Goss manifests loaded from local disk",
	"plugin.scout.goss.denied_remote_resources":                    "List of resource types to deny when Goss manifests or variables were received over rpc",
	"plugin.choria.require_client_filter":                          "If a client filter should always be required, only used in Go clients",
	"plugin.choria.services.registry.store":                        "Directory where the Registry service finds DDLs to read",
	"plugin.choria.services.registry.cache":                        "Directory where the Registry client stores DDLs found in the registry",
	"plugin.choria.submission.spool":                               "Path to a directory holding messages to submit to the middleware",
	"plugin.choria.submission.max_spool_size":                      "Maximum amount of messages allowed into each priority",
	"plugin.rpcaudit.logfile":                                      "Path to the RPC audit log",
	"plugin.rpcaudit.logfile.group":                                "User group to set file ownership to",
	"plugin.rpcaudit.logfile.mode":                                 "File mode to apply to the file",
	"plugin.choria.executor.enabled":                               "Enables the long running command executor",
	"plugin.choria.executor.spool":                                 "Path where the command executor writes state",
	"plugin.machines.download":                                     "Activate run-time installation of Autonomous Agents",
	"plugin.machines.bucket":                                       "The KV bucket to query for plugins to install",
	"plugin.machines.key":                                          "The Key to query in KV bucket for plugins to install",
	"plugin.machines.purge":                                        "Purge autonomous agents installed using other methods",
	"plugin.machines.poll_interval":                                "How frequently to poll the KV bucket for updates",
	"plugin.machines.check_interval":                               "How frequently to integrity check deployed autonomous agents",
	"plugin.machines.signing_key":                                  "The public key to validate the plugins manifest with",
}
