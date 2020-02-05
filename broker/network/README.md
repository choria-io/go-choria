# Choria Network Broker

This is a [NATS](https://nats.io) compatible Network Broker for use by the Choria Orchestration System.

Please review the official documentation at [choria.io](https://choria.io) for installation and usage.

## Motivation

Running a middleware broker for Choria is quite the undertaking, while NATS is really easy to operate it does have a plethora of settings and using the wrong ones can adversely affect your network.

The Choria Network Broker is a managed NATS broker that integrates into the `choria broker` command.  It ships as part of the normal Choria package and exist within the single binary.

It sets up the NATS server in ways thats suitable for use by Choria with sane defaults enabled.

Features:

* Works by default without any broker specific configuration in your Choria broker
* Secure by default - only accepts TLS connections with certificates signed by the known CA
* Support NATS Accounts technology for large scale multi tenancy
* Supports Clustering using a simple comma separated list of peers - TLS by default
* Support Gateways enabling communication between NATS clusters - an alternative to Choria Federation
* Support Leafnodes enabling joining older or unauthenticated clients to a secure multi tenant network
* Exports statistics using the popular Prometheus format via the normal Choria statistics port

## Configuration

The broker is configured using the Choria daemon configuration, below a reference of the settings it supports.

### Choria core settings that affect the broker:

|Setting|Description|
|-------|-----------|
|`plugin.choria.broker_network`|Enables the network broker when running `choria broker run`|
|`loglevel`|The logging level to use|
|`plugin.choria.stats_port`|The port Choria listens on for metrics, when >0 the broker enables statistics|
|`plugin.choria.stats_address`|The network address to listen on for metrics requests|

It also uses the `build.maxBrokerClients` build time configuration in Choria to configure it's maximum connection limit, this defaults to 50 000.

### Basic Broker Settings

|Setting|Description|Default|
|-------|-----------|-------|
|`plugin.choria.network.listen_address`|The network address to listen on|`::`|
|`plugin.choria.network.client_port`|The port to listen on for network clients|`4222`|
|`plugin.choria.network.write_deadline`|The time to allow for writes to network clients to complete before considering them slow|5s|
|`plugin.choria.network.client_hosts`|List of hosts - ip addresses or cidrs - that are allowed to use clients|all|
|`plugin.choria.network.client_tls_force_required`|Force TLS on for client connections regardless of build settings|`false`|
|`plugin.choria.network.tls_timeout`|Sets the timeout for establishing TLS connections|`2`|

### Cluster Settings

Network Clusters are suitable for creating a cluster of up to 5 nodes on a local LAN. These form a full Mesh and provides scalability and HA.

They are based on NATS technology and you can read more about them [at NATS.io](https://nats-io.github.io/docs/nats_server/clustering.html)

|Setting|Description|Default|
|-------|-----------|-------|
|`plugin.choria.network.peer_port`|The port to listen on for broker cluster peers|`5222`|
|`plugin.choria.network.peer_user`|Username to connect to cluster peers with|unset|
|`plugin.choria.network.peer_password`|Password to use when connecting to cluster peers|unset|
|`plugin.choria.network.peers`|Comma separated List of cluster peers to connect to|unset|

### Gateway Settings

Gateways allow you to combine multiple Clusters into a single large cluster.  This allow you to span your collective across multiple data centers without the need for the much harder to configure federation brokers.

By default if the broker is compiled with TLS the Gateway will use the same TLS settings for the connection - you can customize it on a per remote basis.

They are based on NATS technology and you can read more about them [at NATS.io](https://nats-io.github.io/docs/gateways/)

|Setting|Description|Default|
|-------|-----------|-------|
|`plugin.choria.network.gateway_port`|The port to listen to for Gateway connections, disabled when 0|`0`|
|`plugin.choria.network.gateway_name`|Unique name for the cluster listening on the port|`CHORIA`|
|`plugin.choria.network.gateway_remotes`|A comma sep list of remote names to activate|`""`|
|`plugin.choria.network.gateway_remote.C1.urls`|A comma sep list of `host:port` combinations to connect to for the remote `C1` cluster||
|`plugin.choria.network.gateway_remote.C1.tls.cert`|Path to a custom certificate for this remote only||
|`plugin.choria.network.gateway_remote.C1.tls.key`|Path to a custom private key for this remote only||
|`plugin.choria.network.gateway_remote.C1.tls.ca`|Path to a custom ca for this remote only||
|`plugin.choria.network.gateway_remote.C1.tls.disable`|Disables the TLS configuration that would have inherited from the Choria Security system|`false`|
|`plugin.choria.network.gateway_remote.C1.tls.verify`|Disables full TLS verify for this remote only|`true`|

### Leafnode Settings

Leafnodes exist to take unauthenticated or unsecured connections and forge them into a specific Account (see below). They allow older Choria agents and clients to take part of a multi tenant or account secured network.

By default if the broker is compiled with TLS the leafnode will use the same TLS settings for the connection - you can customize it on a per remote basis.

They are based on NATS technology and you can read more about them [at NATS.io](https://nats-io.github.io/docs/leafnodes/)

|Setting|Description|Default|
|-------|-----------|-------|
|`plugin.choria.network.leafnode_port`|The port to listen to for Gateway connections, disabled when 0|`0`|
|`plugin.choria.network.leafnode_remotes`|A comma sep list of remote names to activate|`""`|
|`plugin.choria.network.leafnode_remote.C1.url`|A `host:port` combination to connect to for the remote `C1` leafnode||
|`plugin.choria.network.leafnode_remote.C1.account`|The local account name to use when connecting to the remote||
|`plugin.choria.network.leafnode_remote.C1.credential`|The local credential file to use when connecting to the remote||
|`plugin.choria.network.leafnode_remote.C1.tls.cert`|Path to a custom certificate for this remote only||
|`plugin.choria.network.leafnode_remote.C1.tls.key`|Path to a custom private key for this remote only||
|`plugin.choria.network.leafnode_remote.C1.tls.ca`|Path to a custom ca for this remote only||
|`plugin.choria.network.leafnode_remote.C1.tls.disable`|Disables the TLS configuration that would have inherited from the Choria Security system|`false`|
|`plugin.choria.network.leafnode_remote.C1.tls.verify`|Disables full TLS verify for this remote only|`true`|

### Accounts

Accounts are based on NATS technology, you can read more about them [at NATS.io](https://nats-io.github.io/docs/nats_server/accounts.html)

|Setting|Description|Default|
|-------|-----------|-------|
|`plugin.choria.network.operator_account`|The operator account that is managing this cluster||
|`plugin.choria.network.system_account`|The system account to use, when set enables server events||

## Statistics

When Statistics are enabled in Choria by setting `plugin.choria.stats_port` to nonzero the Choria Broker expose the following Prometheus statistics:

|Statistic|Description|
|---------|-----------|
|`choria_network_connections`|Current connections on the network broker|
|`choria_network_total_connections`|Total connections received since start|
|`choria_network_routes`|Current active routes to other brokers|
|`choria_network_remotes`|Current active connections to other brokers|
|`choria_network_leafnode_remotes`|Current active connections to leaf nodes|
|`choria_network_in_msgs`|Messages received by the network broker|
|`choria_network_out_msgs`|Messages sent by the network broker|
|`choria_network_in_bytes`|Total size of messages received by the network broker|
|`choria_network_out_bytes`|Total size of messages sent by the network broker|
|`choria_network_slow_consumers`|Total number of clients who were considered slow consumers|
|`choria_network_subscriptions`|Number of active subscriptions to subjects on this broker|