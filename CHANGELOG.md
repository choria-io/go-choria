|Date      |Issue |Description                                                                                              |
|----------|------|---------------------------------------------------------------------------------------------------------|
|2020/11/25|      |Release 0.18.0                                                                                           |
|2020/10/21|999   |Add a timer watcher that changes state after a time expires                                              |
|2020/10/21|999   |Support creating Apple Homekit buttons in Autonomous Agents                                              |
|2020/09/28|      |Release 0.17.0                                                                                           |
|2020/09/04|989   |Add a generic shell completion helper and support ZSH completion                                         |
|2020/08/25|987   |Support NATS Leafnodes to extend the Choria Broker in a TLS free way specifically usable by AAA clients  |
|2020/08/03|982   |Scout checks can have annotations that are published in events                                           |
|2020/08/03|920   |Add `choria scout maintenance` and `choria scout resume` commands                                        |
|2020/08/01|920   |Add a `choria scout trigger` command that triggers an immediate check and associated events              |
|2020/08/01|977   |Generated clients can now set a progress bar                                                             |
|2020/07/30|975   |Prevent int overflow in time fields in some Scout events                                                 |
|2020/07/26|920   |Add a `--table` option to `choria req` and a new formatter in generated clients                          |
|2020/07/26|920   |Add a `choria scout status` command that can show all checks on a node                                   |
|2020/07/24|968   |Improve the history presented in Scout events                                                            |
|2020/07/22|966   |Remove the concept of a site wide Gossfile                                                               |
|2020/07/21|964   |Allow multiple Gossfiles and multiple Goss checks                                                        |
|2020/07/18|      |Release 0.16.0                                                                                           |
|2020/07/18|960   |Add a `choria scout watch` command                                                                       |
|2020/07/17|957   |Restore the ability for DDLs to declare display formats for aggregate outputs                            |
|2020/07/16|948   |Support performing `goss` validation in the `nagios` autonomous agent                                    |
|2020/07/15|842   |Avoid zombies when Ruby agents exceed their allowed run time                                             |
|2020/07/09|944   |Extract the generic result display logic from `choria req` into a reusable package                       |
|2020/07/09|942   |Include a snapshot of recent check states in published check events                                      |
|2020/07/08|939   |Improve using the supplied logger in generated clients                                                   |
|2020/07/08|938   |Add helpers to parse complex data in generated clients                                                   |
|2020/07/08|937   |Generated clients perform 2 discoveries per request                                                      |
|2020/07/07|935   |Release packages for Ubuntu Focal (20.04 LTS)                                                            |
|2020/07/07|932   |Fix targeting a specific sub collective in the `req` command                                             |
|2020/07/07|928   |Add a new `scout` agent and Golang client                                                                |
|2020/07/03|920   |Initial work on a Scout framework towards building a monitoring related distribution                     |
|2020/07/01|      |Release 0.15.0                                                                                           |
|2020/06/29|913   |Support preparing for shutdown by closing connections and emiting shutdown events when embedded          |
|2020/06/26|895   |Support NATS JetStream Streaming Server in Choria Broker                                                 |
|2020/06/24|907   |Support arm5 and 7 Debian packages                                                                       |
|2020/06/20|895   |Support Nagios compatible plugins in the new `nagios` autonomous agent watcher                           |
|2020/06/16|893   |Server instances embedded in other software can now be shutdown using `Shutdown()`                       |
|2020/06/15|887   |Track nodes expired by maintenance in the tally helper                                                   |
|2020/06/13|      |Improve FQDN resolution when running in a kubernetes pod                                                 |
|2020/06/12|879   |Allow the public name of the network broker to be configured                                             |
|2020/06/12|877   |Support cert-manager.io as security provider                                                             |
|2020/06/07|865   |Correctly handle provisioning by SRV domain                                                              |
|2020/06/07|863   |Allow provisioning brokers to have user/password authentication                                          |
|2020/05/14|860   |Perform backoffs between reconnects to the network broker                                                |
|2020/04/22|857   |Cosmetic improvements to windows packages                                                                |
|2020/04/19|      |Release 0.14.0                                                                                           |
|2020/04/16|854   |Correctly report insecure builds                                                                         |
|2020/04/07|852   |Install `choria` binary in /usr/bin and not /usr/sbin                                                    |
|2020/03/25|846   |Various improvements to generated RPC clients                                                            |
|2020/03/24|844   |Export facts to external agents                                                                          |
|2020/03/22|801   |Expose statistics for NATS Leafnodes                                                                     |
|2020/03/16|840   |Improve formatting of node lists at the end of requests                                                  |
|2020/03/11|687   |Support enforcing the use of filters on all RPC requests using `plugin.choria.require_client_filter`     |
|2020/03/03|834   |Add Debian Buster support                                                                                |
|2020/02/17|831   |Cache transport messages when doing batched requests to improve pkcs11 integration                       |
|2020/02/13|827   |Ensure agent filter is added when discovering nodes                                                      |
|2020/02/08|817   |Add `choria tool config` to view configuration paramters and current values                              |
|2020/02/08|814   |Set `PATH` when calling external agents                                                                  |
|2020/02/05|794   |Merge `go-lifecycle` into `go-choria`                                                                    |
|2020/02/05|794   |Merge `go-protocol`, `go-security`, `mcorpc-agent-provider` and `go-config` into `go-choria`             |
|2020/02/05|794   |Merge `go-confkey`, `go-validator`, `go-puppet`, `go-network-broker` and `go-srvcache` into `go-choria`  |
|2020/01/30|      |Update to CloudEvents 1.0.0                                                                              |
|2020/01/23|774   |Support logging to Windows Event log                                                                     |
|2020/01/23|772   |Support running as a Windows service                                                                     |
|2020/01/17|769   |Add basic Windows pacakges                                                                               |
|2020/01/16|      |Support use selectable SSL Ciphers using `plugin.security.cipher_suites` and `plugin.security.ecc_curves`|
|2020/01/12|      |Release 0.13.1                                                                                           |
|2019/12/25|758   |Extract RPC reply rendering to the mcorpc package-agent-provider                                         |
|2019/12/23|754   |Extract parts of the filter parsing logic to the `protocol` package                                      |
|2019/12/15|746   |Support remote request signers such as `aaasvc`                                                          |
|2019/12/09|743   |Support generating Go clients using `choria tool generate client`                                        |
|2019/12/05|      |Release 0.13.0                                                                                           |
|2019/12/05|737   |Add a tech preview JetStream adapter                                                                     |
|2019/12/04|731   |Switch to CloudEvents v1.0 format for lifecycle events and machine events                                |
|2019/12/02|709   |Build RHEL 8 packages nightly and on release                                                             |
|2019/12/02|548   |Improve startup when embedding the server in other programs                                              |
|2019/11/29|724   |Improve stability on a NATS network with Gateways                                                        |
|2019/11/28|720   |Improve the calculations of total request time in the `choria req` command                               |
|2019/11/21|710   |Support Synadia NGS as a NATS server for Choria                                                          |
|2019/10/26|705   |Add `choria tool jwt` to create provisioning tokens                                                      |
|2019/10/25|705   |Allow a JWT file to configure provisioning behavior and enable provisioning in the FOSS binary           |
|2019/10/14|703   |Allow `choria req` output to be saved to a file                                                          |
|2019/10/01|700   |Force convert a DDL from JSON on the CLI without prompts                                                 |
|2019/09/20|      |Release 0.12.1                                                                                           |
|2019/09/19|      |Support Authorization and External Agents via latest MCORPC provider                                     |
|2019/09/16|681   |Allow agents to associate with specific agent providers using the `provider` field in metadata           |
|2019/09/12|678   |Support generating Ruby and JSON DDL files using `choria tool generate ddl`                              |
|2019/09/09|      |Release 0.12.0                                                                                           |
|2019/09/09|      |Broker based on NATS 2.0 via `go-network-broker` version `1.3.1`                                         |
|2019/09/07|670   |Improve the output from `choria ping --graph`                                                            |
|2019/09/06|664   |Add a pkcs11 security provider                                                                           |
|2019/09/04|663   |Add a `choria req` tool to eventually replace `mco rpc`                                                  |
|2019/08/09|652   |Write init scripts to the correct location on RHEL                                                       |
|2019/07/24|642   |Show dependencies compiled into the binary in `choria buildinfo`                                         |
|2019/07/15|632   |Decrease memory use in adapters by lowering the work queue length                                        |
|2019/06/27|621   |Choria Provisioner is now a proper plugin                                                                |
|2019/06/27|623   |Support `agents.ShouldActivate()` checks when loading agents                                             |
|2019/06/26|617   |Support NATS 2.0 credentials and user/password                                                           |
|2019/06/26|617   |Fix `choria ping`                                                                                        |
|2019/06/12|      |Release 0.11.1                                                                                           |
|2019/04/20|      |Support email SANs in client certificates via `go-security` `0.4.2`                                      |
|2019/06/11|609   |Verify that only known transitions and states are mentioned in the machine specification                 |
|2019/06/11|607   |Ensure the machine directory is in the `PATH`                                                            |
|2019/05/30|605   |Fix `environment` handling for exec watchers                                                             |
|2019/05/29|602   |Ensure machines are runable on the CLI                                                                   |
|2019/05/29|599   |Support run-once exec watchers by setting `interval=0`                                                   |
|2019/05/29|597   |Do not manage Autonomous Agents in provisioning mode                                                     |
|2019/05/28|591   |Add a `scheduler` watcher for Autonomous Agents                                                          |
|2019/05/27|      |Release 0.11.0                                                                                           |
|2019/05/23|      |Log discovery requests in a similar manner to RPC requests via `mcorpc-agent-provider` `0.4.0`           |
|2019/05/23|      |Fix puppet provider support for `SecurityAlwaysOverwriteCache` via `go-security` `0.4.0`                 |
|2019/05/23|      |Improve excessive logging when privilged certificates are used via `go-security` `0.4.0`                 |
|2019/05/23|      |Only write certificates on change if `SecurityAlwaysOverwriteCache` is set via `go-security` `0.4.0`     |
|2019/05/22|554   |Retry SRV lookups on reconnect attempts                                                                  |
|2019/05/27|563   |Support Choria Autonomous Agents                                                                         |
|2019/03/21|557   |Force puppet environment to `production` to avoid failures about missing environment directories         |
|2019/03/19|557   |Improve error messages logged when invoking `puppet` to retrieve setting values fail                     |
|2019/03/15|555   |Add a basic utility to assist with creating deep monitoring `choria tool status`                         |
|2019/03/04|      |Release 0.10.1                                                                                           |
|2019/02/25|      |Resolve broker instability on large networks via `go-network-broker#19`                                  |
|2019/01/23|      |Release 0.10.0                                                                                           |
|2019/01/17|      |Various fixes to privileged security certificate handling via `go-security` release `0.3.0`              |
|2019/01/17|      |Allow limiting clients to sets of IPs via `go-network-broker#12`                                         |
|2019/01/09|534   |Ensure the server status file is world readable                                                          |
|2019/01/07|532   |Force exit even when worker routines are not done after `soft_shutdown_timeout`, default 2 seconds       |
|2019/01/05|530   |Further fixes to avoid concurrent hash access panics for golang client code                              |
|2019/01/03|524   |Include the server version when creating life cycle events                                               |
|2018/12/27|521   |Improve `alive` event spread by sleeping for up to a hour for initial publish                            |
|2018/12/27|519   |Expose `security.Validate` to users of the go framework                                                  |
|2018/12/27|      |Release 0.9.0                                                                                            |
|2018/12/26|      |Fix reboot splay time when doing self updates via `provisioning-agent#67`                                |
|2018/12/26|      |Increase `choria_util` agent timeout to facilitate slow facter runs via `mcorpc-agent-provider#36`       |
|2018/12/26|515   |Cache facter lookups                                                                                     |
|2018/12/21|510   |Publish new `alive` life cycle events every hour                                                         |
|2018/12/19|      |support `~/.choriarc` and `/etc/choria/client.conf` for client configs                                   |
|2018/12/19|      |Report protocol security and connector TLS in `choria_util#info` via `mcorpc-agent-provider#33`          |
|2018/12/19|501   |Allow default configuration values to be mutated at startup using a plugin                               |
|2018/12/07|495   |Allow server status to be written during provision mode                                                  |
|2018/11/30|      |Release 0.8.0                                                                                            |
|2018/11/28|489   |Avoid a panic that affected clients written in Go when closing connections to the broker                 |
|2018/11/23|      |Improve backward compatibility when handling slashes in regex for allowed certs (go-security#22)         |
|2018/11/23|485   |Fail gracefully in the `ping` app when the configuration is not present                                  |
|2018/11/20|483   |Resolve a client subscription leak by unsubscribing on context cancellation                              |
|2018/11/15|      |When provisioning is compiled in - support self updating using `go-updater` (provisioning-agent#53)      |
|2018/11/14|476   |Allow the SSL cache to always be written via `plugin.security.always_overwrite_cache`                    |
|2018/11/02|473   |Support running Choria Server in a namespace on Enterprise Linux via a COMMAND_PREFIX in the init script |
|2018/10/24|467   |Support writing server status regularly                                                                  |
|2018/10/27|470   |Switch to `github.com/gofrs/uuid` for UUID generation                                                    |
|2018/10/18|      |Release 0.7.0                                                                                            |
|2018/10/02|462   |Allow custom packages to supply their own sysv init start order                                          |
|2018/09/18|458   |Update network broker to 1.1.0 which includes `gnatsd` 1.3.0                                             |
|2018/09/17|456   |Provisioner Target plugins now have a context in their calls so they can do internal retries             |
|2018/09/15|447   |Create a single plugin interface that supports many types of plugin                                      |
|2018/09/11|444   |Set ulimits for the broker appropriately for 50 000 connections                                          |
|2018/09/02|430   |Allow agents to publish lifecycle events                                                                 |
|2018/08/31|428   |Add a CLI tool to view provisioning broker decisions - `tool provisioner`                                |
|2018/08/29|426   |Correctly compiled servers will enter provisioning mode when the configuration file is missing entirely  |
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
