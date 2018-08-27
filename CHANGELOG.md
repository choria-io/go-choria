|Date      |Issue |Description                                                                                              |
|----------|------|---------------------------------------------------------------------------------------------------------|
|2018/08/27|      |Release 0.6.0                                                                                            |
|2018/08/26|      |Update provisioning agent dependency, allows `restart` when not in provisioning mode if a token is set   |
|2018/08/26|422   |publish a shutdown lifecycle event on clean shutdowns                                                    |
|2018/08/25|419   |Add `tool event`, `tool sub` and `tool pub` commands to the CLI                                          |
|2018/08/24|416   |Publish a startup lifecycle event on startup                                                             |
|2018/08/22|411   |Expose the version to Prometheus as `choria_build_info`                                                  |
|2018/08/22|412   |Attempt to find the FQDN via DNS lookups before calling Puppet when the hostname is incomplete           |
|2018/08/16|408   |Add a plugin to assist with resolving the provisioning mode target brokers                               |
|2018/08/10|402   |Ensure provisioning mode is active only for the server and not client invocations                        |
|2018/08/09|      |Release 0.5.1                                                                                            |
|2018/08/09|403   |Ensure insecure provisioning mode works on non puppet nodes                                              |
|2018/08/03|398   |Support an auth token during provisioning                                                                |
|2018/08/02|394   |Support a fact source during provisioning mode                                                           |
|2018/08/02|394   |Fix registration handling when in provisioning mode                                                      |
|2018/07/31|390   |Avoid leaking metrics in long running clients that make many new client instances                        |
|2018/07/20|      |Release 0.5.0                                                                                            |
|2018/07/13|382   |Improve mcollective compatibility by supporting shallow merges of multiple fact files                    |
|2018/07/12|379   |Increase the NATS Stream Adapter work channel size to function on large networks                         |
|2018/07/12|377   |When adapting Choria messages to NATS Streams include the Choria RequestID                               |
|2018/07/12|375   |Ensure all loggers are configured with the correct level and format                                      |
|2018/07/11|      |Disable full JSON schema validation by default due to performance concerns (go-protocol#23)              |
|2018/07/11|      |Update `gnatsd` to `1.2.0` to improve stability at >30k nodes when clustered (go-network-broker#6)       |
|2018/07/11|373   |Support Ubuntu 18.04                                                                                     |
|2018/07/11|361   |When embedding the Choria Server initial startup errors can now be detected and handled                  |
|2018/07/11|362   |When embedding the Choria Server one can now prevent certain agents from loading                         |
|2018/07/11|366   |Consult `/etc/choria/client.cfg` and `~/.choria` in addition to mcollective locations                    |
|2018/07/03|359   |Resolve a go routine leak when using the connector in a long running client                              |
|2018/06/26|353   |Handle connection errors in NATS Streaming brokers in the Adapters, require NATS Streaming >= `0.10.0`   |
|2018/06/18|346   |Add a high performance, filtering capable basic network validation CLI `choria ping`                     |
|2018/06/15|343   |Resolve the merging of identity & certname concepts that erroneously happened during the security refacor|
|2018/06/14|341   |Ensure non root users - like clients - get a username based certname not FQDN one                        |
|2018/06/07|336   |Fix the setting that allows disabling broker order randomization                                         |
|2018/06/06|333   |Randomize broker connections by default                                                                  |
|2018/06/06|331   |Add a short grace period to clock checks to allow for real world realities wrt synced clocks             |
|2018/05/31|      |Release 0.4.0                                                                                            |
|2018/05/29|320   |Make the enroll process more robust in the face of interruptions                                         |
|2018/05/23|308   |Fix running `choria buildinfo`                                                                           |
|2018/05/23|309   |Create `go-security` package with the Choria security providers for reuse in other eco system projects   |
|2018/05/22|306   |Fix registration feature when running in insecure mode                                                   |
|2018/05/18|302   |Include a hint when the source data for registration changed                                             |
|2018/05/16|      |Release 0.3.0                                                                                            |
|2018/05/08|287   |Create the concept of a Security Provider and create providers for Puppet and File, add `choria enroll`  |
|2018/05/03|284   |On systemd based distributions ensure that upgrading choria with Puppet is more reliable                 |
|2018/04/25|271   |Log rotation for `choria-*.log` which covers audit, ruby and more                                        |
|2018/04/25|267   |Ensure that the ruby shim based agents have access to the correct request time                           |
|2018/04/24|      |Release 0.2.0                                                                                            |
|2018/04/23|243   |Create a compatibility framework for MCollective Agents written in Ruby                                  |
|2018/04/23|252   |Avoid logrotate errors when the package was installed but choria never ran                               |
|2018/04/09|240   |When facter is available use it to determine the FQDN to improve default behavior on debian like systems|
|2018/04/09|236   |Allow `nats://host:port` and `host:port` to be used when referencing brokers                             |
|2018/04/09|235   |Detect empty initial server list when starting federation brokers                                        |
|2018/03/29|229   |Surface more NATS internal debug logs as notice and error                                                |
|2018/03/29|228   |Increase TLS timeouts to 2 seconds to improve functioning over latency and heavily loaded servers        |
|2018/03/26|199   |Do not use HTTP to fetch internal NATS stats                                                             |
|2018/03/26|220   |Update gnats and go-nats to latest versions                                                              |
|2018/03/26|222   |Allow the network broker write deadline to be configured                                                 |
|2018/03/23|218   |Avoid rotating empty log files and ensure the newest log is the one being written too                    |
|2018/03/21|      |Release 0.1.0                                                                                            |
|2018/03/08|208   |Improve compatibility with MCollective Choria by not base64 encoding payloads                            |
|2018/03/08|207   |Ensure the filter is valid when creating `direct_request` messages                                       |
|2018/03/07|204   |Support writing a thread dump to the OS temp dir on receiving SIGQUIT                                    |
|2018/03/07|202   |Do not rely purely on `PATH` to find `puppet`, look in some standard paths as well                       |
|2018/03/06|      |Release 0.0.11                                                                                           |
|2018/03/06|198   |Reuse http.Transport used to fetch gnatsd statistics to avoid a leak on recent go+gnatsd combination     |
|2018/03/05|      |Release 0.0.10                                                                                           |
|2018/03/05|194   |Revert `gnatsd` to `1.0.4`, upgrade Golang to `1.10`                                                     |
|2018/03/05|      |Release 0.0.9                                                                                            |
|2018/03/05|190   |Downgrade to Go 1.9.2 to avoid run away go routines                                                      |
|2018/03/05|      |Release 0.0.8                                                                                            |
|2018/03/05|187   |Create a schema for the NATS Stream Adapter and publish it in the messages                               |
|2018/03/05|174   |Report the `mtime` of the file in the file content registration plugin, support compressing the data     |
|2018/03/02|183   |Update Go to `1.10`                                                                                      |
|2018/03/01|180   |Show the Go version used to compile the binary in `buildinfo`                                            |
|2018/03/01|173   |Record and expose the total number of messages received by the `server`                                  |
|2018/03/01|176   |Intercept various `gnatsd` debug log messages and elevate them to notice and error                       |
|2018/03/01|175   |Update embedded `gnatsd` to `1.0.6`                                                                      |
|2018/02/19|171   |Show embedded `gnatsd` version in `buildinfo`                                                            |
|2018/02/19|      |Release 0.0.7                                                                                            |
|2018/02/19|165   |Discard NATS messages when the work buffer is full in the NATS Streaming adapter                         |
|2018/02/19|166   |Remove unwanted debug output                                                                             |
|2018/02/16|167   |Clarify the Choria flavor reported by choria_util#info                                                   |
|2018/02/01|163   |Avoid large data storms after a reconnect cycle by limiting the publish buffer                           |
|2018/02/01|151   |Add xenial and stretch packages                                                                          |
|2018/01/22|152   |Support automagic validation of structs received over the network, support shellsafe for now             |
|2018/01/20|150   |Release 0.0.6                                                                                            |
|2018/01/20|58    |A mostly compatible `rpcutil` agent was added                                                            |
|2018/01/20|148   |The TTL of incoming request messages are checked                                                         |
|2018/01/20|146   |Stats about the server and message life cycle are recorded                                               |
|2018/01/19|133   |A timeout context is supplied to actions when they get executed                                          |
|2018/01/16|134   |Use new packaging infrastructure and move building to a circleci pipeline                                |
|2018/01/12|131   |Additional agents can now be added into the binary at compile time                                       |
|2018/01/12|125   |All files in additional dot config dirs are now parsed                                                   |
|2018/01/12|128   |Add additional fields related to the RPC request to mcorpc.Request                                       |
|2018/01/10|120   |The concept of a provisioning mode was added along with a agent to assist automated provisioning         |
|2018/01/09|60    |Auditing was added for mcorpc agents                                                                     |
|2018/01/09|69    |The protocol package has been moved to `choria-io/go-protocol`                                           |
|2018/01/08|118   |Create a helper to parse mcorpc requests into a standard structure                                       |
|2018/01/05|114   |Ensure the logfile name matches the package name                                                         |
|2018/01/06|      |Release 0.0.5                                                                                            |
|2018/01/05|110   |Correctly detect startup failures in the el6 init script                                                 |
|2018/01/04|111   |Treat the defaults file as config in the el6 rpm                                                         |
|2017/12/25|108   |Improve logrotation - avoid appending to a rotated file                                                  |
|2017/12/21|106   |Make the max connections a build parameter and default it to 50 000                                      |
|2017/12/20|101   |Add a random backoff to initial connection in adapters and the connector                                 |
|2017/12/20|102   |Expose connector details to prometheus                                                                   |
|2017/12/13|      |Release 0.0.4                                                                                            |
|2017/12/14|97    |Stats about the internals of the protocol are exposed                                                    |
|2017/12/14|80    |When doing SRV lookups employ a cache to speed things up                                                 |
|2017/12/14|92    |When shutting down daemons on rhel6 wait for them to exit and then KILL them after 5 seconds             |
|2017/12/14|91    |Avoid race condition while determining if the network broker started                                     |
|2017/12/14|90    |Emit build info on `/choria/`                                                                            |
|2017/12/13|      |Release 0.0.3                                                                                            |
|2017/12/12|81    |Export metrics `/choria/prometheus` when enabled                                                         |
|2017/12/10|73    |Federation brokers now correctly subscribe to the configured names                                       |
|2017/12/10|71    |Fix TLS network cluster                                                                                  |
|2017/12/10|      |Release 0.0.2                                                                                            |
|2017/12/10|67    |Distribute sample `broker.conf` and `server.conf`                                                        |
|2017/12/10|65    |When running as root do not call `puppet apply` 100s of times                                            |
|2017/12/10|64    |Ensure the broker exits on interrupt when the NATS based broker is running                               |
|2017/12/09|59    |Add a compatible `choria_util` agent                                                                     |
|2017/12/09|57    |Create basic MCollective SimpleRPC compatible agents written in Go and compiled in                       |
|2017/12/08|53    |Adds the `buildinfo` subcommand                                                                          |
|2017/12/08|52    |Improve cross compile compatibility by using `os.Getuid()` instead of `user.Current()`                   |
|2017/12/08|      |Release 0.0.1                                                                                            |
