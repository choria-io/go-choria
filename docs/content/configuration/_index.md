+++
title = "Config Reference"
toc = true
weight = 40
pre = "<b>4. </b>"
+++

This is a list of all known Configuration settings. This list is based on declared settings within the Choria Go code base and so will not cover 100% of settings - plugins can contribute their own settings which are note known at compile time.

{{% notice secondary "Version Hint" code-branch %}}
Built on *21 Feb 23 11:53 UTC* using version *0.26.2*
{{% /notice %}}

### Run-time configuration

The run-time configuration can be inspected using `choria tool config --config /etc/choria/server.cfg`, this will show the active configuration.

### Search and list directives

In addition to the full list below you can get configuration information for your version using the CLI:

```nohighlight
% choria tool config security.provider
....
Configuration item: plugin.security.provider

║        Value: puppet
║    Data Type: string
║   Validation: enum=puppet,file,pkcs11,certmanager,choria
║      Default: puppet
║
║ The Security Provider to use
╙─
```

### Data Types

A few special types are defined, the rest map to standard Go types

|Type|Description|
|----|-----------|
|comma_split|A comma separated list of strings, possibly with spaces between|
|duration|A duration such as `1h`, `300ms`, `-1.5h` or `2h45m`. Valid time units are `ns`, `ms`, `s`, `m`, `h`|
|path_split|A list of paths split by a OS specific PATH separator|
|path_string|A path that can include `~` for the users home directory|
|strings|A space separated list of strings|
|title_string|A string that will be stored as a `Title String`|

### Index

| | |
|-|-|
|[classesfile](#classesfile)|[collectives](#collectives)|
|[color](#color)|[default_discovery_method](#default_discovery_method)|
|[default_discovery_options](#default_discovery_options)|[discovery_timeout](#discovery_timeout)|
|[identity](#identity)|[libdir](#libdir)|
|[logfile](#logfile)|[loglevel](#loglevel)|
|[main_collective](#main_collective)|[plugin.choria.adapters](#pluginchoriaadapters)|
|[plugin.choria.agent_provider.mcorpc.agent_shim](#pluginchoriaagent_providermcorpcagent_shim)|[plugin.choria.agent_provider.mcorpc.config](#pluginchoriaagent_providermcorpcconfig)|
|[plugin.choria.agent_provider.mcorpc.libdir](#pluginchoriaagent_providermcorpclibdir)|[plugin.choria.broker_federation](#pluginchoriabroker_federation)|
|[plugin.choria.broker_network](#pluginchoriabroker_network)|[plugin.choria.discovery.broadcast.windowed_timeout](#pluginchoriadiscoverybroadcastwindowed_timeout)|
|[plugin.choria.discovery.external.command](#pluginchoriadiscoveryexternalcommand)|[plugin.choria.discovery.inventory.source](#pluginchoriadiscoveryinventorysource)|
|[plugin.choria.federation.cluster](#pluginchoriafederationcluster)|[plugin.choria.federation.collectives](#pluginchoriafederationcollectives)|
|[plugin.choria.federation_middleware_hosts](#pluginchoriafederation_middleware_hosts)|[plugin.choria.legacy_lifecycle_format](#pluginchorialegacy_lifecycle_format)|
|[plugin.choria.machine.signing_key](#pluginchoriamachinesigning_key)|[plugin.choria.machine.store](#pluginchoriamachinestore)|
|[plugin.choria.middleware_hosts](#pluginchoriamiddleware_hosts)|[plugin.choria.network.client_hosts](#pluginchorianetworkclient_hosts)|
|[plugin.choria.network.client_port](#pluginchorianetworkclient_port)|[plugin.choria.network.client_signer_cert](#pluginchorianetworkclient_signer_cert)|
|[plugin.choria.network.client_tls_force_required](#pluginchorianetworkclient_tls_force_required)|[plugin.choria.network.deny_server_connections](#pluginchorianetworkdeny_server_connections)|
|[plugin.choria.network.gateway_name](#pluginchorianetworkgateway_name)|[plugin.choria.network.gateway_port](#pluginchorianetworkgateway_port)|
|[plugin.choria.network.gateway_remotes](#pluginchorianetworkgateway_remotes)|[plugin.choria.network.leafnode_port](#pluginchorianetworkleafnode_port)|
|[plugin.choria.network.leafnode_remotes](#pluginchorianetworkleafnode_remotes)|[plugin.choria.network.listen_address](#pluginchorianetworklisten_address)|
|[plugin.choria.network.mapping.names](#pluginchorianetworkmappingnames)|[plugin.choria.network.peer_password](#pluginchorianetworkpeer_password)|
|[plugin.choria.network.peer_port](#pluginchorianetworkpeer_port)|[plugin.choria.network.peer_user](#pluginchorianetworkpeer_user)|
|[plugin.choria.network.peers](#pluginchorianetworkpeers)|[plugin.choria.network.pprof_port](#pluginchorianetworkpprof_port)|
|[plugin.choria.network.provisioning.client_password](#pluginchorianetworkprovisioningclient_password)|[plugin.choria.network.provisioning.provisioner_without_token](#pluginchorianetworkprovisioningprovisioner_without_token)|
|[plugin.choria.network.provisioning.signer_cert](#pluginchorianetworkprovisioningsigner_cert)|[plugin.choria.network.public_url](#pluginchorianetworkpublic_url)|
|[plugin.choria.network.server_signer_cert](#pluginchorianetworkserver_signer_cert)|[plugin.choria.network.stream.advisory_replicas](#pluginchorianetworkstreamadvisory_replicas)|
|[plugin.choria.network.stream.advisory_retention](#pluginchorianetworkstreamadvisory_retention)|[plugin.choria.network.stream.event_replicas](#pluginchorianetworkstreamevent_replicas)|
|[plugin.choria.network.stream.event_retention](#pluginchorianetworkstreamevent_retention)|[plugin.choria.network.stream.leader_election_replicas](#pluginchorianetworkstreamleader_election_replicas)|
|[plugin.choria.network.stream.leader_election_ttl](#pluginchorianetworkstreamleader_election_ttl)|[plugin.choria.network.stream.machine_replicas](#pluginchorianetworkstreammachine_replicas)|
|[plugin.choria.network.stream.machine_retention](#pluginchorianetworkstreammachine_retention)|[plugin.choria.network.stream.manage_streams](#pluginchorianetworkstreammanage_streams)|
|[plugin.choria.network.stream.store](#pluginchorianetworkstreamstore)|[plugin.choria.network.system.password](#pluginchorianetworksystempassword)|
|[plugin.choria.network.system.user](#pluginchorianetworksystemuser)|[plugin.choria.network.tls_timeout](#pluginchorianetworktls_timeout)|
|[plugin.choria.network.websocket_advertise](#pluginchorianetworkwebsocket_advertise)|[plugin.choria.network.websocket_port](#pluginchorianetworkwebsocket_port)|
|[plugin.choria.network.write_deadline](#pluginchorianetworkwrite_deadline)|[plugin.choria.prometheus_textfile_directory](#pluginchoriaprometheus_textfile_directory)|
|[plugin.choria.puppetca_host](#pluginchoriapuppetca_host)|[plugin.choria.puppetca_port](#pluginchoriapuppetca_port)|
|[plugin.choria.puppetdb_host](#pluginchoriapuppetdb_host)|[plugin.choria.puppetdb_port](#pluginchoriapuppetdb_port)|
|[plugin.choria.puppetserver_host](#pluginchoriapuppetserver_host)|[plugin.choria.puppetserver_port](#pluginchoriapuppetserver_port)|
|[plugin.choria.registration.file_content.compression](#pluginchoriaregistrationfile_contentcompression)|[plugin.choria.registration.file_content.data](#pluginchoriaregistrationfile_contentdata)|
|[plugin.choria.registration.file_content.target](#pluginchoriaregistrationfile_contenttarget)|[plugin.choria.registration.inventory_content.compression](#pluginchoriaregistrationinventory_contentcompression)|
|[plugin.choria.registration.inventory_content.target](#pluginchoriaregistrationinventory_contenttarget)|[plugin.choria.require_client_filter](#pluginchoriarequire_client_filter)|
|[plugin.choria.security.certname_whitelist](#pluginchoriasecuritycertname_whitelist)|[plugin.choria.security.privileged_users](#pluginchoriasecurityprivileged_users)|
|[plugin.choria.security.request_signer.seed_file](#pluginchoriasecurityrequest_signerseed_file)|[plugin.choria.security.request_signer.service](#pluginchoriasecurityrequest_signerservice)|
|[plugin.choria.security.request_signer.token_file](#pluginchoriasecurityrequest_signertoken_file)|[plugin.choria.security.request_signer.url](#pluginchoriasecurityrequest_signerurl)|
|[plugin.choria.security.server.seed_file](#pluginchoriasecurityserverseed_file)|[plugin.choria.security.server.token_file](#pluginchoriasecurityservertoken_file)|
|[plugin.choria.server.provision](#pluginchoriaserverprovision)|[plugin.choria.server.provision.allow_update](#pluginchoriaserverprovisionallow_update)|
|[plugin.choria.services.registry.cache](#pluginchoriaservicesregistrycache)|[plugin.choria.services.registry.store](#pluginchoriaservicesregistrystore)|
|[plugin.choria.srv_domain](#pluginchoriasrv_domain)|[plugin.choria.ssldir](#pluginchoriassldir)|
|[plugin.choria.stats_address](#pluginchoriastats_address)|[plugin.choria.stats_port](#pluginchoriastats_port)|
|[plugin.choria.status_file_path](#pluginchoriastatus_file_path)|[plugin.choria.status_update_interval](#pluginchoriastatus_update_interval)|
|[plugin.choria.submission.max_spool_size](#pluginchoriasubmissionmax_spool_size)|[plugin.choria.submission.spool](#pluginchoriasubmissionspool)|
|[plugin.choria.use_srv](#pluginchoriause_srv)|[plugin.login.aaasvc.login.url](#pluginloginaaasvcloginurl)|
|[plugin.nats.credentials](#pluginnatscredentials)|[plugin.nats.pass](#pluginnatspass)|
|[plugin.nats.user](#pluginnatsuser)|[plugin.scout.agent_disabled](#pluginscoutagent_disabled)|
|[plugin.scout.goss.denied_local_resources](#pluginscoutgossdenied_local_resources)|[plugin.scout.goss.denied_remote_resources](#pluginscoutgossdenied_remote_resources)|
|[plugin.scout.overrides](#pluginscoutoverrides)|[plugin.scout.tags](#pluginscouttags)|
|[plugin.security.certmanager.alt_names](#pluginsecuritycertmanageralt_names)|[plugin.security.certmanager.api_version](#pluginsecuritycertmanagerapi_version)|
|[plugin.security.certmanager.issuer](#pluginsecuritycertmanagerissuer)|[plugin.security.certmanager.namespace](#pluginsecuritycertmanagernamespace)|
|[plugin.security.certmanager.replace](#pluginsecuritycertmanagerreplace)|[plugin.security.choria.ca](#pluginsecuritychoriaca)|
|[plugin.security.choria.certificate](#pluginsecuritychoriacertificate)|[plugin.security.choria.key](#pluginsecuritychoriakey)|
|[plugin.security.choria.seed_file](#pluginsecuritychoriaseed_file)|[plugin.security.choria.sign_replies](#pluginsecuritychoriasign_replies)|
|[plugin.security.choria.token_file](#pluginsecuritychoriatoken_file)|[plugin.security.choria.trusted_signers](#pluginsecuritychoriatrusted_signers)|
|[plugin.security.cipher_suites](#pluginsecuritycipher_suites)|[plugin.security.client_anon_tls](#pluginsecurityclient_anon_tls)|
|[plugin.security.ecc_curves](#pluginsecurityecc_curves)|[plugin.security.file.ca](#pluginsecurityfileca)|
|[plugin.security.file.certificate](#pluginsecurityfilecertificate)|[plugin.security.file.key](#pluginsecurityfilekey)|
|[plugin.security.issuer.names](#pluginsecurityissuernames)|[plugin.security.pkcs11.driver_file](#pluginsecuritypkcs11driver_file)|
|[plugin.security.pkcs11.slot](#pluginsecuritypkcs11slot)|[plugin.security.provider](#pluginsecurityprovider)|
|[plugin.security.server_anon_tls](#pluginsecurityserver_anon_tls)|[plugin.security.support_legacy_certificates](#pluginsecuritysupport_legacy_certificates)|
|[plugin.yaml](#pluginyaml)|[registerinterval](#registerinterval)|
|[registration](#registration)|[registration_collective](#registration_collective)|
|[registration_splay](#registration_splay)|[rpcaudit](#rpcaudit)|
|[rpcauthorization](#rpcauthorization)|[rpcauthprovider](#rpcauthprovider)|
|[rpclimitmethod](#rpclimitmethod)|[soft_shutdown_timeout](#soft_shutdown_timeout)|
|[ttl](#ttl)|[](#)|


### classesfile

 * **Type:** path_string
 * **Default Value:** /opt/puppetlabs/puppet/cache/state/classes.txt

Path to a file listing configuration classes applied to a node, used in matches using Class filters

### collectives

 * **Type:** comma_split

The list of known Sub Collectives this node will join or communicate with, Servers will subscribe the node and each agent to each sub collective and Clients will publish to a chosen sub collective. Defaults to the build settin build.DefaultCollectives

### color

 * **Type:** boolean
 * **Default Value:** true

Disables or enable CLI color

### default_discovery_method

 * **Type:** string
 * **Validation:** enum=mc,broadcast,puppetdb,choria,external,inventory
 * **Default Value:** mc

The default discovery plugin to use. The default "mc" uses a network broadcast, "choria" uses PuppetDB, external calls external commands

### default_discovery_options

 * **Type:** strings

Default options to pass to the discovery plugin

### discovery_timeout

 * **Type:** integer
 * **Default Value:** 2

How long to wait for responses while doing broadcast discovery

### identity

 * **Type:** string

The identity this machine is known as, when empty it's derived based on the operating system hostname or by calling facter fqdn

### libdir

 * **Type:** path_split

The directory where Agents, DDLs and other plugins are found

### logfile

 * **Type:** path_string
 * **Default Value:** stdout

The file to write logs to, when set to 'discard' logging will be disabled. Also supports 'stdout' and 'stderr' as special log destinations.

### loglevel

 * **Type:** string
 * **Validation:** enum=debug,info,warn,error,fatal
 * **Default Value:** info

The lowest level log to add to the logfile

### main_collective

 * **Type:** string

The Sub Collective where a Client will publish to when no specific Sub Collective is configured

### plugin.choria.adapters

 * **Type:** comma_split
 * **Additional Information:** https://choria.io/docs/adapters/

The list of Data Adapters to activate

### plugin.choria.agent_provider.mcorpc.agent_shim

 * **Type:** string

Path to the helper used to call MCollective Ruby agents

### plugin.choria.agent_provider.mcorpc.config

 * **Type:** string

Path to the MCollective configuration file used when running MCollective Ruby agents

### plugin.choria.agent_provider.mcorpc.libdir

 * **Type:** path_split

Path to the libdir MCollective Ruby agents should have

### plugin.choria.broker_federation

 * **Type:** boolean
 * **Additional Information:** https://choria.io/docs/federation/
 * **Default Value:** false

Enables the Federation Broker

### plugin.choria.broker_network

 * **Type:** boolean
 * **Additional Information:** https://choria.io/docs/deployment/broker/
 * **Default Value:** false

Enables the Network Broker

### plugin.choria.discovery.broadcast.windowed_timeout

 * **Type:** boolean

Enables the experimental dynamic timeout for choria/mc discovery

### plugin.choria.discovery.external.command

 * **Type:** path_string

The command to use for external discovery

### plugin.choria.discovery.inventory.source

 * **Type:** path_string

The file to read for inventory discovery

### plugin.choria.federation.cluster

 * **Type:** string
 * **Additional Information:** https://choria.io/docs/federation/
 * **Default Value:** mcollective

The cluster name a Federation Broker serves

### plugin.choria.federation.collectives

 * **Type:** comma_split
 * **Additional Information:** https://choria.io/docs/federation/
 * **Environment Variable:** CHORIA_FED_COLLECTIVE

List of known remote collectives accessible via Federation Brokers

### plugin.choria.federation_middleware_hosts

 * **Type:** comma_split
 * **Additional Information:** https://choria.io/docs/federation/

Middleware brokers used by the Federation Broker, if unset uses SRV

### plugin.choria.legacy_lifecycle_format

 * **Type:** boolean
 * **Default Value:** 0

When enabled will publish lifecycle events in the legacy format, else Cloud Events format is used

### plugin.choria.machine.signing_key

 * **Type:** string

Public key used to sign data for watchers like machines watcher. Will override the value compiled in or in the watcher definitions if set here. This is primarily to allow development environments to use different private keys.

### plugin.choria.machine.store

 * **Type:** string
 * **Additional Information:** https://choria.io/docs/autoagents/

Directory where Autonomous Agents are stored

### plugin.choria.middleware_hosts

 * **Type:** comma_split

Set specific middleware hosts in the format host:port, if unset uses SRV

### plugin.choria.network.client_hosts

 * **Type:** comma_split

CIDRs to limit client connections from, appropriate ACLs are added based on this

### plugin.choria.network.client_port

 * **Type:** integer
 * **Additional Information:** https://choria.io/docs/deployment/broker/
 * **Default Value:** 4222

Port the Network Broker will accept client connections on

### plugin.choria.network.client_signer_cert

 * **Type:** comma_split

Fully qualified paths to the public certificates used by the AAA Service to sign client JWT tokens. This enables users with signed JWTs to use unverified TLS to connect. Can also be a list of ed25519 public keys.

### plugin.choria.network.client_tls_force_required

 * **Type:** boolean

Force requiring/not requiring TLS for all clients

### plugin.choria.network.deny_server_connections

 * **Type:** boolean

Set ACLs denying server connections to this broker

### plugin.choria.network.gateway_name

 * **Type:** string
 * **Default Value:** CHORIA

Name for the Super Cluster

### plugin.choria.network.gateway_port

 * **Type:** integer
 * **Default Value:** 0

Port to listen on for Super Cluster connections

### plugin.choria.network.gateway_remotes

 * **Type:** comma_split

List of remote Super Clusters to connect to

### plugin.choria.network.leafnode_port

 * **Type:** integer
 * **Default Value:** 0

Port to listen on for Leafnode connections, disabled with 0

### plugin.choria.network.leafnode_remotes

 * **Type:** comma_split

Remote networks to connect to as a Leafnode

### plugin.choria.network.listen_address

 * **Type:** string
 * **Additional Information:** https://choria.io/docs/deployment/broker/
 * **Default Value:** ::

Address the Network Broker will listen on

### plugin.choria.network.mapping.names

 * **Type:** comma_split

List of subject remappings to apply

### plugin.choria.network.peer_password

 * **Type:** string

Password to use when connecting to cluster peers

### plugin.choria.network.peer_port

 * **Type:** integer
 * **Additional Information:** https://choria.io/docs/deployment/broker/

Port used to communicate with other local cluster peers

### plugin.choria.network.peer_user

 * **Type:** string

Username to use when connecting to cluster peers

### plugin.choria.network.peers

 * **Type:** comma_split
 * **Additional Information:** https://choria.io/docs/deployment/broker/

List of cluster peers in host:port format

### plugin.choria.network.pprof_port

 * **Type:** integer
 * **Default Value:** 0

The port the network broker will listen on for pprof requests

### plugin.choria.network.provisioning.client_password

 * **Type:** string

Password the provisioned clients should use to connect

### plugin.choria.network.provisioning.provisioner_without_token

 * **Type:** boolean

Allows a provisioner without a token to connect over TLS using username and password.  This facilitates v1 provisioning on an Issuer based network

### plugin.choria.network.provisioning.signer_cert

 * **Type:** path_string

Path to the public cert that signs provisioning tokens, enables accepting provisioning connections into the provisioning account

### plugin.choria.network.public_url

 * **Type:** string

Name:Port to advertise to clients, useful when fronted by a proxy

### plugin.choria.network.server_signer_cert

 * **Type:** comma_split

Fully qualified Paths to the public certificates used by the Provisioner Service to sign server JWT tokens. This enables servers with signed JWTs to use unverified TLS to connect. Can also be a list of ed25519 public keys.

### plugin.choria.network.stream.advisory_replicas

 * **Type:** integer
 * **Default Value:** -1

When configuring Stream advisories storage ensure data is replicated in the cluster over this many servers, -1 means count of peers

### plugin.choria.network.stream.advisory_retention

 * **Type:** duration
 * **Default Value:** 168h

When not zero enables retaining Stream advisories in the Stream Store

### plugin.choria.network.stream.event_replicas

 * **Type:** integer
 * **Default Value:** -1

When configuring LifeCycle events ensure data is replicated in the cluster over this many servers, -1 means count of peers

### plugin.choria.network.stream.event_retention

 * **Type:** duration
 * **Default Value:** 24h

When not zero enables retaining Lifecycle events in the Stream Store

### plugin.choria.network.stream.leader_election_replicas

 * **Type:** integer
 * **Default Value:** -1

When configuring Stream based Leader Election storage ensure data is replicated in the cluster over this many servers, -1 means count of peers

### plugin.choria.network.stream.leader_election_ttl

 * **Type:** duration
 * **Default Value:** 1m

The TTL for leader election, leaders must vote at least this frequently to remain leader

### plugin.choria.network.stream.machine_replicas

 * **Type:** integer
 * **Default Value:** -1

When configuring Autonomous Agent event storage ensure data is replicated in the cluster over this many servers, -1 means count of peers

### plugin.choria.network.stream.machine_retention

 * **Type:** duration
 * **Default Value:** 24h

When not zero enables retaining Autonomous Agent events in the Stream Store

### plugin.choria.network.stream.manage_streams

 * **Type:** boolean
 * **Default Value:** 1

When set to zero will disable managing the standard streams on this node

### plugin.choria.network.stream.store

 * **Type:** path_string

Enables Streaming data persistence stored in this path

### plugin.choria.network.system.password

 * **Type:** string

Password used to access the Choria system account

### plugin.choria.network.system.user

 * **Type:** string

Username used to access the Choria system account

### plugin.choria.network.tls_timeout

 * **Type:** integer
 * **Default Value:** 2

Time to allow for TLS connections to establish, increase on slow or very large networks

### plugin.choria.network.websocket_advertise

 * **Type:** string
 * **Additional Information:** https://choria.io/docs/deployment/broker/

The URL to advertise for websocket connections

### plugin.choria.network.websocket_port

 * **Type:** integer
 * **Additional Information:** https://choria.io/docs/deployment/broker/

Port to listen on for websocket connections

### plugin.choria.network.write_deadline

 * **Type:** duration
 * **Default Value:** 10s

How long to allow clients to process traffic before treating them as slow, increase this on large networks or slow networks

### plugin.choria.prometheus_textfile_directory

 * **Type:** path_string

Directory where Prometheus Node Exporter textfile collector reads data

### plugin.choria.puppetca_host

 * **Type:** string
 * **Default Value:** puppet

The hostname where your Puppet Certificate Authority can be found

### plugin.choria.puppetca_port

 * **Type:** integer
 * **Default Value:** 8140

The port your Puppet Certificate Authority listens on

### plugin.choria.puppetdb_host

 * **Type:** string

The host hosting your PuppetDB, used by the "choria" discovery plugin

### plugin.choria.puppetdb_port

 * **Type:** integer
 * **Default Value:** 8081

The port your PuppetDB listens on

### plugin.choria.puppetserver_host

 * **Type:** string
 * **Default Value:** puppet

The hostname where your Puppet Server can be found

### plugin.choria.puppetserver_port

 * **Type:** integer
 * **Default Value:** 8140

The port your Puppet Server listens on

### plugin.choria.registration.file_content.compression

 * **Type:** boolean
 * **Default Value:** true

Enables gzip compression of registration data

### plugin.choria.registration.file_content.data

 * **Type:** string

YAML or JSON file to use as data source for registration

### plugin.choria.registration.file_content.target

 * **Type:** string

NATS Subject to publish registration data to

### plugin.choria.registration.inventory_content.compression

 * **Type:** boolean
 * **Default Value:** true

Enables gzip compression of registration data

### plugin.choria.registration.inventory_content.target

 * **Type:** string

NATS Subject to publish registration data to

### plugin.choria.require_client_filter

 * **Type:** boolean
 * **Default Value:** false

If a client filter should always be required, only used in Go clients

### plugin.choria.security.certname_whitelist

 * **Type:** comma_split
 * **Default Value:** \.mcollective$,\.choria$

Patterns of certificate names that are allowed to be clients

### plugin.choria.security.privileged_users

 * **Type:** comma_split
 * **Additional Information:** https://choria.io/docs/configuration/aaa/
 * **Default Value:** \.privileged.mcollective$,\.privileged.choria$

Patterns of certificate names that would be considered privileged and able to set custom callers

### plugin.choria.security.request_signer.seed_file

 * **Type:** path_string
 * **Additional Information:** https://github.com/choria-io/aaasvc

Path to the seed file used to access a Central Authenticator

### plugin.choria.security.request_signer.service

 * **Type:** boolean
 * **Additional Information:** https://choria-io.github.io/aaasvc/

Enables signing requests via Choria RPC requests

### plugin.choria.security.request_signer.token_file

 * **Type:** path_string
 * **Additional Information:** https://github.com/choria-io/aaasvc

Path to the token used to access a Central Authenticator

### plugin.choria.security.request_signer.url

 * **Type:** string
 * **Additional Information:** https://choria-io.github.io/aaasvc/

URL to the Signing Service

### plugin.choria.security.server.seed_file

 * **Type:** path_string

The server token seed to use for authentication, defaults to server.seed in the same location as server.conf

### plugin.choria.security.server.token_file

 * **Type:** path_string

The server token file to use for authentication, defaults to serer.jwt in the same location as server.conf

### plugin.choria.server.provision

 * **Type:** boolean
 * **Additional Information:** https://choria-io.github.io/provisioner/
 * **Default Value:** false

Specifically enable or disable provisioning

### plugin.choria.server.provision.allow_update

 * **Type:** boolean
 * **Additional Information:** https://choria-io.github.io/provisioner/
 * **Default Value:** false

Allows the provisioner to perform in-place version updates

### plugin.choria.services.registry.cache

 * **Type:** path_string
 * **Environment Variable:** CHORIA_REGISTRY

Directory where the Registry client stores DDLs found in the registry

### plugin.choria.services.registry.store

 * **Type:** path_string

Directory where the Registry service finds DDLs to read

### plugin.choria.srv_domain

 * **Type:** string
 * **Additional Information:** https://choria.io/docs/deployment/dns/

The domain to use for SRV records, defaults to the domain the server FQDN is in

### plugin.choria.ssldir

 * **Type:** path_string

The SSL directory, auto detected via Puppet, when specifically set Puppet will not be consulted

### plugin.choria.stats_address

 * **Type:** string
 * **Default Value:** 127.0.0.1

The address to listen on for statistics

### plugin.choria.stats_port

 * **Type:** integer
 * **Default Value:** 0

The port to listen on for HTTP requests for statistics, setting to 0 disables it

### plugin.choria.status_file_path

 * **Type:** path_string

Path to a JSON file to write server health information to regularly

### plugin.choria.status_update_interval

 * **Type:** integer
 * **Default Value:** 30

How frequently to write to the status_file_path

### plugin.choria.submission.max_spool_size

 * **Type:** integer
 * **Default Value:** 500

Maximum amount of messages allowed into each priority

### plugin.choria.submission.spool

 * **Type:** path_string

Path to a directory holding messages to submit to the middleware

### plugin.choria.use_srv

 * **Type:** boolean
 * **Additional Information:** https://choria.io/docs/deployment/dns/
 * **Default Value:** true

If SRV record lookups should be attempted to find Puppet, PuppetDB, Brokers etc

### plugin.login.aaasvc.login.url

 * **Type:** comma_split
 * **Additional Information:** https://choria-io.github.io/aaasvc/

List of URLs to attempt to login against when the remote signer is enabled

### plugin.nats.credentials

 * **Type:** string
 * **Environment Variable:** MCOLLECTIVE_NATS_CREDENTIALS

The NATS 2.0 credentials to use, required for accessing NGS

### plugin.nats.pass

 * **Type:** string
 * **Environment Variable:** MCOLLECTIVE_NATS_PASSWORD

The password to use when connecting to the NATS server

### plugin.nats.user

 * **Type:** string
 * **Environment Variable:** MCOLLECTIVE_NATS_USERNAME

The user to connect to the NATS server as. When unset no username is used.

### plugin.scout.agent_disabled

 * **Type:** boolean

Disables the scout agent

### plugin.scout.goss.denied_local_resources

 * **Type:** comma_split

List of resource types to deny for Goss manifests loaded from local disk

### plugin.scout.goss.denied_remote_resources

 * **Type:** comma_split
 * **Default Value:** command

List of resource types to deny when Goss manifests or variables were received over rpc

### plugin.scout.overrides

 * **Type:** path_string

Path to a file holding overrides for Scout checks

### plugin.scout.tags

 * **Type:** path_string

Path to a file holding tags for a Scout entity

### plugin.security.certmanager.alt_names

 * **Type:** comma_split

when using Cert Manager security provider, add these additional names to the CSR

### plugin.security.certmanager.api_version

 * **Type:** string
 * **Default Value:** v1

the API version to call in cert manager

### plugin.security.certmanager.issuer

 * **Type:** string

When using Cert Manager security provider, the name of the issuer

### plugin.security.certmanager.namespace

 * **Type:** string
 * **Default Value:** choria

When using Cert Manager security provider, the namespace the issuer is in

### plugin.security.certmanager.replace

 * **Type:** boolean
 * **Default Value:** true

when using Cert Manager security provider, replace existing CSRs with new ones

### plugin.security.choria.ca

 * **Type:** path_string

When using choria security provider, the path to the optional Certificate Authority public certificate

### plugin.security.choria.certificate

 * **Type:** path_string

When using choria security provider, the path to the optional public certificate

### plugin.security.choria.key

 * **Type:** path_string

When using choria security provider, the path to the optional private key

### plugin.security.choria.seed_file

 * **Type:** path_string

The path to the seed file

### plugin.security.choria.sign_replies

 * **Type:** boolean
 * **Default Value:** true

Disables signing replies which would significantly trim down the size of replies but would remove the ability to verify signatures or verify message origin

### plugin.security.choria.token_file

 * **Type:** path_string

The path to the JWT token file

### plugin.security.choria.trusted_signers

 * **Type:** comma_split

Ed25119 public keys of entities allowed to sign client and server JWT tokens in hex encoded format

### plugin.security.cipher_suites

 * **Type:** comma_split

List of allowed cipher suites

### plugin.security.client_anon_tls

 * **Type:** boolean
 * **Default Value:** false

Use anonymous TLS to the Choria brokers from a client, also disables security provider verification - only when a remote signer is set

### plugin.security.ecc_curves

 * **Type:** comma_split

List of allowed ECC curves

### plugin.security.file.ca

 * **Type:** path_string

When using file security provider, the path to the Certificate Authority public certificate

### plugin.security.file.certificate

 * **Type:** path_string

When using file security provider, the path to the public certificate

### plugin.security.file.key

 * **Type:** path_string

When using file security provider, the path to the private key

### plugin.security.issuer.names

 * **Type:** comma_split

List of names of valid issuers this server will accept, set indvidiaul issuer data using plugin.security.issuer.<name>.public

### plugin.security.pkcs11.driver_file

 * **Type:** path_string
 * **Additional Information:** https://choria.io/blog/post/2019/09/09/pkcs11/

When using the pkcs11 security provider, the path to the PCS11 driver file

### plugin.security.pkcs11.slot

 * **Type:** integer
 * **Additional Information:** https://choria.io/blog/post/2019/09/09/pkcs11/

When using the pkcs11 security provider, the slot to use in the device

### plugin.security.provider

 * **Type:** string
 * **Validation:** enum=puppet,file,pkcs11,certmanager,choria
 * **Default Value:** puppet

The Security Provider to use

### plugin.security.server_anon_tls

 * **Type:** boolean
 * **Default Value:** false

Use anonymous TLS to the Choria brokers from a server

### plugin.security.support_legacy_certificates

 * **Type:** boolean
 * **Default Value:** false

Allow certificates without SANs to be used

### plugin.yaml

 * **Type:** path_string
 * **Default Value:** /etc/puppetlabs/mcollective/generated-facts.yaml

Where to look for YAML or JSON based facts

### registerinterval

 * **Type:** integer
 * **Default Value:** 300

How often to publish registration data

### registration

 * **Type:** comma_split

The plugins used when publishing Registration data, when this is unset or empty sending registration data is disabled

### registration_collective

 * **Type:** string

The Sub Collective to publish registration data to

### registration_splay

 * **Type:** boolean
 * **Default Value:** true

When true delays initial registration publish by a random period up to registerinterval following registration publishes will be at registerinterval without further splay

### rpcaudit

 * **Type:** boolean
 * **Additional Information:** https://choria.io/docs/configuration/aaa/
 * **Default Value:** false

When enabled uses rpcauditprovider to audit RPC requests processed by the server

### rpcauthorization

 * **Type:** boolean
 * **Additional Information:** https://choria.io/docs/configuration/aaa/
 * **Default Value:** true

When enables authorization is performed on every RPC request based on rpcauthprovider

### rpcauthprovider

 * **Type:** title_string
 * **Additional Information:** https://choria.io/docs/configuration/aaa/
 * **Default Value:** action_policy

The Authorization system to use

### rpclimitmethod

 * **Type:** string
 * **Validation:** enum=first,random
 * **Default Value:** first

When limiting nodes to a subset of discovered nodes this is the method to use, random is influenced by

### soft_shutdown_timeout

 * **Type:** integer
 * **Default Value:** 2

The amount of time to allow the server to exit, after this memory and thread dumps will be performed and a force exit will be done

### ttl

 * **Type:** integer
 * **Default Value:** 60

How long published messages are allowed to linger on the network, lower numbers have a higher reliance on clocks being in sync

