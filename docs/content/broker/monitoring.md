+++
title = "Monitoring"
toc = true
weight = 10
pre = "<b>1. </b>"
+++

Choria Broker is just one process running in your server, the command `systemctl status choria-broker` will show a basic
overview of the process running, you should have basic monitoring for the process in place.

## Prometheus Metrics

By default, the broker does not expose any monitoring metrics, if you set the `plugin.choria.stats_port` configuration
to a port number it will listen on that port. You can listen on non localhost by setting `plugin.choria.stats_address`.

It will then serve a number of URLs, some from Choria and some from the embedded NATS Server - see [NATS Monitoring](https://docs.nats.io/running-a-nats-service/nats_admin/monitoring) for detail about those.

| Path              | Description                              |
|-------------------|------------------------------------------|
| `/choria/`        | Build information, run-time resource use |
| `/choria/metrics` | Prometheus format metrics                |

## System Account

Monitoring NATS and Choria Streams require a System account enabled:

```ini
plugin.choria.network.system.user = system
plugin.choria.network.system.password = s3cret
```

This should be set in the Broker configuration and any client who wish to access the broker.

We have a [basic Dashboard](https://grafana.com/grafana/dashboards/12430-choria-broker/) you can use to view these.

## Included Checks

{{% notice tip %}}
All the `choria broker server` commands are from the embedded [NATS CLI](https://github.com/nats-io/natscli) and so can be an awkward fit within our CLI hierarchy
{{% /notice %}}

We include a number of checks in the binary that can be used to monitor various aspects of the service.

| Command                                 | Description                                                                                |
|-----------------------------------------|--------------------------------------------------------------------------------------------|
| `choria broker server check connection` | Performs a basic network connection and round-trip test of the NATS service                |
| `choria broker server check stream`     | Checks the health of individual [Choria Streams](https://choria.io/docs/streams/) Streams  |
| `choria broker server check meta`       | Checks the health of the overall Choria Streams System                                     |
| `choria broker server check jetstream`  | Checks Choria Streams usage limits                                                         |
| `choria broker server check server`     | Checks the health of the embedded NATS Server                                              |
| `choria broker server check kv`         | Checks the health of [Choria Key-Value](https://choria.io/docs/streams/key-value/) buckets |

All of these Checks require the System Account to be enabled in your broker and the client configuration to have the same settings.
A custom Choria Client configuration can be set using `--choria-config` on these commands.

By default, these commands act like Nagios checks:

```nohighlight
% choria broker server check js
OK JetStream | memory=0B memory_pct=0%;75;90 storage=1942997776B storage_pct=0%;75;90 streams=13 streams_pct=0% consumers=21 consumers_pct=0%
% echo $?
0
```

They can though also output `json`, `prometheus` and `text` formats:

```nohighlight
% choria broker server check js --format text
JetStream: OK

Check Metrics

╭───────────────┬───────────────┬──────┬────────────────────┬───────────────────╮
│ Metric        │ Value         │ Unit │ Critical Threshold │ Warning Threshold │
├───────────────┼───────────────┼──────┼────────────────────┼───────────────────┤
│ memory        │ 0.00          │ B    │ 0.00               │ 0.00              │
│ memory_pct    │ 0.00          │ %    │ 90.00              │ 75.00             │
│ storage       │ 1942955289.00 │ B    │ 0.00               │ 0.00              │
│ storage_pct   │ 0.00          │ %    │ 90.00              │ 75.00             │
│ streams       │ 13.00         │      │ 0.00               │ 0.00              │
│ streams_pct   │ 0.00          │ %    │ -1.00              │ -1.00             │
│ consumers     │ 21.00         │      │ 0.00               │ 0.00              │
│ consumers_pct │ 0.00          │ %    │ -1.00              │ -1.00             │
╰───────────────┴───────────────┴──────┴────────────────────┴───────────────────╯
```

## Reports

Several run time reports are included that can show connection states and more, all of these require the System Account.

### List Brokers in the cluster

This shows all the connected NATS Servers / Choria Brokers in your cluster and some basic information about them

```nohighlight
$ choria broker server list
```

### Broker Connections

You can view and search active connections to your brokers, here we limit it to the top-5 by subject, see `--help` for other options

```nohighlight
% choria broker server report connections --top 5
```

Add the `--account=provisioning` option to see connections waiting to be provisioned if enabled.

### Streams Report

One can get a overview of Choria Streams backends:

```nohighlight
% choria broker server report jetstream
```

### Details Broker Data

A wealth of data is available in the Brokers about every connection and every subscription and more, run `choria broker server req --help` to see a full list.

## Golang Profiling

As an advanced option, that should not be enabled by default, one can enable the [Golang PProf](https://jvns.ca/blog/2017/09/24/profiling-go-with-pprof/)
port to facilitate deep debugging of memory allocations and more. How to use this is out of scope of this document and
really only useful for developers.

```ini
plugin.choria.network.pprof_port = 9090
```
