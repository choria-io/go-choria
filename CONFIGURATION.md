# Choria Configuration Settings

This is a list of all known Configuration settings. This list is based on declared settings within the Choria Go code base and so will not cover 100% of settings - plugins can contribute their own settings which are note known at compile time.

## Data Types

A few special types are defined, the rest map to standard Go types

|Type|Description|
|----|-----------|
|comma_split|A comma separated list of strings, possibly with spaces between|
|duration|A duration such as "1h", "300ms", "-1.5h" or "2h45m". Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h"|
|path_split|A list of paths split by a OS specific PATH separator|
|path_string|A path that can include "~" for the users home directory|
|strings|A space separated list of strings|
|title_string|A string that will be stored as a Title String|

## Index

| | |
|-|-|
|[activate_agents](#activate_agents)|[classesfile](#classesfile)|
|[collectives](#collectives)|[color](#color)|
|[connection_timeout](#connection_timeout)|[connector](#connector)|
|[daemonize](#daemonize)|[default_discovery_method](#default_discovery_method)|
|[default_discovery_options](#default_discovery_options)|[direct_addressing](#direct_addressing)|
|[direct_addressing_threshold](#direct_addressing_threshold)|[discovery_timeout](#discovery_timeout)|
|[fact_cache_time](#fact_cache_time)|[factsource](#factsource)|
|[identity](#identity)|[keeplogs](#keeplogs)|
|[libdir](#libdir)|[logfacility](#logfacility)|
|[logfile](#logfile)|[logger_type](#logger_type)|
|[loglevel](#loglevel)|[main_collective](#main_collective)|
|[max_log_size](#max_log_size)|[plugin.choria.adapters](#pluginchoriaadapters)|
|[plugin.choria.agent_provider.mcorpc.agent_shim](#pluginchoriaagent_providermcorpcagent_shim)|[plugin.choria.agent_provider.mcorpc.config](#pluginchoriaagent_providermcorpcconfig)|
|[plugin.choria.agent_provider.mcorpc.libdir](#pluginchoriaagent_providermcorpclibdir)|[plugin.choria.broker_discovery](#pluginchoriabroker_discovery)|
|[plugin.choria.broker_federation](#pluginchoriabroker_federation)|[plugin.choria.broker_network](#pluginchoriabroker_network)|
|[plugin.choria.discovery.external.command](#pluginchoriadiscoveryexternalcommand)|[plugin.choria.discovery.inventory.source](#pluginchoriadiscoveryinventorysource)|
|[plugin.choria.discovery_host](#pluginchoriadiscovery_host)|[plugin.choria.discovery_port](#pluginchoriadiscovery_port)|
|[plugin.choria.discovery_proxy](#pluginchoriadiscovery_proxy)|[plugin.choria.federation.cluster](#pluginchoriafederationcluster)|
|[plugin.choria.federation.collectives](#pluginchoriafederationcollectives)|[plugin.choria.federation_middleware_hosts](#pluginchoriafederation_middleware_hosts)|
|[plugin.choria.legacy_lifecycle_format](#pluginchorialegacy_lifecycle_format)|[plugin.choria.machine.store](#pluginchoriamachinestore)|
|[plugin.choria.middleware_hosts](#pluginchoriamiddleware_hosts)|[plugin.choria.network.client_anon_tls](#pluginchorianetworkclient_anon_tls)|
|[plugin.choria.network.client_hosts](#pluginchorianetworkclient_hosts)|[plugin.choria.network.client_port](#pluginchorianetworkclient_port)|
|[plugin.choria.network.client_tls_force_required](#pluginchorianetworkclient_tls_force_required)|[plugin.choria.network.deny_server_connections](#pluginchorianetworkdeny_server_connections)|
|[plugin.choria.network.gateway_name](#pluginchorianetworkgateway_name)|[plugin.choria.network.gateway_port](#pluginchorianetworkgateway_port)|
|[plugin.choria.network.gateway_remotes](#pluginchorianetworkgateway_remotes)|[plugin.choria.network.leafnode_port](#pluginchorianetworkleafnode_port)|
|[plugin.choria.network.leafnode_remotes](#pluginchorianetworkleafnode_remotes)|[plugin.choria.network.listen_address](#pluginchorianetworklisten_address)|
|[plugin.choria.network.operator_account](#pluginchorianetworkoperator_account)|[plugin.choria.network.peer_password](#pluginchorianetworkpeer_password)|
|[plugin.choria.network.peer_port](#pluginchorianetworkpeer_port)|[plugin.choria.network.peer_user](#pluginchorianetworkpeer_user)|
|[plugin.choria.network.peers](#pluginchorianetworkpeers)|[plugin.choria.network.public_url](#pluginchorianetworkpublic_url)|
|[plugin.choria.network.stream.advisory_retention](#pluginchorianetworkstreamadvisory_retention)|[plugin.choria.network.stream.event_retention](#pluginchorianetworkstreamevent_retention)|
|[plugin.choria.network.stream.machine_retention](#pluginchorianetworkstreammachine_retention)|[plugin.choria.network.stream.store](#pluginchorianetworkstreamstore)|
|[plugin.choria.network.system_account](#pluginchorianetworksystem_account)|[plugin.choria.network.tls_timeout](#pluginchorianetworktls_timeout)|
|[plugin.choria.network.write_deadline](#pluginchorianetworkwrite_deadline)|[plugin.choria.prometheus_textfile_directory](#pluginchoriaprometheus_textfile_directory)|
|[plugin.choria.puppetca_host](#pluginchoriapuppetca_host)|[plugin.choria.puppetca_port](#pluginchoriapuppetca_port)|
|[plugin.choria.puppetdb_host](#pluginchoriapuppetdb_host)|[plugin.choria.puppetdb_port](#pluginchoriapuppetdb_port)|
|[plugin.choria.puppetserver_host](#pluginchoriapuppetserver_host)|[plugin.choria.puppetserver_port](#pluginchoriapuppetserver_port)|
|[plugin.choria.randomize_middleware_hosts](#pluginchoriarandomize_middleware_hosts)|[plugin.choria.registration.file_content.compression](#pluginchoriaregistrationfile_contentcompression)|
|[plugin.choria.registration.file_content.data](#pluginchoriaregistrationfile_contentdata)|[plugin.choria.registration.file_content.target](#pluginchoriaregistrationfile_contenttarget)|
|[plugin.choria.require_client_filter](#pluginchoriarequire_client_filter)|[plugin.choria.security.certname_whitelist](#pluginchoriasecuritycertname_whitelist)|
|[plugin.choria.security.privileged_users](#pluginchoriasecurityprivileged_users)|[plugin.choria.security.request_signer.token_environment](#pluginchoriasecurityrequest_signertoken_environment)|
|[plugin.choria.security.request_signer.token_file](#pluginchoriasecurityrequest_signertoken_file)|[plugin.choria.security.request_signer.url](#pluginchoriasecurityrequest_signerurl)|
|[plugin.choria.security.request_signing_certificate](#pluginchoriasecurityrequest_signing_certificate)|[plugin.choria.security.serializer](#pluginchoriasecurityserializer)|
|[plugin.choria.server.provision](#pluginchoriaserverprovision)|[plugin.choria.srv_domain](#pluginchoriasrv_domain)|
|[plugin.choria.ssldir](#pluginchoriassldir)|[plugin.choria.stats_address](#pluginchoriastats_address)|
|[plugin.choria.stats_port](#pluginchoriastats_port)|[plugin.choria.status_file_path](#pluginchoriastatus_file_path)|
|[plugin.choria.status_update_interval](#pluginchoriastatus_update_interval)|[plugin.choria.use_srv](#pluginchoriause_srv)|
|[plugin.nats.credentials](#pluginnatscredentials)|[plugin.nats.ngs](#pluginnatsngs)|
|[plugin.nats.pass](#pluginnatspass)|[plugin.nats.user](#pluginnatsuser)|
|[plugin.scout.agent_disabled](#pluginscoutagent_disabled)|[plugin.scout.overrides](#pluginscoutoverrides)|
|[plugin.scout.tags](#pluginscouttags)|[plugin.security.always_overwrite_cache](#pluginsecurityalways_overwrite_cache)|
|[plugin.security.certmanager.alt_names](#pluginsecuritycertmanageralt_names)|[plugin.security.certmanager.issuer](#pluginsecuritycertmanagerissuer)|
|[plugin.security.certmanager.namespace](#pluginsecuritycertmanagernamespace)|[plugin.security.certmanager.replace](#pluginsecuritycertmanagerreplace)|
|[plugin.security.cipher_suites](#pluginsecuritycipher_suites)|[plugin.security.client_anon_tls](#pluginsecurityclient_anon_tls)|
|[plugin.security.ecc_curves](#pluginsecurityecc_curves)|[plugin.security.file.ca](#pluginsecurityfileca)|
|[plugin.security.file.cache](#pluginsecurityfilecache)|[plugin.security.file.certificate](#pluginsecurityfilecertificate)|
|[plugin.security.file.key](#pluginsecurityfilekey)|[plugin.security.pkcs11.driver_file](#pluginsecuritypkcs11driver_file)|
|[plugin.security.pkcs11.slot](#pluginsecuritypkcs11slot)|[plugin.security.provider](#pluginsecurityprovider)|
|[plugin.security.support_legacy_certificates](#pluginsecuritysupport_legacy_certificates)|[plugin.yaml](#pluginyaml)|
|[publish_timeout](#publish_timeout)|[registerinterval](#registerinterval)|
|[registration](#registration)|[registration_collective](#registration_collective)|
|[registration_splay](#registration_splay)|[rpcaudit](#rpcaudit)|
|[rpcauditprovider](#rpcauditprovider)|[rpcauthorization](#rpcauthorization)|
|[rpcauthprovider](#rpcauthprovider)|[rpclimitmethod](#rpclimitmethod)|
|[securityprovider](#securityprovider)|[soft_shutdown](#soft_shutdown)|
|[soft_shutdown_timeout](#soft_shutdown_timeout)|[threaded](#threaded)|
|[ttl](#ttl)|[](#)|


## activate_agents

 * **Type:** boolean
 * **Default Value:** true

**This setting is deprecated or already unused**

## classesfile

 * **Type:** path_string
 * **Default Value:** /opt/puppetlabs/puppet/cache/state/classes.txt

Path to a file listing configuration classes applied to a node, used in matches using Class filters

## collectives

 * **Type:** comma_split
 * **Default Value:** mcollective

The list of known Sub Collectives this node will join or communicate with, Servers will subscribe the node and each agent to each sub collective and Clients will publish to a chosen sub collective

## color

 * **Type:** boolean
 * **Default Value:** true

Disables or enable CLI color

## connection_timeout

 * **Type:** integer

Ruby clients use this to determine how long they will try to connect, fails after timeout

## connector

 * **Type:** title_string
 * **Default Value:** nats

Configures the network connector to use, only sensible value is "nats", unused in Go based code

## daemonize

 * **Type:** boolean
 * **Default Value:** false

**This setting is deprecated or already unused**

## default_discovery_method

 * **Type:** string
 * **Validation:** enum=mc,broadcast,puppetdb,choria,external,inventory
 * **Default Value:** mc

The default discovery plugin to use. The default "mc" uses a network broadcast, "choria" uses PuppetDB, external calls external commands

## default_discovery_options

 * **Type:** strings

Default options to pass to the discovery plugin

## direct_addressing

 * **Type:** boolean
 * **Default Value:** true

Enables the direct-to-node communications pattern, unused in the Go clients

## direct_addressing_threshold

 * **Type:** integer
 * **Default Value:** 10

**This setting is deprecated or already unused**

## discovery_timeout

 * **Type:** integer
 * **Default Value:** 2

How long to wait for responses while doing broadcast discovery

## fact_cache_time

 * **Type:** integer
 * **Default Value:** 300

**This setting is deprecated or already unused**

## factsource

 * **Type:** string
 * **Default Value:** yaml

**This setting is deprecated or already unused**

## identity

 * **Type:** string

The identity this machine is known as, when empty it's derived based on the operating system hostname or by calling facter fqdn

## keeplogs

 * **Type:** integer
 * **Default Value:** 5

**This setting is deprecated or already unused**

## libdir

 * **Type:** path_split

The directory where Agents, DDLs and other plugins are found

## logfacility

 * **Type:** string
 * **Default Value:** user

**This setting is deprecated or already unused**

## logfile

 * **Type:** path_string

The file to write logs to, when set to an empty string logging will be to the console, when set to 'discard' logging will be disabled

## logger_type

 * **Type:** string
 * **Validation:** enum=console,file,syslog
 * **Default Value:** file

The type of logging to use, unused in Go based programs

## loglevel

 * **Type:** string
 * **Validation:** enum=debug,info,warn,error,fatal
 * **Default Value:** info

The lowest level log to add to the logfile

## main_collective

 * **Type:** string

The Sub Collective where a Client will publish to when no specific Sub Collective is configured

## max_log_size

 * **Type:** integer
 * **Default Value:** 2097152

**This setting is deprecated or already unused**

## plugin.choria.adapters

 * **Type:** comma_split
 * **Additional Information:** https://choria.io/docs/adapters/

The list of Data Adapters to activate

## plugin.choria.agent_provider.mcorpc.agent_shim

 * **Type:** string

Path to the helper used to call MCollective Ruby agents

## plugin.choria.agent_provider.mcorpc.config

 * **Type:** string

Path to the MCollective configuration file used when running MCollective Ruby agents

## plugin.choria.agent_provider.mcorpc.libdir

 * **Type:** path_split

Path to the libdir MCollective Ruby agents should have

## plugin.choria.broker_discovery

 * **Type:** boolean
 * **Default Value:** false

**This setting is deprecated or already unused**

## plugin.choria.broker_federation

 * **Type:** boolean
 * **Additional Information:** https://choria.io/docs/federation/
 * **Default Value:** false

Enables the Federation Broker

## plugin.choria.broker_network

 * **Type:** boolean
 * **Additional Information:** https://choria.io/docs/deployment/broker/
 * **Default Value:** false

Enables the Network Broker

## plugin.choria.discovery.external.command

 * **Type:** path_string

The command to use for external discovery

## plugin.choria.discovery.inventory.source

 * **Type:** path_string

The file to read for inventory discovery

## plugin.choria.discovery_host

 * **Type:** string
 * **Default Value:** puppet

discovery proxy

**This setting is deprecated or already unused**

## plugin.choria.discovery_port

 * **Type:** integer
 * **Default Value:** 8085

**This setting is deprecated or already unused**

## plugin.choria.discovery_proxy

 * **Type:** boolean
 * **Default Value:** false

**This setting is deprecated or already unused**

## plugin.choria.federation.cluster

 * **Type:** string
 * **Additional Information:** https://choria.io/docs/federation/
 * **Default Value:** mcollective

The cluster name a Federation Broker serves

## plugin.choria.federation.collectives

 * **Type:** comma_split
 * **Additional Information:** https://choria.io/docs/federation/
 * **Environment Variable:** CHORIA_FED_COLLECTIVE

List of known remote collectives accessible via Federation Brokers

## plugin.choria.federation_middleware_hosts

 * **Type:** comma_split
 * **Additional Information:** https://choria.io/docs/federation/

Middleware brokers used by the Federation Broker, if unset uses SRV

## plugin.choria.legacy_lifecycle_format

 * **Type:** boolean
 * **Default Value:** 0

When enabled will publish lifecycle events in the legacy format, else Cloud Events format is used

## plugin.choria.machine.store

 * **Type:** string
 * **Additional Information:** https://choria.io/docs/autoagents/

Directory where Autonomous Agents are stored

## plugin.choria.middleware_hosts

 * **Type:** comma_split

Set specific middleware hosts in the format host:port, if unset uses SRV

## plugin.choria.network.client_anon_tls

 * **Type:** boolean

Use anonymous TLS for client connections (disables verification)

## plugin.choria.network.client_hosts

 * **Type:** comma_split

CIDRs to limit client connections from, appropriate ACLs are added based on this

## plugin.choria.network.client_port

 * **Type:** integer
 * **Additional Information:** https://choria.io/docs/deployment/broker/
 * **Default Value:** 4222

Port the Network Broker will accept client connections on

## plugin.choria.network.client_tls_force_required

 * **Type:** boolean

Force requiring/not requiring TLS for all clients

## plugin.choria.network.deny_server_connections

 * **Type:** boolean

Set ACLs denying server connections to this broker

## plugin.choria.network.gateway_name

 * **Type:** string
 * **Default Value:** CHORIA

Name for the Super Cluster

## plugin.choria.network.gateway_port

 * **Type:** integer
 * **Default Value:** 0

Port to listen on for Super Cluster connections

## plugin.choria.network.gateway_remotes

 * **Type:** comma_split

List of remote Super Clusters to connect to

## plugin.choria.network.leafnode_port

 * **Type:** integer
 * **Default Value:** 0

Port to listen on for Leafnode connections, disabled with 0

## plugin.choria.network.leafnode_remotes

 * **Type:** comma_split

Remote networks to connect to as a Leafnode

## plugin.choria.network.listen_address

 * **Type:** string
 * **Additional Information:** https://choria.io/docs/deployment/broker/
 * **Default Value:** ::

Address the Network Broker will listen on

## plugin.choria.network.operator_account

 * **Type:** string

NATS 2.0 Operator account

## plugin.choria.network.peer_password

 * **Type:** string

Password to use when connecting to cluster peers

## plugin.choria.network.peer_port

 * **Type:** integer
 * **Additional Information:** https://choria.io/docs/deployment/broker/

Port used to communicate with other local cluster peers

## plugin.choria.network.peer_user

 * **Type:** string

Username to use when connecting to cluster peers

## plugin.choria.network.peers

 * **Type:** comma_split
 * **Additional Information:** https://choria.io/docs/deployment/broker/

List of cluster peers in host:port format

## plugin.choria.network.public_url

 * **Type:** string

Name to advertise to clients, useful when fronted by a proxy

## plugin.choria.network.stream.advisory_retention

 * **Type:** duration
 * **Default Value:** 168h

When not zero enables retaining Stream advisories in the Stream Store

## plugin.choria.network.stream.event_retention

 * **Type:** duration
 * **Default Value:** 24h

When not zero enables retaining Lifecycle events in the Stream Store

## plugin.choria.network.stream.machine_retention

 * **Type:** duration
 * **Default Value:** 24h

When not zero enables retaining Autonomous Agent events in the Stream Store

## plugin.choria.network.stream.store

 * **Type:** path_string

Enables Streaming data persistence stored in this path

## plugin.choria.network.system_account

 * **Type:** string

NATS 2.0 System Account

## plugin.choria.network.tls_timeout

 * **Type:** integer
 * **Default Value:** 2

Time to allow for TLS connections to establish, increase on slow or very large networks

## plugin.choria.network.write_deadline

 * **Type:** duration
 * **Default Value:** 10s

How long to allow clients to process traffic before treating them as slow, increase this on large networks or slow networks

## plugin.choria.prometheus_textfile_directory

 * **Type:** path_string

Directory where Prometheus Node Exporter textfile collector reads data

## plugin.choria.puppetca_host

 * **Type:** string
 * **Default Value:** puppet

The hostname where your Puppet Certificate Authority can be found

## plugin.choria.puppetca_port

 * **Type:** integer
 * **Default Value:** 8140

The port your Puppet Certificate Authority listens on

## plugin.choria.puppetdb_host

 * **Type:** string

The host hosting your PuppetDB, used by the "choria" discovery plugin

## plugin.choria.puppetdb_port

 * **Type:** integer
 * **Default Value:** 8081

The port your PuppetDB listens on

## plugin.choria.puppetserver_host

 * **Type:** string
 * **Default Value:** puppet

The hostname where your Puppet Server can be found

## plugin.choria.puppetserver_port

 * **Type:** integer
 * **Default Value:** 8140

The port your Puppet Server listens on

## plugin.choria.randomize_middleware_hosts

 * **Type:** boolean
 * **Default Value:** true

Shuffle middleware hosts before connecting to spread traffic of initial connections

## plugin.choria.registration.file_content.compression

 * **Type:** boolean
 * **Default Value:** true

Enables gzip compression of registration data

## plugin.choria.registration.file_content.data

 * **Type:** string

YAML or JSON file to use as data source for registration

## plugin.choria.registration.file_content.target

 * **Type:** string

NATS Subject to publish registration data to

## plugin.choria.require_client_filter

 * **Type:** boolean
 * **Default Value:** false

If a client filter should always be required, only used in Go clients

## plugin.choria.security.certname_whitelist

 * **Type:** comma_split
 * **Default Value:** \.mcollective$,\.choria$

Patterns of certificate names that are allowed to be clients

## plugin.choria.security.privileged_users

 * **Type:** comma_split
 * **Additional Information:** https://choria.io/docs/configuration/aaa/
 * **Default Value:** \.privileged.mcollective$,\.privileged.choria$

Patterns of certificate names that would be considered privileged and able to set custom callers

## plugin.choria.security.request_signer.token_environment

 * **Type:** string
 * **Additional Information:** https://github.com/choria-io/aaasvc

Environment variable to store Central Authenticator tokens

## plugin.choria.security.request_signer.token_file

 * **Type:** path_string
 * **Additional Information:** https://github.com/choria-io/aaasvc

Path to the token used to access a Central Authenticator

## plugin.choria.security.request_signer.url

 * **Type:** string
 * **Additional Information:** https://github.com/choria-io/aaasvc

URL to the Signing Service

## plugin.choria.security.request_signing_certificate

 * **Type:** string

The public certificate of the key used to sign the JWTs in the Signing Service

## plugin.choria.security.serializer

 * **Type:** string
 * **Validation:** enum=json,yaml
 * **Default Value:** json

**This setting is deprecated or already unused**

## plugin.choria.server.provision

 * **Type:** boolean
 * **Additional Information:** https://github.com/choria-io/provisioning-agent
 * **Default Value:** false

Specifically enable or disable provisioning

## plugin.choria.srv_domain

 * **Type:** string
 * **Additional Information:** https://choria.io/docs/deployment/dns/

The domain to use for SRV records, defaults to the domain the server FQDN is in

## plugin.choria.ssldir

 * **Type:** path_string

The SSL directory, auto detected via Puppet, when specifically set Puppet will not be consulted

## plugin.choria.stats_address

 * **Type:** string
 * **Default Value:** 127.0.0.1

The address to listen on for statistics

## plugin.choria.stats_port

 * **Type:** integer
 * **Default Value:** 0

The port to listen on for HTTP requests for statistics, setting to 0 disables it

## plugin.choria.status_file_path

 * **Type:** path_string

Path to a JSON file to write server health information to regularly

## plugin.choria.status_update_interval

 * **Type:** integer
 * **Default Value:** 30

How frequently to write to the status_file_path

## plugin.choria.use_srv

 * **Type:** boolean
 * **Additional Information:** https://choria.io/docs/deployment/dns/
 * **Default Value:** true

If SRV record lookups should be attempted to find Puppet, PuppetDB, Brokers etc

## plugin.nats.credentials

 * **Type:** string
 * **Environment Variable:** MCOLLECTIVE_NATS_CREDENTIALS

The NATS 2.0 credentials to use, required for accessing NGS

## plugin.nats.ngs

 * **Type:** boolean
 * **Environment Variable:** MCOLLECTIVE_NATS_NGS

Uses NATS NGS global managed network as middleware, overrides broker names to "connect.ngs.global"

## plugin.nats.pass

 * **Type:** string
 * **Environment Variable:** MCOLLECTIVE_NATS_PASSWORD

The password to use when connecting to the NATS server

## plugin.nats.user

 * **Type:** string
 * **Environment Variable:** MCOLLECTIVE_NATS_USERNAME

The user to connect to the NATS server as. When unset no username is used.

## plugin.scout.agent_disabled

 * **Type:** boolean

Disables the scout agent

## plugin.scout.overrides

 * **Type:** path_string

Path to a file holding overrides for Scout checks

## plugin.scout.tags

 * **Type:** path_string

Path to a file holding tags for a Scout entity

## plugin.security.always_overwrite_cache

 * **Type:** boolean
 * **Default Value:** false

Always store new Public Keys to the cache overwriting existing ones

## plugin.security.certmanager.alt_names

 * **Type:** comma_split

when using Cert Manager security provider, add these additional names to the CSR

## plugin.security.certmanager.issuer

 * **Type:** string

When using Cert Manager security provider, the name of the issuer

## plugin.security.certmanager.namespace

 * **Type:** string
 * **Default Value:** choria

When using Cert Manager security provider, the namespace the issuer is in

## plugin.security.certmanager.replace

 * **Type:** boolean
 * **Default Value:** true

when using Cert Manager security provider, replace existing CSRs with new ones

## plugin.security.cipher_suites

 * **Type:** comma_split

List of allowed cipher suites

## plugin.security.client_anon_tls

 * **Type:** boolean
 * **Default Value:** false

Use anonymous TLS to the Choria brokers from a client, also disables security provider verification - only when a remote signer is set

## plugin.security.ecc_curves

 * **Type:** comma_split

List of allowed ECC curves

## plugin.security.file.ca

 * **Type:** path_string

When using file security provider, the path to the Certificate Authority public certificate

## plugin.security.file.cache

 * **Type:** path_string

When using file security provider, the path to the client cache

## plugin.security.file.certificate

 * **Type:** path_string

When using file security provider, the path to the public certificate

## plugin.security.file.key

 * **Type:** path_string

When using file security provider, the path to the private key

## plugin.security.pkcs11.driver_file

 * **Type:** path_string
 * **Additional Information:** https://choria.io/blog/post/2019/09/09/pkcs11/

When using the pkcs11 security provider, the path to the PCS11 driver file

## plugin.security.pkcs11.slot

 * **Type:** integer
 * **Additional Information:** https://choria.io/blog/post/2019/09/09/pkcs11/

When using the pkcs11 security provider, the slot to use in the device

## plugin.security.provider

 * **Type:** string
 * **Validation:** enum=puppet,file,pkcs11,certmanager
 * **Default Value:** puppet

The Security Provider to use

## plugin.security.support_legacy_certificates

 * **Type:** boolean
 * **Default Value:** false

Allow certificates without SANs to be used

## plugin.yaml

 * **Type:** path_string
 * **Default Value:** /etc/puppetlabs/mcollective/generated-facts.yaml

Where to look for YAML or JSON based facts

## publish_timeout

 * **Type:** integer
 * **Default Value:** 2

Ruby clients use this to determine how long they will allow when publishing requests

## registerinterval

 * **Type:** integer
 * **Default Value:** 300

How often to publish registration data

## registration

 * **Type:** comma_split

The plugins used when publishing Registration data, when this is unset or empty sending registration data is disabled

## registration_collective

 * **Type:** string

The Sub Collective to publish registration data to

## registration_splay

 * **Type:** boolean
 * **Default Value:** false

When true delays initial registration publish by a random period up to registerinterval following registration publishes will be at registerinterval without further splay

## rpcaudit

 * **Type:** boolean
 * **Additional Information:** https://choria.io/docs/configuration/aaa/
 * **Default Value:** false

When enabled uses rpcauditprovider to audit RPC requests processed by the server

## rpcauditprovider

 * **Type:** title_string
 * **Additional Information:** https://choria.io/docs/configuration/aaa/

The audit provider to use, unused at present as there is only a "choria" one

## rpcauthorization

 * **Type:** boolean
 * **Additional Information:** https://choria.io/docs/configuration/aaa/
 * **Default Value:** false

When enables authorization is performed on every RPC request based on rpcauthprovider

## rpcauthprovider

 * **Type:** title_string
 * **Additional Information:** https://choria.io/docs/configuration/aaa/
 * **Default Value:** action_policy

The Authorization system to use

## rpclimitmethod

 * **Type:** string
 * **Validation:** enum=first,random
 * **Default Value:** first

When limiting nodes to a subset of discovered nodes this is the method to use, random is influenced by

## securityprovider

 * **Type:** title_string
 * **Default Value:** choria

Used to select the security provider in Ruby clients, only sensible value is "choria"

**This setting is deprecated or already unused**

## soft_shutdown

 * **Type:** boolean
 * **Default Value:** true

**This setting is deprecated or already unused**

## soft_shutdown_timeout

 * **Type:** integer
 * **Default Value:** 2

**This setting is deprecated or already unused**

## threaded

 * **Type:** boolean
 * **Default Value:** false

Enables multi threaded mode in the Ruby client, generally a bad idea

## ttl

 * **Type:** integer
 * **Default Value:** 60

How long published messages are allowed to linger on the network, lower numbers have a higher reliance on clocks being in sync

