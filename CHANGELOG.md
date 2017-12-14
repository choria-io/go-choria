|Date      |Issue |Description                                                                                              |
|----------|------|---------------------------------------------------------------------------------------------------------|
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
