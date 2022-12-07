+++
title = "Monitoring"
toc = true
weight = 10
pre = "<b>1. </b>"
+++

Choria Server is designed to not open any listening ports unless its Apple HomeKit integration is enabled.

Without any opening ports monitoring it is via a state file that it writes regularly when enabled:

```ini
plugin.choria.status_file_path = /var/log/choria-status.json
plugin.choria.status_update_interval = 30
```

The above configuration will cause the status file to update every 30 seconds. This needs to be enabled for any
deep introspection.

### Nagios Check

A `nagios` protocol test is included in the command `choria tool status`, this can check various aspects of the server
operation.

```nohighlight
$ choria tool status --status-file /var/log/choria-status.json \
    --disconnected \        # alerts when the server is not connected to a broker
    --message-since 1h \    # must have received RPC requests within the last 1 hour
    --max-age 1m \          # Status file may not be older than 1 minute
    --token-age 24h \       # Alert 1 day before the token expires
    --certificate-age 24h \ # Alert 1 day before the certificate expires
    --provisioned           # Alerts if the server is in provisioned mode
```

### Autonomous Agent Check

A running instance can check itself using an Autonomous Agent, it will then public Cloud Events about it's internal state and, optionally, expose it's state to a local Prometheus Node Exporter via its text file directory.

```yaml
watchers:
  - name: check_choria
    type: nagios
    interval: 5m # checks every 5 minutes, require the status file to be 15 minutes or newer
    properties:
      builtin: choria_status
      token_expire: 1d    # alerts when the token expires soon
      pubcert_expire: 1d  # alerts when the certificate expires soon
      last_message: 1h    # alerts when no RPC message was received in 1 hour
```

Review the [Autonomous Agent](https://choria.io/docs/autoagents/) section for full detail about these checks.

If you have Prometheus Node Exporter running locally with an argument `--collector.textfile.directory=/var/lib/node_exporter/textfile` set
you can configure this path in Choria which would cause the above Autonomous Agent to write status to that directory:

```ini
plugin.choria.prometheus_textfile_directory = /var/lib/node_exporter/textfile
```

### Lifecycle Events

Choria will publish a number of events in [Cloud Events](https://cloudevents.io/) format, these can be
observed using `choria tool event`, this will include start, stop, provisioned etc events from every Choria Server instance.

Some details about these events are in these blog posts:
 
 * [Choria Lifecycle Events](https://choria.io/blog/post/2019/01/03/lifecycle/)
 * [Transitioning Events to Cloud Events](https://choria.io/blog/post/2019/12/05/cloudevents_transition/)
