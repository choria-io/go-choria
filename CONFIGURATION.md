# Choria Configuration Settings

This is a list of all known Configuration settings. This list is based on declared settings within the Choria Go code base and so will not cover 100% of settings - plugins can contribute their own settings.

Some emoji are used: 

 * :spider_web: Deprecated setting
 * :notebook: Additional information

|Key|Description|
|---|-----------|
|activate_agents :spider_web:|Undocumented|
|classesfile |Path to a file listing configuration classes applied to a node, used in matches using Class filters|
|collectives |The list of known Sub Collectives this node will join or communicate with, Servers will subscribe the node and each agent to each sub collective and Clients will publish to a chosen sub collective|
|color |Disables or enable CLI color, not well supported in Go based code|
|connection_timeout |Ruby clients use this to determine how long they will try to connect, fails after timeout|
|connector |Configures the network connector to use, only sensible value is "nats", unused in Go based code|
|daemonize :spider_web:|Undocumented|
|default_discovery_method |The default discovery plugin to use. The default "mc" uses a network broadcast and "choria" uses PuppetDB|
|default_discovery_options |Configurable options to always pass to the discovery subsystem|
|direct_addressing |Enables the direct-to-node communications pattern, unused in the Go clients|
|direct_addressing_threshold :spider_web:|Undocumented|
|discovery_timeout |How long to wait for responses while doing broadcast discovery|
|fact_cache_time :spider_web:|Undocumented|
|factsource :spider_web:|Undocumented|
|identity |The identity this machine is known as, when empty it's derived based on the operating system hostname or by calling facter fqnd|
|keeplogs :spider_web:|Undocumented|
|libdir |The directory where Agents, DDLs and other plugins are found|
|logfacility :spider_web:|Undocumented|
|logfile |The file to write logs to, when set to an empty string logging will be to the console|
|logger_type |The type of logging to use, unused in Go based programs|
|loglevel |The lowest level log to add to the logfile|
|main_collective |The Sub Collective where a Client will publish to when no specific Sub Collective is configured|
|max_log_size :spider_web:|Undocumented|
|plugin.choria.adapters [:notebook:](https://choria.io/docs/adapters/)|The list of Data Adapters to activate|
|plugin.choria.agent_provider.mcorpc.agent_shim |Path to the helper used to call MCollective Ruby agents|
|plugin.choria.agent_provider.mcorpc.config |Path to the MCollective configuration file used when running MCollective Ruby agents|
|plugin.choria.agent_provider.mcorpc.libdir |Path to the libdir MCollective Ruby agents should have|
|plugin.choria.broker_discovery :spider_web:|Undocumented|
|plugin.choria.broker_federation [:notebook:](https://choria.io/docs/federation/)|Enables the Federation Broker|
|plugin.choria.broker_network [:notebook:](https://choria.io/docs/deployment/broker/)|Enables the Network Broker|
|plugin.choria.discovery_host :spider_web:|discovery proxy|
|plugin.choria.discovery_port :spider_web:|Undocumented|
|plugin.choria.discovery_proxy :spider_web:|Undocumented|
|plugin.choria.federation.cluster [:notebook:](https://choria.io/docs/federation/)|The cluster name a Federation Broker serves|
|plugin.choria.federation.collectives [:notebook:](https://choria.io/docs/federation/)|List of known remote collectives accessible via Federation Brokers|
|plugin.choria.federation_middleware_hosts [:notebook:](https://choria.io/docs/federation/)|Middleware brokers used by the Federation Broker, if unset uses SRV|
|plugin.choria.legacy_lifecycle_format |When enabled will publish lifecycle events in the legacy format, else Cloud Events format is used|
|plugin.choria.machine.store [:notebook:](https://choria.io/docs/autoagents/)|Directory where Autonomous Agents are stored|
|plugin.choria.middleware_hosts |Set specific middleware hosts in the format host:port, if unset uses SRV|
|plugin.choria.network.client_hosts |CIDRs to limit client connections from, appropriate ACLs are added based on this|
|plugin.choria.network.client_port [:notebook:](https://choria.io/docs/deployment/broker/)|Port the Network Broker will accept client connections on|
|plugin.choria.network.client_tls_force_required |Force requiring/not requiring TLS for all clients|
|plugin.choria.network.gateway_name |Name for the Super Cluster|
|plugin.choria.network.gateway_port |Port to listen on for Super Cluster connections|
|plugin.choria.network.gateway_remotes |List of remote Super Clusters to connect to|
|plugin.choria.network.leafnode_port |Port to listen on for Leafnode connections, disabled with 0|
|plugin.choria.network.leafnode_remotes |Remote networks to connect to as a Leafnode|
|plugin.choria.network.listen_address [:notebook:](https://choria.io/docs/deployment/broker/)|Address the Network Broker will listen on|
|plugin.choria.network.operator_account |NATS 2.0 Operator account|
|plugin.choria.network.peer_password |Password to use when connecting to cluster peers|
|plugin.choria.network.peer_port [:notebook:](https://choria.io/docs/deployment/broker/)|Port used to communicate with other local cluster peers|
|plugin.choria.network.peer_user |Username to use when connecting to cluster peers|
|plugin.choria.network.peers [:notebook:](https://choria.io/docs/deployment/broker/)|List of cluster peers in host:port format|
|plugin.choria.network.system_account |NATS 2.0 System Account|
|plugin.choria.network.tls_timeout |Time to allow for TLS connections to establish, increase on slow or very large networks|
|plugin.choria.network.write_deadline |How long to allow clients to process traffic before treating them as slow, increase this on large networks or slow networks|
|plugin.choria.puppetca_host |The hostname where your Puppet Certificate Authority can be found|
|plugin.choria.puppetca_port |The port your Puppet Certificate Authority listens on|
|plugin.choria.puppetdb_host |The host hosting your PuppetDB, used by the "choria" discovery plugin|
|plugin.choria.puppetdb_port |The port your PuppetDB listens on|
|plugin.choria.puppetserver_host |The hostname where your Puppet Server can be found|
|plugin.choria.puppetserver_port |The port your Puppet Server listens on|
|plugin.choria.randomize_middleware_hosts |Shuffle middleware hosts before connecting to spread traffic of initial connections|
|plugin.choria.registration.file_content.compression |Enables gzip compression of registration data|
|plugin.choria.registration.file_content.data |YAML or JSON file to use as data source for registration|
|plugin.choria.registration.file_content.target |NATS Subject to publish registration data to|
|plugin.choria.security.certname_whitelist |Patterns of certificate names that are allowed to be clients|
|plugin.choria.security.privileged_users [:notebook:](https://choria.io/docs/configuration/aaa/)|Patterns of certificate names that would be considered privileged and able to set custom callers|
|plugin.choria.security.request_signer.token_environment [:notebook:](https://github.com/choria-io/aaasvc)|Environment variable to store Central Authenticator tokens|
|plugin.choria.security.request_signer.token_file [:notebook:](https://github.com/choria-io/aaasvc)|Path to the token used to access a Central Authenticator|
|plugin.choria.security.request_signer.url [:notebook:](https://github.com/choria-io/aaasvc)|URL to the Signing Service|
|plugin.choria.security.serializer :spider_web:|Undocumented|
|plugin.choria.server.provision [:notebook:](https://github.com/choria-io/provisioning-agent)|Specifically enable or disable provisioning|
|plugin.choria.srv_domain [:notebook:](https://choria.io/docs/deployment/dns/)|The domain to use for SRV records, defaults to the domain the server FQDN is in|
|plugin.choria.ssldir |The SSL directory, auto detected via Puppet, when specifically set Puppet will not be consulted|
|plugin.choria.stats_address |The address to listen on for statistics|
|plugin.choria.stats_port |The port to listen on for HTTP requests for statistics, setting to 0 disables it|
|plugin.choria.status_file_path |Path to a JSON file to write server health information to regularly|
|plugin.choria.status_update_interval |How frequently to write to the status_file_path|
|plugin.choria.use_srv [:notebook:](https://choria.io/docs/deployment/dns/)|If SRV record lookups should be attempted to find Puppet, PuppetDB, Brokers etc|
|plugin.nats.credentials |The NATS 2.0 credentials to use, required for accessing NGS|
|plugin.nats.ngs |Uses NATS NGS global managed network as middleware, overrides broker names to "connect.ngs.global"|
|plugin.nats.pass |The password to use when connecting to the NATS server|
|plugin.nats.user |The user to connect to the NATS server as. When unset no username is used.|
|plugin.security.always_overwrite_cache |Always store new Public Keys to the cache overwriting existing ones|
|plugin.security.cipher_suites |List of allowed cipher suites|
|plugin.security.ecc_curves |List of allowed ECC curves|
|plugin.security.file.ca |When using file security provider, the path to the Certificate Authority public certificate|
|plugin.security.file.cache |When using file security provider, the path to the client cache|
|plugin.security.file.certificate |When using file security provider, the path to the public certificate|
|plugin.security.file.key |When using file security provider, the path to the private key|
|plugin.security.pkcs11.driver_file [:notebook:](https://choria.io/blog/post/2019/09/09/pkcs11/)|When using the pkcs11 security provider, the path to the PCS11 driver file|
|plugin.security.pkcs11.slot [:notebook:](https://choria.io/blog/post/2019/09/09/pkcs11/)|When using the pkcs11 security provider, the slot to use in the device|
|plugin.security.provider |The Security Provider to use|
|plugin.yaml |Where to look for YAML or JSON based facts|
|publish_timeout |Ruby clients use this to determine how long they will allow when publishing requests|
|registerinterval |How often to publish registration data|
|registration |The plugins used when publishing Registration data, when this is unset or empty sending registration data is disabled|
|registration_collective |The Sub Collective to publish registration data to|
|registration_splay |When true delays initial registration publish by a random period up to registerinterval following registration publishes will be at registerinterval without further splay|
|require_client_filter |If a client filter should always be required, appears unused at the moment|
|rpcaudit [:notebook:](https://choria.io/docs/configuration/aaa/)|When enabled uses rpcauditprovider to audit RPC requests processed by the server|
|rpcauditprovider [:notebook:](https://choria.io/docs/configuration/aaa/)|The audit provider to use, unused at present as there is only a "choria" one|
|rpcauthorization [:notebook:](https://choria.io/docs/configuration/aaa/)|When enables authorization is performed on every RPC request based on rpcauthprovider|
|rpcauthprovider [:notebook:](https://choria.io/docs/configuration/aaa/)|The Authorization system to use|
|rpclimitmethod |When limiting nodes to a subset of discovered nodes this is the method to use, random is influenced by|
|securityprovider :spider_web:|Used to select the security provider in Ruby clients, only sensible value is "choria"|
|soft_shutdown :spider_web:|Undocumented|
|soft_shutdown_timeout :spider_web:|Undocumented|
|threaded |Enables multi threaded mode in the Ruby client, generally a bad idea|
|ttl |How long published messages are allowed to linger on the network, lower numbers have a higher reliance on clocks being in sync|
