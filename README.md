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
* Supports Clustering using a simple comma separated list of peers - TLS by default
* Exports statistics using the popular Prometheus format via the normal Choria statistics port

## Configuration

The broker is configured using the Choria daemon configuration, below a reference of the settings it supports.

|Setting|Description|Default|
|-------|-----------|-------|
|`plugin.choria.network.listen_address`|The network address to listen on|`::`|
|`plugin.choria.network.client_port`|The port to listen on for network clients|`4222`|
|`plugin.choria.network.peer_port`|The port to listen on for broker cluster peers|`5222`|
|`plugin.choria.network.peer_user`|Username to connect to cluster peers with|unset|
|`plugin.choria.network.peer_password`|Password to use when connecting to cluster peers|unset|
|`plugin.choria.network.peers`|Comma separated List of cluster peers to connect to|unset|
|`plugin.choria.network.write_deadline`|The time to allow for writes to network clients to complete before considering them slow|5s|
|`plugin.choria.network.client_hosts`|List of hosts - ip addresses or cidrs - that are allowed to use clients|all|

Choria core settings that affect the broker:

|Setting|Description|
|-------|-----------|
|`plugin.choria.broker_network`|Enables the network broker when running `choria broker run`|
|`loglevel`|The logging level to use|
|`plugin.choria.stats_port`|The port Choria listens on for metrics, when >0 the broker enables statistics|
|`plugin.choria.stats_address`|The network address to listen on for metrics requests|

It also uses the `build.maxBrokerClients` build time configuration in Choria to configure it's maximum connection limit, this defaults to 50 000.

## Statistics

When Statistics are enabled in Choria by setting `plugin.choria.stats_port` to nonzero the Choria Broker expose the following Prometheus statistics:

|Statistic|Description|
|---------|-----------|
|`choria_network_connections`|Current connections on the network broker|
|`choria_network_total_connections`|Total connections received since start|
|`choria_network_routes`|Current active routes to other brokers|
|`choria_network_remotes`|Current active connections to other brokers|
|`choria_network_in_msgs`|Messages received by the network broker|
|`choria_network_out_msgs`|Messages sent by the network broker|
|`choria_network_in_bytes`|Total size of messages received by the network broker|
|`choria_network_out_bytes`|Total size of messages sent by the network broker|
|`choria_network_slow_consumers`|Total number of clients who were considered slow consumers|
|`choria_network_subscriptions`|Number of active subscriptions to subjects on this broker|