# Choria Broker and Server

Choria is a framework for building Control Planes, Orchestration Systems and Programmable Infrastructure.

This is a daemon and related tools written in Go that hosts services, autonomous agents and generally provide a secure hosting environment for callable logic that you can interact with from code.

More information about the project can be found on [Choria.IO](https://choria.io).

[![CodeFactor](https://www.codefactor.io/repository/github/choria-io/go-choria/badge)](https://www.codefactor.io/repository/github/choria-io/go-choria) [![CII Best Practices](https://bestpractices.coreinfrastructure.org/projects/3558/badge)](https://bestpractices.coreinfrastructure.org/projects/3558)

# Bundled tools

Various helpers and utilities are bundled with the Choria binary to assist in testing and debugging the Choria connection to the brokers and observe its general life cycle.

These will connect to the middleware using your usual client configuration.

|Command|Description|
|-------|-----------|
|`choria ping`|Basic network testing utility, like `mco ping` but fast and with extra options|
|`choria req`|Generic RPC request tool to interact with remove RPC Agents|
|`choria tool event`|Listens for Choria life cycle events emitted by various daemons and related tools|
|`choria tool sub`|Subscribes to any middleware topic|
|`choria tool pub`|Publishes to any middleware topic|
|`choria tool provision`|Tool to test provision target discovery|
|`choria tool status`|Parse the status file and check overall health|
|`choria tool generate`|Generates various related files like DDLs|
|`choria tool jwt`|Generates JWT tokens used to configure automatic provisioning|
|`choria tool config`|To view details about known configuration options|

# Configuration

This code base represents the current recommended Choria Broker and Federation and will soon also be the recommended Server component.
Follow [choria.io](https://choria.io) for the official means of installing and configuring it.

Sample configs are shown, subject to change

Running `choria broker run --config /path/to/broker.cfg` will start the various broker components, you can safely run the Middleware Broker,
Federation Broker and Protocol Adapter all in the same process.

The tool `choria tool config` can be used to list and view known configuration options - be aware though that individual agents might
use their own configuration - but this tool lists all known configuration keys.

## Metrics

When enabled a vast cache of Prometheus compatible metrics are exposed under `/choria/prometheus`, use `plugin.choria.stats_port` and `plugin.choria.stats_address` to enable

Additionally server status can be written regularly - 30 seconds interval by default:

```ini
plugin.choria.status_file_path = /var/tmp/choria_status.json
plugin.choria.status_update_interval = 30
```

This status file can be checked using `choria tool check` to ensure messages are received regularly, the server is connected to a broker and that the file is written regularly.  The purpose of this tool is to enable scripts, monitoring systems and more to have a standard way to parse this file.  Exit code will be non 0 when the server is not healthy.

```
$ choria tool status --status choria-status.json --message-since 10m --max-age 1h
choria-status.json no recent messages: last message at 2019-03-15 15:53:30 +0100 CET
$ echo $?
1
```

## Configurable Security Subsystems

Choria has 3 major security providers:

 * `puppet` - integrates with the Puppet Certificate Authority
 * `file` - configurable paths for certificate, key, ca and cache
 * `pkcs11` - [pkcs11](https://choria.io/blog/post/2019/09/09/pkcs11/) integration for hardware tokens

### General Settings

Generally each provider will have it's own settings, there are a few system wide ones:

|Setting|Default|Description|
|-------|-------|-----------|
|`plugin.security.provider`|`puppet`|The security provider to use, can be `puppet`, `file` or `pkcs11`|
|`plugin.security.always_overwrite_cache`|`false`|Tell the security provider to always overwrite the certificate cache, can be `true`, or `false`|
|`plugin.choria.security.privileged_users`|`\\.privileged.mcollective$`|Comma sep list of valid certificates for privileged client users.|
|`plugin.choria.security.certname_whitelist`|`\\.mcollective$`|Comma sep list of valid certificates for normal client users.|

### Puppet Security Provider

This is today the default provider and works exactly like always for Choria, it supports enrolling with the Puppet CA via `choria enroll` and does basically what `puppet agent --waitforcert 10` would do.

By default it will ask Puppet for its configured SSL directory and in there expect certificates, ca, keys etc in all the places Puppet will put them.

It has relatively few settings since it's designed to just work with Puppet:

|Setting|Default|Description|
|-------|-------|-----------|
|`plugin.choria.ssldir`|unset|Override the path to the Puppet SSL directory|
|`plugin.choria.puppetca_host`|`puppet`|By default it will use SRV records to locate the Puppet CA, when set this overrides|
|`plugin.choria.puppetca_port`|`8140`|By default it will use SRV records to locate the Puppet CA, when set this overrides|

### File Security Provider

The file security provider is designed for people who wish to place their SSL related files on nodes using non Puppet means, it does not support enrolling but supports entirely arbitrary locations.

|Setting|Default|Description|
|-------|-------|-----------|
|`plugin.security.file.certificate`|unset|Path to the public certificate for the instance|
|`plugin.security.file.key`|unset|Path to the private key for the instance|
|`plugin.security.file.ca`|unset|Path to the CA to use|
|`plugin.security.file.cache`|unset|Path to the directory to cache client certificates|

## NATS based Middleware Broker

This sets up a managed NATS instance, it's functionally equivalent to just running NATS standalone but it's easier to get going and with fewer settings to consider.

SSL is setup to be compatible with Choria - ie. uses the Puppet certificates etc.

```ini
# enables the middleware broker
plugin.choria.broker_network = true

# address for all the network ports, defaults to :: which should be fine for most so this is optional
plugin.choria.network.listen_address = 0.0.0.0

# port it listens on for clients, this is the default when not set
plugin.choria.network.client_port = 4222

# port it listens on for other choria brokers for clustering purposes, this is the default when not set
plugin.choria.network.peer_port = 5222

# other brokers in a cluster
plugin.choria.network.peers = nats://choria1:5222, nats://choria2:5222, nats://choria3:5222

# username and password for cluster connections, no need for this typically, it would use CA validated TLS
# allowing only certs signed by your CA to connect
plugin.choria.network.peer_user = choria_cluster
plugin.choria.network.peer_password = s£cret

# the NATS network write deadline time, generally this is best left untouched
plugin.choria.network.write_deadline = 5s # default

# enables the typical NATS stats/status port, default is set to 0 and so disabled
plugin.choria.stats_port = 8222

# listens for stats to everyone, 127.0.0.1 by default
plugin.choria.stats_address = 0.0.0.0
```

## Federation Broker

The Federation Broker for now is configured using the exact same method as the Ruby one, it should be a drop in replacement.

```ini
# enables the federation broker
plugin.choria.broker_federation = true

# these settings are identical as the ruby one so I wont show them all
plugin.choria.broker_federation_cluster = production
plugin.choria.federation.instance = 1
```

## Protocol Adapter

The Protocol Adapter is a new feature that exist to adapt Choria traffic into other systems.  The initial use case is to receive all Registration data within a specific Collective and publish those into NATS Streaming.

I imagine a number of other scenarios:

  * Publishing registration data to Kafka, Lambda, Search systems, other CMDB
  * Updating PuppetDB facts more frequently than node runs to optimize discovery against it
  * Setting up generic listeners like `choria.adapter.elk` that can be used to receive replies and publish them into ELK.  You can do `mco rpc ... --reply-to choria.adapter.elk --nr` which would then not show the results to the user but instead publish them to ELK

Here we configure the NATS Streaming Adapter.  It listens for `request` messages from the old school MCollective Registration system and republish those messages into NATS Streaming where you can process them at a more leisurely pace and configure retention to your own needs.

Reliable connection handling requires at least NATS Streaming Server 0.10.0

```ini
# sets up a named adapter, you can run many of the same type
plugin.choria.adapters = discovery

# configure this discovery adapter
plugin.choria.adapter.discovery.type = nats_stream

# configure the work queue size, this can be big when the stream is far from the
# adapter and you have a high frequency result set like discovery with 50 000 nodes.
# This is basically the buffer where messages are stored in, on a big network with
# many nodes you should cater for your biggest bursts in traffic.
# The default is 1000
plugin.choria.adapter.queue_len = 50000

# in this case the adapter does NATS->NATS Streaming so you need to configure both sides
# here is NATS Streaming
plugin.choria.adapter.discovery.stream.servers = stan1:4222,stan2:4222
plugin.choria.adapter.discovery.stream.clusterid = prod
plugin.choria.adapter.discovery.stream.topic = discovery # defaults to same as adapter name
plugin.choria.adapter.discovery.stream.workers = 10 # default

# here is the Collective side
plugin.choria.adapter.discovery.ingest.topic = mcollective.broadcast.agent.discovery
plugin.choria.adapter.discovery.ingest.protocol = request # or reply
plugin.choria.adapter.discovery.ingest.workers = 10 # default
```

## Choria Server

This is a replacement `mcollectived`, that can host MCollective agents written in ruby along with a host of other features, notable absence:

  * Compound filters do not work at all - those with -S

You run it with `choria server run --config server.cfg`

Apart from all the usual stuff about identity, logfile etc, you can enable the new registration publisher like this, it just publishes registration data found in the file every 10 seconds:

```ini
registration = file_content
registerinterval = 10
registration_splay = true # optional, false by default
plugin.choria.registration.file_content.data = /tmp/json_registration.json
plugin.choria.registration.file_content.target = myco.cmdb # optional
```

### Custom binaries and packages

The building and packaging is done using a set of commands ran in docker containers - so your builder needs docker, you can configure quite a lot about the build.

To build a custom el7 rpm with custom paths and names and TLS/SSL turned off, you do create a section in `packager/buildspec.yaml` like this:

```yaml
orch:
  compile_targets:
    defaults:
      output: acme-orch-{{version}}-{{os}}-{{arch}}
      pre:
        - go generate
      flags:
        TLS: "false"
        Secure: "false"

    64bit_linux:
      os: linux
      arch: amd64

  packages:
    defaults:
      name: acme-orch
      bindir: /usr/local/acme/orch/bin
      etcdir: /usr/local/acme/orch/etc
      release: 1
      manage_conf: 1
      contact: you@example.net

    el7_64:
      template: el/el7
      dist: el7
      target_arch: x86_64
      binary: 64bit_linux
```

You can now build the whole thing:

```bash
BUILD=orch VERSION=1.0.0acme rake build
```

When you are done you will have:

  * an rpm called `acme-orch-1.0.0acme-1.el7.x86_64.rpm`
  * the binary will be `/usr/local/acme/bin/acme-orch`
  * config files, log files, services all will be personalized around `acme-orch`
  * It will not speak TLS
  * It will not use Puppet certificates for security

A number of things are customizable see the section at the top of the `buildspec.yaml` and comments in the build file.

In general you should only do this if you know what you are doing, have special needs, want custom agents etc

### Compiling in custom agents
Agents can be written in Go and if you're building a custom binary you can include your agents
in your binary.

During your CI or whatever you have to `glide get` the repo with your agent so it's available during compile, then create a file `packager/user_plugins.yaml`:

```yaml
---
foo: github.com/acme/foo_agent/foo
```

When you run `go generate` (done during the building phase for you) this will create the shim you need to compile your agent into the binary.

Your agent must implement the `plugin.Pluggable` interface.

### Provisioning

Choria supports a auto provisioning flow where should it start with a configuration that enables provisioning - or optionally one that does not specifically disable it - it will connect to a broker that gets set during compile time.

Provisioning is not enabled - or configurable - in the default shipped binaries, as an advanced feature this is reserved for people who build their own custom setups.

The Choria Server in provisioning mode will be configured with:

 * Federation is disabled
 * The `provisioning` collective is the only active one and the main collective
 * Registration is enabled with a 120 second interval
 * Registration splay is off
 * File registration gets published to `choria.provisioning_data`
 * File registration data source set to what is in `build.ProvisionRegistrationData`
 * The default [provisioning-agent](https://github.com/choria-io/provisioning-agent) unless `build.ProvisionAgent` is `"false"`
 * Security enabled - requiring SSL certificates like those from Puppet - unless `build.ProvisionSecure` is `"false"`

Please see the documentation in the [provisioning-agent](https://github.com/choria-io/provisioning-agent) repository for how to enable and use this feature.

### Compiling in custom agent providers
Agent Providers allow entirely new ways of writing agents to be created.  An example is the one that runs old mcollective ruby agents
within a choria instance.

During your CI or whatever you have to `glide get` the repo with your agent so it's available during compile, then create a file `packager/user_plugins.yaml`:

```yaml
---
rubymco: github.com/choria-io/go-choria/mcorpc/ruby
```

When you run `go generate` (done during the building phase for you) this will create the shim you need to compile your agent into the binary.

Your agent must implement the `plugin.Pluggable` interface.

## Packages

RPMs are hosted in the Choria yum repository for el6 and 7 64bit systems, the official choria Puppet module can configure these for you:

```ini
[choria_release]
name=choria_release
baseurl=https://packagecloud.io/choria/release/el/$releasever/$basearch
repo_gpgcheck=1
gpgcheck=0
enabled=1
gpgkey=https://packagecloud.io/choria/release/gpgkey
sslverify=1
sslcacert=/etc/pki/tls/certs/ca-bundle.crt
metadata_expire=300
```

## Nightly Builds

Nightly RPMs are published for EL7 64bit in the following repo:

```ini
[choria_nightly]
name=choria_nightly
baseurl=https://packagecloud.io/choria/nightly/el/$releasever/$basearch
repo_gpgcheck=1
gpgcheck=0
enabled=1
gpgkey=https://packagecloud.io/choria/nightly/gpgkey
sslverify=1
sslcacert=/etc/pki/tls/certs/ca-bundle.crt
metadata_expire=300
```

Nightly packages are versioned `0.99.0` with a date portion added:  `choria-0.99.0.20180126-1.el7.x86_64.rpm`

## Thanks

<img src="https://packagecloud.io/images/packagecloud-badge.png" width="158">
