| Date       | Issue | Description                                                                                                        |
|------------|-------|--------------------------------------------------------------------------------------------------------------------|
| 2023/04/19 | 2052  | Ensure stream users can access KV and Object stores                                                                |
| 2023/04/18 | 2049  | Timeout initial connection attempts while preparing embedded nats CLI connection                                   |
| 2023/04/18 | 2047  | Grant access to governor lifecycle events for clients with the governor permission                                 |
| 2023/04/18 | 2045  | Expose the client governor permission on the jwt cli                                                               |
| 2023/04/18 | 2043  | Support using in-process connections for adapter communications                                                    |
| 2023/04/18 | 2041  | Only validate ed25519 signed provisioner tokens using the Issuer flow, fall back for rsa signed tokens             |
| 2023/04/14 | 2037  | Trim spaces in received kv data in order to determine if it's JSON data or not                                     |
| 2023/04/11 | 2029  | Adds a new `plugins` watcher that can manage auto agents and external rpc agents                                   |
| 2023/04/10 | 2026  | Support booleans, enums and more in the `rpc` builder command flags parsing                                        |
| 2023/03/31 | 2022  | Use a native sha256 checker rather than rely on OS provided binary in the `archive` watcher                        |
| 2023/03/30 | 2019  | Support runtime reloading and relocation of external agents without restarting the server                          |
| 2023/03/22 |       | Release 0.27.0                                                                                                     |
| 2023/03/21 | 2010  | Record the builtin type as plugin in nagios watcher events                                                         |
| 2023/03/14 | 2001  | Support adding headers to Choria Message Submit messages                                                           |
| 2023/03/07 | 1998  | Support leader election for tally and label metrics by leader state                                                |
| 2023/03/06 | 1996  | Emit new `upgraded` events when release upgrading a running server via provisioning                                |
| 2023/03/03 | 1994  | Record exec watcher events in lifecycle recorder                                                                   |
| 2023/02/21 | 1740  | Allow protocol v1 provisioners and servers to be provisioned on a v2 broker                                        |
| 2023/02/20 | 1990  | Add `context.Context` to the provisioner target resolve `Configure()` method                                       |
| 2023/02/19 | 1984  | Include the number of Lifecycle events published in instance stats, data and rpcutil output                        |
| 2023/02/17 | 1987  | Export `SetBuildBasedOnJWT` in default proftarget plugin                                                           |
| 2023/02/15 | 1982  | Ensure multiple AAA Login URLs are parsed correctly                                                                | 
| 2023/02/14 | 1980  | Correctly detect paths to ed25519 public keys that are 64 characters long as paths                                 |
| 2023/02/13 | 1978  | Add the `--governor` permission to `choria jwt server`                                                             |
| 2023/02/02 | 1976  | Allow `choria machine run` to be used without a valid Choria install                                               |
| 2023/02/02 | 1974  | Fix validation of Autonomous Agents that use timer watchers                                                        |
| 2023/01/26 | 1972  | Create a tool to monitor JWT token health and contents                                                             |
| 2023/01/23 | 1968  | Improve handling of governors on slow nodes and during critical failures                                           |
| 2023/01/19 | 1966  | Improve `plugin generate ddl` UX                                                                                   |
| 2023/01/18 | 1964  | Improve DDL schema validation                                                                                      |
| 2023/01/12 | 1959  | Ensure provisioning tokens have a default non-zero expiry                                                          |
| 2023/01/12 | 1740  | Extract the `tokens` package into `github.com/choria-io/tokens`                                                    |
| 2023/01/10 | 1952  | Improve fact filter parsing to handle functions both left and right of the equation                                |
| 2022/12/15 | 1942  | Support streaming JSON output on `choria req` to assist non-golang clients to be built quicker                     |
| 2022/12/14 | 1939  | Support multi-arch binaries for external agents                                                                    |
| 2022/12/12 |       | Support `direct` mode for Choria Key-Value Stores to increase scale and throughput                                 |
| 2022/12/08 | 1934  | Allow the `machines` watcher spec signer public key to be set in config                                            |
| 2022/12/06 |       | Create a new dedicated backplane docs site https://choria-io.github.io/go-choria                                   |
| 2022/12/01 |       | Remove numerous deprecated configuration settings                                                                  |
| 2022/12/01 | 1924  | Add a new RPC Authorization plugin that requires and authorize policies found in client JWTs                       |
| 2022/11/29 | 1922  | Improve handling defaults in output DDLs for generated clients                                                     |
| 2022/11/29 | 1918  | Support full Choria version upgrades during provisioning                                                           |
| 2022/11/29 | 1916  | Support Choria Provisioner using version to Protocol                                                               |
| 2022/11/28 | 1913  | Allow provisioning over non TLS when holding an Org Issuer signed provisioning JWT                                 | 
| 2022/11/28 | 1913  | New Client JWT permissions to indicate a client can access the `provisioning` account in the broker                |
| 2022/11/23 | 1911  | Do not terminate servers on authentication error                                                                   |
| 2022/11/22 | 1909  | Support Hashicorp Vault as storage for the Organization Issuer and the `choria jwt` command                        |
| 2022/11/10 | 1900  | Introduce the concept of a Organization Issuer and chain of trust JWT tokens for Server and Client issuers         |
| 2022/11/09 | 1898  | Enhance the request signing protocol to include signatures made using the private key                              | 
| 2022/11/06 | 1863  | Choria Message Submit can sign published messages when using Choria Security                                       |
| 2022/11/07 | 1740  | Introduce Choria JWT based security and Protocol version 2                                                         |
| 2022/11/07 |       | Release 0.26.2                                                                                                     |
| 2022/11/01 | 1886  | Allow additional publish and subscribe subjects to be added to client tokens                                       |
| 2022/10/05 | 1869  | Improve the error handling in `choria tool status` when the status file does not exist                             |
| 2022/10/04 | 1866  | Fix inventory groups in inventory files, they now work with all agents                                             |
| 2022/10/03 | 1740  | Support ed25519 keys for signing JWT tokens                                                                        |
| 2022/10/03 | 1840  | Additional JWT permissions that should be set to allow fleet management access                                     |
| 2022/09/29 | 1854  | Correctly detect empty filters that might have resulted in unexpected replies                                      |
| 2022/09/29 | 1667  | Upgrade to a faster and more modern JSON schema validator                                                          |
| 2022/09/28 | 1740  | Adds an experimental `choria tool protocol` command that can live view Choria traffic                              |
| 2022/09/28 | 1845  | Avoid some blocking writes in autonomous agent startup, internal efficiency only                                   |
| 2022/09/27 | 1740  | Remove the concept of a cache from the security subsystem and other refactors                                      |
| 2022/09/26 | 1836  | Add the delegation property to client JWTs                                                                         |
| 2022/09/21 | 1832  | Render all tables using UTF-8, remove old table dependency                                                         |
| 2022/09/19 | 1826  | Add the new `choria scout validate` command that acts as a goss frontend                                           |
| 2022/09/14 | 1822  | Allow RPC clients to supply a goss manifest to execute on the network, from file or KV bucket                      |
| 2022/09/14 | 1820  | Improve UX of some table layouts                                                                                   |
| 2022/09/13 | 1818  | Allow direct get to be configured for KV                                                                           |
| 2022/09/08 | 1815  | Restore the ability for provisioners to version update Choria in-place                                             |
| 2022/09/08 | 1812  | Own implementation of the Streams based Governor                                                                   |
| 2022/08/30 | 1795  | Fix building packages for armel                                                                                    |
| 2022/09/02 |       | Improve performance of the optional `machines` watchers                                                            |
| 2022/09/02 | 1802  | Set up the embedded NATS CLI using the correct inbox prefix                                                        |
| 2022/08/31 | 1798  | Allow provisioners to cleanly shutdown a server, thus opting it out of provisioning or being managed               |
| 2022/08/27 |       | Speed up leader elections                                                                                          |
| 2022/08/16 | 1785  | Do not read config or setup security framework for election file check                                             |
| 2022/08/08 | 1782  | Improve flag handling for the rpc builder command                                                                  |
| 2022/08/04 | 1779  | Work around breaking changes in NATS Server                                                                        |
| 2022/07/15 | 1768  | Improve processing of lifecycle events by implementing Stringer for event types                                    |
| 2022/08/03 |       | Support go1.18 as minimum version, support go 1.19                                                                 |
| 2022/08/03 |       | Release 0.26.1                                                                                                     |
| 2022/08/02 | 1774  | When generating a new ed25519 server seed, remove the invalidated JWT from disk                                    |
| 2022/08/02 | 1774  | Detect JWT and Server seed mismatches, trigger reprovision if possible                                             |
| 2022/08/01 |       | Upgrade `appbuilder` to `v0.3.0`                                                                                   |
| 2022/08/01 | 1722  | Allow in-process connections to bypass TLS, use them to configure Choria Streams                                   |
| 2022/07/07 | 1756  | Allow governors to control executions per period                                                                   |
| 2022/07/07 | 1755  | Add various election based tools for command execution, cron jobs and more, also various management tools          |
| 2022/07/10 | 1760  | Add a `gossip` watcher to publish regular to the Choria Broker, including basic service registration               |
| 2022/07/05 | 1712  | Switch to a new help template                                                                                      |
| 2022/06/27 | 1732  | Improved detection of STDIN avoiding some unexpected switches in discovery method and improving running under cron |
| 2022/06/27 | 1528  | Improve reliability of removing self-managed autonomous agents                                                     |
| 2022/06/27 | 1747  | Force gzip compression on Jammy debs to improve compatability with other distros and mirroring tools               |
| 2022/06/24 |       | End to end nightly installation tests on EL9                                                                       |
| 2022/06/24 | 1740  | Support signing JWT tokens using ed25519 tokens                                                                    |
| 2022/06/24 | 1740  | Refactor protocol and security layers to start work on version 2 of the network protocol                           |
| 2022/06/23 |       | Release 0.26.0                                                                                                     |
| 2022/06/21 | 1735  | Support Ubuntu 22.04 LTS                                                                                           |
| 2022/06/21 | 1735  | Debian packages will not have the distro as part of the package name to easy mirroring                             |
| 2022/06/15 | 1738  | KV autonomous agent watchers will now template parse the names of keys                                             |
| 2022/06/12 | 1725  | Exec autonomous agent watchers can now perform a fast - subject to splay - initial gather                          |
| 2022/06/12 | 1726  | Remove the deprecated Anonymous TLS server mode                                                                    |
| 2022/06/12 | 1722  | Prevent client permissions from being set on servers, not triggered in choria, was possible by using the libraries |
| 2022/06/11 | 1720  | Provisioning JWT can have user provided extensions to convey additional non-choria information to provisioner      |
| 2022/06/09 | 1712  | Various UX improvements to the help output                                                                         |
| 2022/06/08 | 1713  | Allow access to the broker system account via unverified TLS given a JWT with the system permission                |
| 2022/06/06 | 1703  | Adds `kv create` and `kv update`                                                                                   |
| 2022/06/06 | 1708  | Move to `fisk` CLI package, improve default help output verbosity                                                  |
| 2022/06/02 | 1704  | Support EL9                                                                                                        |
| 2022/05/26 | 1697  | Support subject mappings witin the Choria Brokers for partitioning                                                 |
| 2022/05/10 | 1665  | Introduce a new kind of Application by adopting the `appbuilder` project and extending it for Choria               |
| 2022/05/09 | 1663  | Adds a `semver` reply filter function                                                                              |
| 2022/05/28 | 1659  | Use core Go list of supported ciphersuites                                                                         |
| 2022/05/20 | 1654  | Prevent use of JWT tokens with incorrect caller ID                                                                 |
| 2022/05/14 | 1649  | Fix typo in generated code                                                                                         |
| 2022/05/12 | 1647  | Expand the `inventory` registration payload to include version, hash and auto agent information                    |
| 2022/05/11 | 1645  | Work around breaking change in nats.go related to KV access                                                        |
| 2022/05/08 | 1643  | Allow slow TTLs for leader elections                                                                               |
| 2022/05/06 | 1640  | Improve reliability of clean shutdowns                                                                             |
| 2022/04/05 | 1445  | Reject agents without a name or too small timeout                                                                  |
| 2022/04/05 | 1636  | Support skipping system stream management                                                                          |
| 2022/04/05 | 1591  | Use correct credentials when running `choria broker server check jetstream`                                        |
| 2022/04/05 | 1633  | Remove the Provisioner agent `release_update` action                                                               |
| 2022/03/31 | 1630  | UX improvements for `choria kv`                                                                                    |
| 2022/03/30 | 1628  | Use correct credentials when running `choria broker server check kv`                                               |
| 2022/03/29 | 1625  | When using the embedded `nats` cli allow a custom Choria configuration to be set                                   |
| 2022/03/22 | 1619  | Adds full end to end integration testing                                                                           |
| 2022/03/21 |       | Improve logging during initial connection establishment                                                            |
| 2022/03/17 | 1615  | Remove obsolete operating system distributions                                                                     |
| 2022/03/16 |       | Switch to go 1.18                                                                                                  |
| 2022/03/10 |       | Redact some passwords when logging                                                                                 |
| 2022/03/01 | 1608  | Improve hostname validation checks in flatfile discovery                                                           |
| 2022/02/25 |       | Release 0.25.1                                                                                                     |
| 2022/02/25 | 1604  | Fix startup on windows                                                                                             |
| 2022/02/23 |       | Release 0.25.0                                                                                                     |
| 2022/02/07 | 1590  | Support checking server JWT token validity                                                                         |
| 2022/01/18 | 1405  | Add 64 bit ARM packages                                                                                            |
| 2022/01/18 | 1576  | Allow custom builders to set the server service to auto start after install                                        |
| 2022/01/17 | 1573  | Compatibility fix for 32 bit builds                                                                                |
| 2022/01/17 | 1571  | Improve starting Choria Streams between reboots                                                                    |
| 2022/01/13 | 1568  | Improve `tool provision` so debugging custom provisioning targets is more reliable                                 |
| 2022/01/12 | 1561  | Correctly handle missing server configuration files when a custom provisioner is set                               |
| 2022/01/10 | 1555  | Ensure filters work with async requests in the `choria req` command                                                |
| 2021/12/17 | 1526  | Support trace logging of the embedded nats CLI                                                                     |
| 2021/12/16 | 1549  | Expand the `jwt` command to create other types of JWT and move to `choria jwt`                                     |
| 2021/12/16 | 1547  | Unify the `kv del` and `kv rm` commands                                                                            |
| 2021/12/07 | 1543  | Remove NATS Streaming Server support                                                                               |
| 2021/12/07 | 1541  | Improve `choria tool governor run` when the broker is down                                                         |
| 2021/11/29 | 1522  | Specifically use `choria broker run` to start the broker                                                           |
| 2021/11/24 | 1526  | Import the `nats` CLI tool into Choria under `choria broker`                                                       |
| 2021/11/19 | 1522  | Support enabling connection `nonce` feature allowing per connection private key validation                         |
| 2021/11/25 | 1522  | Extend provisioning agent to on board ed25519 seeds and process signed JWTs from the provisioner                   |
| 2021/11/12 | 1509  | Allow JWT clients to have permissions that can restrict access to Choria Streams related features                  |
| 2021/11/12 | 1509  | Extract all jwt handling code in all packages into a new `tokens` package                                          |
| 2021/11/09 | 1507  | Allow non TLS connections from both servers and clients in combination of AAA and Provisioner using JWTs           |
| 2021/10/28 | 1502  | Move to NATS official KV implementation, formalize Leader Election in Choria Broker                                |
| 2021/10/28 | 1499  | Avoid leaving some tempoary directories around in the archive watcher                                              |
| 2021/10/28 | 1495  | Allow succesfull KV operations that do not change data to transition autonomous agents                             |
| 2021/10/28 | 1494  | Relax identity validation in flatfile discovery to avoid rejecting some valid hostnames as identities              |
| 2021/10/27 | 1491  | Add `--senders` to `choria req` that shows only those replying identities                                          |
| 2021/10/26 | 1487  | Support for latest Cert Manager APIs                                                                               |
| 2021/10/25 | 1482  | Support tallying governor events                                                                                   |
| 2021/10/25 | 1483  | Allow custom loggers to be passed to Choria and avoid changing settings of the default logrus logger               |
| 2021/10/25 | 1480  | Support tallying wildcard components rather than just a single component                                           |
| 2021/10/15 |       | Add SPDX License Identifier and Copyright to source files                                                          |
| 2021/10/15 | 1475  | Support `stdout` and `stderr` as logging destinations in addition to `discard` and a file name                     |
| 2021/10/14 | 1472  | Show additional `mco choria show_config` style information in `choria tool config`                                 |
| 2021/10/14 |       | Change docker base to AlmaLinux                                                                                    |
| 2021/10/12 | 1241  | Refactor DDL resolution, support querying Choria Registry for unknown DDLs                                         |
| 2021/10/07 | 1462  | Adds a new `machines` watcher to manage Choria Autonomous Agents, not enabled by default                           |
| 2021/10/06 | 1459  | Adds a new `archive` watcher to manage `tgz` files, not enabled by default                                         |
| 2021/10/05 | 1454  | Support asserting provisioning state in the health check plugin                                                    |
| 2021/10/05 | 1455  | Ignore machines with `-temp` name suffix and the `tmp` directory                                                   |
| 2021/09/27 | 1446  | Compatibility fix for latest NATS Server code regarding dynamic limits                                             |
| 2021/09/22 | 1438  | Allow `choria scout watch` to show only state changes                                                              |
| 2021/09/22 | 1438  | Add a CLI API for managing KV buckets                                                                              |
| 2021/09/20 |       | Release 0.24.0                                                                                                     |
| 2021/09/19 | 1428  | Adds a helper to assist in creation of Governors from automation tools                                             |
| 2021/09/17 | 1426  | Do not attempt to also load embedded Autonomous Agents from disk                                                   |
| 2021/09/14 | 1418  | Allow provisioning of Action Policies and Open Policy Agent Policies via Choria Provisioner                        |
| 2021/09/13 | 1415  | Support listing known Governors                                                                                    |
| 2021/09/13 | 1413  | Do not create unconfigured Governors when viewing a non existing Governor                                          |
| 2021/09/13 | 1411  | Add `--force` / `-f` to `choria governor add`                                                                      |
| 2021/09/09 | 1407  | Create the `plugin.choria.machine.store` directory if it does not exist                                            |
| 2021/09/06 | 140   | Do not update file mtime on skipped checks in the File watcher                                                     |
| 2021/09/06 | 1401  | Add a `splay` option to the Timer Watcher                                                                          |
| 2021/09/03 | 1397  | Handle JSON data in data better in Autonomous Agent data layer allowing for nested lookups                         |
| 2021/09/02 | 1388  | Various refactors of Debian packages to behave more consistently with RedHat startup/restart flows                 |
| 2021/09/02 | 1393  | Fix logging of embedded NATS Server to Choria logs                                                                 |
| 2021/09/01 | 1386  | Introduce a faster broadcast discovery timeout using sliding windows, behind a opt-in setting                      |
| 2021/08/31 | 1384  | Allow Autonomous Agents to be compiled into the server as plugins                                                  |
| 2021/08/31 | 1377  | Initial support for performing AAA Server signing requests via Choria Services rather than HTTPS                   |
| 2021/08/27 | 1377  | Internal refactoring to improve cross/cyclic package import problems                                               |
| 2021/08/24 |       | Release 0.23.0                                                                                                     |
| 2021/08/24 |       | Support Debian 11                                                                                                  |
| 2021/08/23 | 1367  | Enable the `choria_provision` agent when provisioning is supported                                                 |
| 2021/08/18 | 1359  | Support sorting `choria req` output by identity using `--sort`                                                     |
| 2021/08/18 | 1358  | Ensure SSL Cache is created if needed during provisioning                                                          |
| 2021/08/18 | 1357  | Correctly enter provisioning with a configuration file and without a Puppet installation                           |
| 2021/08/17 | 1355  | Support receiving private keys from the provisioner, protected using Curve 25519 ECDH shared secrets               |
| 2021/08/16 | 1353  | Ensure no responses list and unexpected responses list always prints, capped to 200 nodes                          |
| 2021/08/11 | 1344  | Fix setting workers and expr filter on generated clients                                                           |
| 2021/08/10 | 1342  | Include the Public Key in the CSR reply, add data type hints to the provisioner DDL and update client              |
| 2021/08/09 | 1331  | Include the time a RPC Reply was generated in the reply                                                            |
| 2021/08/09 | 1337  | Generated clients can accept a Choria Framework, avoiding config loading etc                                       |
| 2021/08/09 | 1335  | Support entering provisioning mode when the supplied `server.conf` does not exist                                  |
| 2021/08/09 | 1333  | Disable RPC Auth during provisioning mode                                                                          |
| 2021/08/04 | 1326  | Rename the `jetstream` adapter to `choria_streams`                                                                 |
| 2021/08/03 | 1324  | Allow compiled-in Go agents to access the Submission system                                                        |
| 2021/08/03 | 1321  | Improve the broker shutdown process to cleanly shut down Choria Streams                                            |
| 2021/08/03 | 1319  | Use correct Choria reply subjects when interacting with the Streams API                                            |
| 2021/08/02 | 1316  | Extend the RPC Reply structure to include what action produced the data                                            |
| 2021/08/02 | 1314  | Support Asynchronous Request mode in generated Go clients                                                          |
| 2021/07/25 | 1310  | Export certificate expiry time in Choria status files, support checking from CLI and Scout                         |
| 2021/07/19 | 1291  | Support templates in Exec Watcher `cmd`, `env` and `governor`                                                      |
| 2021/07/12 | 1291  | Expose `kv` data to the Autonomous Agent data system                                                               |
| 2021/07/12 | 1291  | Add a Choria Key-Value Store accessible using `choria kv` and a new `kv` Autonomous Agent Watcher                  |
| 2021/07/12 | 1291  | Allow Exec Watchers to access node facts                                                                           |
| 2021/07/12 | 1291  | Add a Autonomous Agent level data store, allow Exec Watchers to gather and store data in a Auto Agent              |
| 2021/07/09 | 1289  | Additional Prometheus statistics for Choria Streams                                                                |
| 2021/07/05 | 1276  | Support Governors in the Exec Autonomous Agent watcher                                                             |
| 2021/07/02 | 1276  | Introduce `choria governor` for network wide concurrency control                                                   |
| 2021/06/30 | 1277  | Support PKCS8 containers                                                                                           |
| 2021/06/23 | 1273  | Introduce Choria Submission to allow messages to be placed into Streams via Choria Server                          |
| 2021/06/21 | 1272  | Use default client-like resolution to find brokers in the JetStream adapter when no urls are given                 |
| 2021/06/09 | 1042  | Rate limit fast transitions in autonomous agents                                                                   |
| 2021/06/09 | 1264  | Allow a random sleep at the start of schedules for the Schedule watcher                                            |
| 2021/06/06 | 1259  | Allow the default client suffix to be set at compile time (eg. rip.mcollective user id)                            |
| 2021/06/06 | 1258  | Allow the default collective to be set at compile time                                                             |
| 2021/06/03 | 1256  | Fail when a client cannot determine its identity                                                                   |
| 2021/05/11 | 1250  | Improve sorting of `choria inventory` columns                                                                      |
| 2021/04/28 | 1246  | Adds a `choria login` command that supports delegating to `choria-login` in `PATH`                                 |
| 2021/04/27 | 1241  | Initial implementation of the `choria_registry` service agent                                                      |
| 2021/04/27 | 1243  | Support Websockets for connectivity from Leafnodes and Choria Server to Choria Broker, also Go clients             |
| 2021/04/23 | 1234  | Allow the Choria Server to run in an Services-Only mode                                                            |
| 2021/04/23 | 1238  | Improve some core DDLs with better type hints                                                                      |
| 2021/04/22 |       | Release 0.22.0                                                                                                     |
| 2021/04/22 | 1234  | Initial support for Service Agents                                                                                 |
| 2021/04/21 | 1232  | Autonomous Agent transitions now support a human friendly description                                              |
| 2021/04/19 | 1227  | Import the provisioning agent into this code base since it's now always compiled in                                |
| 2021/04/16 | 1222  | Create `choria plugin doc` and move `tool generate` to `plugin generate`                                           |
| 2021/04/15 | 1220  | Handle filter expressions that are not obviously boolean better                                                    |
| 2021/04/13 | 1216  | Improve startup logs when skipping agents in specific providers                                                    |
| 2021/04/08 | 1213  | Increase leafnode authentication timeout                                                                           |
| 2021/04/08 | 1211  | Improve randomness of limited targets                                                                              |
| 2021/04/07 | 1207  | Support wider duration specification by supporting week, month, year etc                                           |
| 2021/04/05 | 1204  | Enable new Go based action policy by default                                                                       |
| 2021/04/01 | 1201  | Support the old `boolean_summary` aggregator and generic output name remapping in summary aggregator               |
| 2021/03/30 | 1195  | Default to the `choria` account for leafnodes                                                                      |
| 2021/03/30 | 1197  | Improve consistency of time durations in ping output                                                               |
| 2021/03/30 | 1195  | Fix ordering of leafnode and acounts setup                                                                         |
| 2021/03/29 | 1193  | JetStream Adapter can publish to wildcard streams with per identity subjects                                       |
| 2021/03/29 | 1189  | Use correct target for registration messages                                                                       |
| 2021/03/29 |       | Release 0.21.0                                                                                                     |
| 2021/03/26 | 1189  | Add a new registration plugin that sends the running inventory rather than file contents                           |
| 2021/03/25 | 1187  | Support enabling listening `pprof` port                                                                            |
| 2021/03/23 | 1185  | Fix validation for integers in the DDLs                                                                            |
| 2021/03/23 | 1183  | Fail `choria facts` when no nodes match supplied filters                                                           |
| 2021/03/19 | 1180  | Restore the data plugin report in rpcutil#inventory                                                                |
| 2021/03/19 | 1178  | Do not send the filter verbatim in `choria req`                                                                    |
| 2021/03/18 | 1175  | Add a client specific `TLSConfig()`, improve adapters and federation support for legacy certs                      |
| 2021/03/18 | 1173  | Create a `choria` account in NATS, move all connections there, enable `system` account                             |
| 2021/03/17 | 1170  | Correctly calculate advertise URL                                                                                  |
| 2021/03/10 | 1165  | Improve support for Clustered JetStream                                                                            |
| 2021/03/10 | 1161  | Add a `machine_state` data plugin                                                                                  |
| 2021/03/05 | 1156  | Support retrieving a single choria autonomous agent state using choria_util                                        |
| 2021/03/02 | 1154  | Support building ppc64le EL7 and EL8 RPMs                                                                          |
| 2021/03/02 | 1152  | Improve ping response calculations in federated networks                                                           |
| 2021/03/01 | 1150  | Avoid unnecessary warning level logs                                                                               |
| 2021/02/23 |       | Drop support for Enterprise Linux 6 due to go1.16                                                                  |
| 2021/02/22 | 1147  | Correctly detect stdin discovery                                                                                   |
| 2021/02/17 | 1145  | Improve stability of `choria scout watch`                                                                          |
| 2021/02/03 |       | Release 0.20.2                                                                                                     |
| 2021/02/03 | 1140  | Ensure logging doesn't happen at warn level                                                                        |
| 2021/02/03 |       | Release 0.20.1                                                                                                     |
| 2021/02/03 | 1140  | Ensure that only client/server connections use no SAN TLS work around, not brokers                                 |
| 2021/02/03 |       | Release 0.20.0                                                                                                     |
| 2021/02/02 | 1136  | Improve progress bars on small screens                                                                             |
| 2021/02/02 |       | Sort classes tags in discovery command and elsewhere                                                               |
| 2021/02/01 | 1074  | Initial support for Data Providers, add `choria`, `scout`, `config_item` providers                                 |
| 2021/01/29 | 1123  | Perform identity-only discovery optimization in `broadcast` and `puppetdb` discovery methods                       |
| 2021/01/29 | 1121  | Add a `--silent` flag to `choria discover` to improve script integration                                           |
| 2021/01/28 | 1060  | Support go 1.15 by putting in work around to support Puppet SAN free TLS certificates                              |
| 2021/01/28 |       | Add a bash completion script in `choria completion` in addition to current ZSH support                             |
| 2021/01/24 | 1113  | Adds a new `inventory` discovery method                                                                            |
| 2021/01/23 | 1110  | Improve SRV handling when trying to find PuppetDB host                                                             |
| 2021/01/23 | 1098  | Improve `choria tool config` to show config files and active settings                                              |
| 2021/01/22 | 1102  | Ensure we discover `rpcutil` in the `discover` command, improves PuppetDB integration                              |
| 2021/01/20 | 751   | Add project level Choria configuration                                                                             |
| 2021/01/21 | 1098  | Allow options to be passed to discovery methods using `--discovery-option`                                         |
| 2021/01/18 | 1081  | Support flatfile discovery from json, yaml, stdin and improve generated clients                                    |
| 2021/01/16 | 1092  | Add the `external` discovery method                                                                                |
| 2021/01/18 | 1072  | Performance improvements for expr expression handling                                                              |
| 2021/01/14 | 281   | Improve identity handling when running on windows, non root and other situations                                   |
| 2021/01/13 | 1089  | Support request chaining in the req command                                                                        |
| 2021/01/13 |       | Release 0.19.0                                                                                                     |
| 2020/01/12 | 1086  | Create a `choria facts` command                                                                                    |
| 2020/01/12 | 1084  | Support full GJSON Path Syntax in rpcutil#get_fact, fix a crash on map data in aggregators                         |
| 2020/01/10 | 1081  | Standardise filter and discovery CLI options                                                                       |
| 2020/01/10 | 1074  | Support compound filters using `expr`                                                                              |
| 2020/01/09 | 1076  | Improve support for HTTPS servers discovered by SRV records by stripping trailing `.` in names                     |
| 2021/01/09 | 1074  | Basic support for Data plugin DDLs                                                                                 |
| 2021/01/09 | 1072  | Add `expr` based client-side filtering of RPC results                                                              |
| 2021/01/08 | 1068  | Improve support for the `color` option and disable it by default on windows                                        |
| 2021/01/07 | 1064  | Calculate `choria ping` times from the moment before publish and report overhead                                   |
| 2021/01/07 | 1062  | Support parsing nagios format Perfdata as output format for the metric watcher                                     |
| 2020/12/29 | 1055  | Report the certificate fingerprint when doing `choria enroll` for Puppet CA                                        |
| 2020/12/28 | 1051  | Add `choria discover`                                                                                              |
| 2020/12/27 | 1049  | Generated clients has a PuppetDB name source                                                                       |
| 2020/12/27 | 1049  | rpc client will now honor the DefaultDiscoveryMethod setting for all clients                                       |
| 2020/12/27 | 1049  | Add `--dm` to the `choria req` command to switch discovery method                                                  |
| 2020/12/27 | 1049  | Add a PuppetDB discovery method                                                                                    |
| 2020/12/27 | 1047  | Create generated clients for `rpcutil`, `scout` and `choria_util` in `go-choria/client`                            |
| 2020/12/26 | 1045  | Add `choria inventory`                                                                                             |
| 2020/12/16 | 1017  | Avoid listening and registering with mDNS when Homekit is not used                                                 |
| 2020/12/12 | 1038  | Add a `choria_status` Nagios builtin allowing Choria to health checks from Scout                                   |
| 2020/12/09 | 1035  | Ignore case when matching against configuration management classes                                                 |
| 2020/12/09 | 1035  | Ignore case when doing fact matching                                                                               |
| 2020/12/08 | 1030  | Allow Autonomous Agent Watchers to be plugins, convert all core ones to plugins                                    |
| 2020/12/03 |       | Major code cleanups and and test coverage for the Autonomous Agents                                                |
| 2020/11/29 | 1009  | Perform DNS lookups on every initial reconnect retry                                                               |
| 2020/11/28 | 1007  | Add a `metrics` Autonomous Agent watcher that can fetch and publish metrics                                        |
| 2020/11/27 | 1006  | Use new JetStream features to improve retrieval of event history                                                   |
| 2020/11/25 |       | Release 0.18.0                                                                                                     |
| 2020/10/21 | 999   | Add a timer watcher that changes state after a time expires                                                        |
| 2020/10/21 | 999   | Support creating Apple Homekit buttons in Autonomous Agents                                                        |
| 2020/09/28 |       | Release 0.17.0                                                                                                     |
| 2020/09/04 | 989   | Add a generic shell completion helper and support ZSH completion                                                   |
| 2020/08/25 | 987   | Support NATS Leafnodes to extend the Choria Broker in a TLS free way specifically usable by AAA clients            |
| 2020/08/03 | 982   | Scout checks can have annotations that are published in events                                                     |
| 2020/08/03 | 920   | Add `choria scout maintenance` and `choria scout resume` commands                                                  |
| 2020/08/01 | 920   | Add a `choria scout trigger` command that triggers an immediate check and associated events                        |
| 2020/08/01 | 977   | Generated clients can now set a progress bar                                                                       |
| 2020/07/30 | 975   | Prevent int overflow in time fields in some Scout events                                                           |
| 2020/07/26 | 920   | Add a `--table` option to `choria req` and a new formatter in generated clients                                    |
| 2020/07/26 | 920   | Add a `choria scout status` command that can show all checks on a node                                             |
| 2020/07/24 | 968   | Improve the history presented in Scout events                                                                      |
| 2020/07/22 | 966   | Remove the concept of a site wide Gossfile                                                                         |
| 2020/07/21 | 964   | Allow multiple Gossfiles and multiple Goss checks                                                                  |
| 2020/07/18 |       | Release 0.16.0                                                                                                     |
| 2020/07/18 | 960   | Add a `choria scout watch` command                                                                                 |
| 2020/07/17 | 957   | Restore the ability for DDLs to declare display formats for aggregate outputs                                      |
| 2020/07/16 | 948   | Support performing `goss` validation in the `nagios` autonomous agent                                              |
| 2020/07/15 | 842   | Avoid zombies when Ruby agents exceed their allowed run time                                                       |
| 2020/07/09 | 944   | Extract the generic result display logic from `choria req` into a reusable package                                 |
| 2020/07/09 | 942   | Include a snapshot of recent check states in published check events                                                |
| 2020/07/08 | 939   | Improve using the supplied logger in generated clients                                                             |
| 2020/07/08 | 938   | Add helpers to parse complex data in generated clients                                                             |
| 2020/07/08 | 937   | Generated clients perform 2 discoveries per request                                                                |
| 2020/07/07 | 935   | Release packages for Ubuntu Focal (20.04 LTS)                                                                      |
| 2020/07/07 | 932   | Fix targeting a specific sub collective in the `req` command                                                       |
| 2020/07/07 | 928   | Add a new `scout` agent and Golang client                                                                          |
| 2020/07/03 | 920   | Initial work on a Scout framework towards building a monitoring related distribution                               |
| 2020/07/01 |       | Release 0.15.0                                                                                                     |
| 2020/06/29 | 913   | Support preparing for shutdown by closing connections and emiting shutdown events when embedded                    |
| 2020/06/26 | 895   | Support NATS JetStream Streaming Server in Choria Broker                                                           |
| 2020/06/24 | 907   | Support arm5 and 7 Debian packages                                                                                 |
| 2020/06/20 | 895   | Support Nagios compatible plugins in the new `nagios` autonomous agent watcher                                     |
| 2020/06/16 | 893   | Server instances embedded in other software can now be shutdown using `Shutdown()`                                 |
| 2020/06/15 | 887   | Track nodes expired by maintenance in the tally helper                                                             |
| 2020/06/13 |       | Improve FQDN resolution when running in a kubernetes pod                                                           |
| 2020/06/12 | 879   | Allow the public name of the network broker to be configured                                                       |
| 2020/06/12 | 877   | Support cert-manager.io as security provider                                                                       |
| 2020/06/07 | 865   | Correctly handle provisioning by SRV domain                                                                        |
| 2020/06/07 | 863   | Allow provisioning brokers to have user/password authentication                                                    |
| 2020/05/14 | 860   | Perform backoffs between reconnects to the network broker                                                          |
| 2020/04/22 | 857   | Cosmetic improvements to windows packages                                                                          |
| 2020/04/19 |       | Release 0.14.0                                                                                                     |
| 2020/04/16 | 854   | Correctly report insecure builds                                                                                   |
| 2020/04/07 | 852   | Install `choria` binary in /usr/bin and not /usr/sbin                                                              |
| 2020/03/25 | 846   | Various improvements to generated RPC clients                                                                      |
| 2020/03/24 | 844   | Export facts to external agents                                                                                    |
| 2020/03/22 | 801   | Expose statistics for NATS Leafnodes                                                                               |
| 2020/03/16 | 840   | Improve formatting of node lists at the end of requests                                                            |
| 2020/03/11 | 687   | Support enforcing the use of filters on all RPC requests using `plugin.choria.require_client_filter`               |
| 2020/03/03 | 834   | Add Debian Buster support                                                                                          |
| 2020/02/17 | 831   | Cache transport messages when doing batched requests to improve pkcs11 integration                                 |
| 2020/02/13 | 827   | Ensure agent filter is added when discovering nodes                                                                |
| 2020/02/08 | 817   | Add `choria tool config` to view configuration paramters and current values                                        |
| 2020/02/08 | 814   | Set `PATH` when calling external agents                                                                            |
| 2020/02/05 | 794   | Merge `go-lifecycle` into `go-choria`                                                                              |
| 2020/02/05 | 794   | Merge `go-protocol`, `go-security`, `mcorpc-agent-provider` and `go-config` into `go-choria`                       |
| 2020/02/05 | 794   | Merge `go-confkey`, `go-validator`, `go-puppet`, `go-network-broker` and `go-srvcache` into `go-choria`            |
| 2020/01/30 |       | Update to CloudEvents 1.0.0                                                                                        |
| 2020/01/23 | 774   | Support logging to Windows Event log                                                                               |
| 2020/01/23 | 772   | Support running as a Windows service                                                                               |
| 2020/01/17 | 769   | Add basic Windows pacakges                                                                                         |
| 2020/01/16 |       | Support use selectable SSL Ciphers using `plugin.security.cipher_suites` and `plugin.security.ecc_curves`          |
| 2020/01/12 |       | Release 0.13.1                                                                                                     |
| 2019/12/25 | 758   | Extract RPC reply rendering to the mcorpc package-agent-provider                                                   |
| 2019/12/23 | 754   | Extract parts of the filter parsing logic to the `protocol` package                                                |
| 2019/12/15 | 746   | Support remote request signers such as `aaasvc`                                                                    |
| 2019/12/09 | 743   | Support generating Go clients using `choria tool generate client`                                                  |
| 2019/12/05 |       | Release 0.13.0                                                                                                     |
| 2019/12/05 | 737   | Add a tech preview JetStream adapter                                                                               |
| 2019/12/04 | 731   | Switch to CloudEvents v1.0 format for lifecycle events and machine events                                          |
| 2019/12/02 | 709   | Build RHEL 8 packages nightly and on release                                                                       |
| 2019/12/02 | 548   | Improve startup when embedding the server in other programs                                                        |
| 2019/11/29 | 724   | Improve stability on a NATS network with Gateways                                                                  |
| 2019/11/28 | 720   | Improve the calculations of total request time in the `choria req` command                                         |
| 2019/11/21 | 710   | Support Synadia NGS as a NATS server for Choria                                                                    |
| 2019/10/26 | 705   | Add `choria tool jwt` to create provisioning tokens                                                                |
| 2019/10/25 | 705   | Allow a JWT file to configure provisioning behavior and enable provisioning in the FOSS binary                     |
| 2019/10/14 | 703   | Allow `choria req` output to be saved to a file                                                                    |
| 2019/10/01 | 700   | Force convert a DDL from JSON on the CLI without prompts                                                           |
| 2019/09/20 |       | Release 0.12.1                                                                                                     |
| 2019/09/19 |       | Support Authorization and External Agents via latest MCORPC provider                                               |
| 2019/09/16 | 681   | Allow agents to associate with specific agent providers using the `provider` field in metadata                     |
| 2019/09/12 | 678   | Support generating Ruby and JSON DDL files using `choria tool generate ddl`                                        |
| 2019/09/09 |       | Release 0.12.0                                                                                                     |
| 2019/09/09 |       | Broker based on NATS 2.0 via `go-network-broker` version `1.3.1`                                                   |
| 2019/09/07 | 670   | Improve the output from `choria ping --graph`                                                                      |
| 2019/09/06 | 664   | Add a pkcs11 security provider                                                                                     |
| 2019/09/04 | 663   | Add a `choria req` tool to eventually replace `mco rpc`                                                            |
| 2019/08/09 | 652   | Write init scripts to the correct location on RHEL                                                                 |
| 2019/07/24 | 642   | Show dependencies compiled into the binary in `choria buildinfo`                                                   |
| 2019/07/15 | 632   | Decrease memory use in adapters by lowering the work queue length                                                  |
| 2019/06/27 | 621   | Choria Provisioner is now a proper plugin                                                                          |
| 2019/06/27 | 623   | Support `agents.ShouldActivate()` checks when loading agents                                                       |
| 2019/06/26 | 617   | Support NATS 2.0 credentials and user/password                                                                     |
| 2019/06/26 | 617   | Fix `choria ping`                                                                                                  |
| 2019/06/12 |       | Release 0.11.1                                                                                                     |
| 2019/04/20 |       | Support email SANs in client certificates via `go-security` `0.4.2`                                                |
| 2019/06/11 | 609   | Verify that only known transitions and states are mentioned in the machine specification                           |
| 2019/06/11 | 607   | Ensure the machine directory is in the `PATH`                                                                      |
| 2019/05/30 | 605   | Fix `environment` handling for exec watchers                                                                       |
| 2019/05/29 | 602   | Ensure machines are runable on the CLI                                                                             |
| 2019/05/29 | 599   | Support run-once exec watchers by setting `interval=0`                                                             |
| 2019/05/29 | 597   | Do not manage Autonomous Agents in provisioning mode                                                               |
| 2019/05/28 | 591   | Add a `scheduler` watcher for Autonomous Agents                                                                    |
| 2019/05/27 |       | Release 0.11.0                                                                                                     |
| 2019/05/23 |       | Log discovery requests in a similar manner to RPC requests via `mcorpc-agent-provider` `0.4.0`                     |
| 2019/05/23 |       | Fix puppet provider support for `SecurityAlwaysOverwriteCache` via `go-security` `0.4.0`                           |
| 2019/05/23 |       | Improve excessive logging when privilged certificates are used via `go-security` `0.4.0`                           |
| 2019/05/23 |       | Only write certificates on change if `SecurityAlwaysOverwriteCache` is set via `go-security` `0.4.0`               |
| 2019/05/22 | 554   | Retry SRV lookups on reconnect attempts                                                                            |
| 2019/05/27 | 563   | Support Choria Autonomous Agents                                                                                   |
| 2019/03/21 | 557   | Force puppet environment to `production` to avoid failures about missing environment directories                   |
| 2019/03/19 | 557   | Improve error messages logged when invoking `puppet` to retrieve setting values fail                               |
| 2019/03/15 | 555   | Add a basic utility to assist with creating deep monitoring `choria tool status`                                   |
| 2019/03/04 |       | Release 0.10.1                                                                                                     |
| 2019/02/25 |       | Resolve broker instability on large networks via `go-network-broker#19`                                            |
| 2019/01/23 |       | Release 0.10.0                                                                                                     |
| 2019/01/17 |       | Various fixes to privileged security certificate handling via `go-security` release `0.3.0`                        |
| 2019/01/17 |       | Allow limiting clients to sets of IPs via `go-network-broker#12`                                                   |
| 2019/01/09 | 534   | Ensure the server status file is world readable                                                                    |
| 2019/01/07 | 532   | Force exit even when worker routines are not done after `soft_shutdown_timeout`, default 2 seconds                 |
| 2019/01/05 | 530   | Further fixes to avoid concurrent hash access panics for golang client code                                        |
| 2019/01/03 | 524   | Include the server version when creating life cycle events                                                         |
| 2018/12/27 | 521   | Improve `alive` event spread by sleeping for up to a hour for initial publish                                      |
| 2018/12/27 | 519   | Expose `security.Validate` to users of the go framework                                                            |
| 2018/12/27 |       | Release 0.9.0                                                                                                      |
| 2018/12/26 |       | Fix reboot splay time when doing self updates via `provisioning-agent#67`                                          |
| 2018/12/26 |       | Increase `choria_util` agent timeout to facilitate slow facter runs via `mcorpc-agent-provider#36`                 |
| 2018/12/26 | 515   | Cache facter lookups                                                                                               |
| 2018/12/21 | 510   | Publish new `alive` life cycle events every hour                                                                   |
| 2018/12/19 |       | support `~/.choriarc` and `/etc/choria/client.conf` for client configs                                             |
| 2018/12/19 |       | Report protocol security and connector TLS in `choria_util#info` via `mcorpc-agent-provider#33`                    |
| 2018/12/19 | 501   | Allow default configuration values to be mutated at startup using a plugin                                         |
| 2018/12/07 | 495   | Allow server status to be written during provision mode                                                            |
| 2018/11/30 |       | Release 0.8.0                                                                                                      |
| 2018/11/28 | 489   | Avoid a panic that affected clients written in Go when closing connections to the broker                           |
| 2018/11/23 |       | Improve backward compatibility when handling slashes in regex for allowed certs (go-security#22)                   |
| 2018/11/23 | 485   | Fail gracefully in the `ping` app when the configuration is not present                                            |
| 2018/11/20 | 483   | Resolve a client subscription leak by unsubscribing on context cancellation                                        |
| 2018/11/15 |       | When provisioning is compiled in - support self updating using `go-updater` (provisioning-agent#53)                |
| 2018/11/14 | 476   | Allow the SSL cache to always be written via `plugin.security.always_overwrite_cache`                              |
| 2018/11/02 | 473   | Support running Choria Server in a namespace on Enterprise Linux via a COMMAND_PREFIX in the init script           |
| 2018/10/24 | 467   | Support writing server status regularly                                                                            |
| 2018/10/27 | 470   | Switch to `github.com/gofrs/uuid` for UUID generation                                                              |
| 2018/10/18 |       | Release 0.7.0                                                                                                      |
| 2018/10/02 | 462   | Allow custom packages to supply their own sysv init start order                                                    |
| 2018/09/18 | 458   | Update network broker to 1.1.0 which includes `gnatsd` 1.3.0                                                       |
| 2018/09/17 | 456   | Provisioner Target plugins now have a context in their calls so they can do internal retries                       |
| 2018/09/15 | 447   | Create a single plugin interface that supports many types of plugin                                                |
| 2018/09/11 | 444   | Set ulimits for the broker appropriately for 50 000 connections                                                    |
| 2018/09/02 | 430   | Allow agents to publish lifecycle events                                                                           |
| 2018/08/31 | 428   | Add a CLI tool to view provisioning broker decisions - `tool provisioner`                                          |
| 2018/08/29 | 426   | Correctly compiled servers will enter provisioning mode when the configuration file is missing entirely            |
| 2018/08/27 |       | Release 0.6.0                                                                                                      |
| 2018/08/26 |       | Update provisioning agent dependency, allows `restart` when not in provisioning mode if a token is set             |
| 2018/08/26 | 422   | publish a shutdown lifecycle event on clean shutdowns                                                              |
| 2018/08/25 | 419   | Add `tool event`, `tool sub` and `tool pub` commands to the CLI                                                    |
| 2018/08/24 | 416   | Publish a startup lifecycle event on startup                                                                       |
| 2018/08/22 | 411   | Expose the version to Prometheus as `choria_build_info`                                                            |
| 2018/08/22 | 412   | Attempt to find the FQDN via DNS lookups before calling Puppet when the hostname is incomplete                     |
| 2018/08/16 | 408   | Add a plugin to assist with resolving the provisioning mode target brokers                                         |
| 2018/08/10 | 402   | Ensure provisioning mode is active only for the server and not client invocations                                  |
| 2018/08/09 |       | Release 0.5.1                                                                                                      |
| 2018/08/09 | 403   | Ensure insecure provisioning mode works on non puppet nodes                                                        |
| 2018/08/03 | 398   | Support an auth token during provisioning                                                                          |
| 2018/08/02 | 394   | Support a fact source during provisioning mode                                                                     |
| 2018/08/02 | 394   | Fix registration handling when in provisioning mode                                                                |
| 2018/07/31 | 390   | Avoid leaking metrics in long running clients that make many new client instances                                  |
| 2018/07/20 |       | Release 0.5.0                                                                                                      |
| 2018/07/13 | 382   | Improve mcollective compatibility by supporting shallow merges of multiple fact files                              |
| 2018/07/12 | 379   | Increase the NATS Stream Adapter work channel size to function on large networks                                   |
| 2018/07/12 | 377   | When adapting Choria messages to NATS Streams include the Choria RequestID                                         |
| 2018/07/12 | 375   | Ensure all loggers are configured with the correct level and format                                                |
| 2018/07/11 |       | Disable full JSON schema validation by default due to performance concerns (go-protocol#23)                        |
| 2018/07/11 |       | Update `gnatsd` to `1.2.0` to improve stability at >30k nodes when clustered (go-network-broker#6)                 |
| 2018/07/11 | 373   | Support Ubuntu 18.04                                                                                               |
| 2018/07/11 | 361   | When embedding the Choria Server initial startup errors can now be detected and handled                            |
| 2018/07/11 | 362   | When embedding the Choria Server one can now prevent certain agents from loading                                   |
| 2018/07/11 | 366   | Consult `/etc/choria/client.cfg` and `~/.choria` in addition to mcollective locations                              |
| 2018/07/03 | 359   | Resolve a go routine leak when using the connector in a long running client                                        |
| 2018/06/26 | 353   | Handle connection errors in NATS Streaming brokers in the Adapters, require NATS Streaming >= `0.10.0`             |
| 2018/06/18 | 346   | Add a high performance, filtering capable basic network validation CLI `choria ping`                               |
| 2018/06/15 | 343   | Resolve the merging of identity & certname concepts that erroneously happened during the security refacor          |
| 2018/06/14 | 341   | Ensure non root users - like clients - get a username based certname not FQDN one                                  |
| 2018/06/07 | 336   | Fix the setting that allows disabling broker order randomization                                                   |
| 2018/06/06 | 333   | Randomize broker connections by default                                                                            |
| 2018/06/06 | 331   | Add a short grace period to clock checks to allow for real world realities wrt synced clocks                       |
| 2018/05/31 |       | Release 0.4.0                                                                                                      |
| 2018/05/29 | 320   | Make the enroll process more robust in the face of interruptions                                                   |
| 2018/05/23 | 308   | Fix running `choria buildinfo`                                                                                     |
| 2018/05/23 | 309   | Create `go-security` package with the Choria security providers for reuse in other eco system projects             |
| 2018/05/22 | 306   | Fix registration feature when running in insecure mode                                                             |
| 2018/05/18 | 302   | Include a hint when the source data for registration changed                                                       |
| 2018/05/16 |       | Release 0.3.0                                                                                                      |
| 2018/05/08 | 287   | Create the concept of a Security Provider and create providers for Puppet and File, add `choria enroll`            |
| 2018/05/03 | 284   | On systemd based distributions ensure that upgrading choria with Puppet is more reliable                           |
| 2018/04/25 | 271   | Log rotation for `choria-*.log` which covers audit, ruby and more                                                  |
| 2018/04/25 | 267   | Ensure that the ruby shim based agents have access to the correct request time                                     |
| 2018/04/24 |       | Release 0.2.0                                                                                                      |
| 2018/04/23 | 243   | Create a compatibility framework for MCollective Agents written in Ruby                                            |
| 2018/04/23 | 252   | Avoid logrotate errors when the package was installed but choria never ran                                         |
| 2018/04/09 | 240   | When facter is available use it to determine the FQDN to improve default behavior on debian like systems           |
| 2018/04/09 | 236   | Allow `nats://host:port` and `host:port` to be used when referencing brokers                                       |
| 2018/04/09 | 235   | Detect empty initial server list when starting federation brokers                                                  |
| 2018/03/29 | 229   | Surface more NATS internal debug logs as notice and error                                                          |
| 2018/03/29 | 228   | Increase TLS timeouts to 2 seconds to improve functioning over latency and heavily loaded servers                  |
| 2018/03/26 | 199   | Do not use HTTP to fetch internal NATS stats                                                                       |
| 2018/03/26 | 220   | Update gnats and go-nats to latest versions                                                                        |
| 2018/03/26 | 222   | Allow the network broker write deadline to be configured                                                           |
| 2018/03/23 | 218   | Avoid rotating empty log files and ensure the newest log is the one being written too                              |
| 2018/03/21 |       | Release 0.1.0                                                                                                      |
| 2018/03/08 | 208   | Improve compatibility with MCollective Choria by not base64 encoding payloads                                      |
| 2018/03/08 | 207   | Ensure the filter is valid when creating `direct_request` messages                                                 |
| 2018/03/07 | 204   | Support writing a thread dump to the OS temp dir on receiving SIGQUIT                                              |
| 2018/03/07 | 202   | Do not rely purely on `PATH` to find `puppet`, look in some standard paths as well                                 |
| 2018/03/06 |       | Release 0.0.11                                                                                                     |
| 2018/03/06 | 198   | Reuse http.Transport used to fetch gnatsd statistics to avoid a leak on recent go+gnatsd combination               |
| 2018/03/05 |       | Release 0.0.10                                                                                                     |
| 2018/03/05 | 194   | Revert `gnatsd` to `1.0.4`, upgrade Golang to `1.10`                                                               |
| 2018/03/05 |       | Release 0.0.9                                                                                                      |
| 2018/03/05 | 190   | Downgrade to Go 1.9.2 to avoid run away go routines                                                                |
| 2018/03/05 |       | Release 0.0.8                                                                                                      |
| 2018/03/05 | 187   | Create a schema for the NATS Stream Adapter and publish it in the messages                                         |
| 2018/03/05 | 174   | Report the `mtime` of the file in the file content registration plugin, support compressing the data               |
| 2018/03/02 | 183   | Update Go to `1.10`                                                                                                |
| 2018/03/01 | 180   | Show the Go version used to compile the binary in `buildinfo`                                                      |
| 2018/03/01 | 173   | Record and expose the total number of messages received by the `server`                                            |
| 2018/03/01 | 176   | Intercept various `gnatsd` debug log messages and elevate them to notice and error                                 |
| 2018/03/01 | 175   | Update embedded `gnatsd` to `1.0.6`                                                                                |
| 2018/02/19 | 171   | Show embedded `gnatsd` version in `buildinfo`                                                                      |
| 2018/02/19 |       | Release 0.0.7                                                                                                      |
| 2018/02/19 | 165   | Discard NATS messages when the work buffer is full in the NATS Streaming adapter                                   |
| 2018/02/19 | 166   | Remove unwanted debug output                                                                                       |
| 2018/02/16 | 167   | Clarify the Choria flavor reported by choria_util#info                                                             |
| 2018/02/01 | 163   | Avoid large data storms after a reconnect cycle by limiting the publish buffer                                     |
| 2018/02/01 | 151   | Add xenial and stretch packages                                                                                    |
| 2018/01/22 | 152   | Support automagic validation of structs received over the network, support shellsafe for now                       |
| 2018/01/20 | 150   | Release 0.0.6                                                                                                      |
| 2018/01/20 | 58    | A mostly compatible `rpcutil` agent was added                                                                      |
| 2018/01/20 | 148   | The TTL of incoming request messages are checked                                                                   |
| 2018/01/20 | 146   | Stats about the server and message life cycle are recorded                                                         |
| 2018/01/19 | 133   | A timeout context is supplied to actions when they get executed                                                    |
| 2018/01/16 | 134   | Use new packaging infrastructure and move building to a circleci pipeline                                          |
| 2018/01/12 | 131   | Additional agents can now be added into the binary at compile time                                                 |
| 2018/01/12 | 125   | All files in additional dot config dirs are now parsed                                                             |
| 2018/01/12 | 128   | Add additional fields related to the RPC request to mcorpc.Request                                                 |
| 2018/01/10 | 120   | The concept of a provisioning mode was added along with a agent to assist automated provisioning                   |
| 2018/01/09 | 60    | Auditing was added for mcorpc agents                                                                               |
| 2018/01/09 | 69    | The protocol package has been moved to `choria-io/go-protocol`                                                     |
| 2018/01/08 | 118   | Create a helper to parse mcorpc requests into a standard structure                                                 |
| 2018/01/05 | 114   | Ensure the logfile name matches the package name                                                                   |
| 2018/01/06 |       | Release 0.0.5                                                                                                      |
| 2018/01/05 | 110   | Correctly detect startup failures in the el6 init script                                                           |
| 2018/01/04 | 111   | Treat the defaults file as config in the el6 rpm                                                                   |
| 2017/12/25 | 108   | Improve logrotation - avoid appending to a rotated file                                                            |
| 2017/12/21 | 106   | Make the max connections a build parameter and default it to 50 000                                                |
| 2017/12/20 | 101   | Add a random backoff to initial connection in adapters and the connector                                           |
| 2017/12/20 | 102   | Expose connector details to prometheus                                                                             |
| 2017/12/13 |       | Release 0.0.4                                                                                                      |
| 2017/12/14 | 97    | Stats about the internals of the protocol are exposed                                                              |
| 2017/12/14 | 80    | When doing SRV lookups employ a cache to speed things up                                                           |
| 2017/12/14 | 92    | When shutting down daemons on rhel6 wait for them to exit and then KILL them after 5 seconds                       |
| 2017/12/14 | 91    | Avoid race condition while determining if the network broker started                                               |
| 2017/12/14 | 90    | Emit build info on `/choria/`                                                                                      |
| 2017/12/13 |       | Release 0.0.3                                                                                                      |
| 2017/12/12 | 81    | Export metrics `/choria/prometheus` when enabled                                                                   |
| 2017/12/10 | 73    | Federation brokers now correctly subscribe to the configured names                                                 |
| 2017/12/10 | 71    | Fix TLS network cluster                                                                                            |
| 2017/12/10 |       | Release 0.0.2                                                                                                      |
| 2017/12/10 | 67    | Distribute sample `broker.conf` and `server.conf`                                                                  |
| 2017/12/10 | 65    | When running as root do not call `puppet apply` 100s of times                                                      |
| 2017/12/10 | 64    | Ensure the broker exits on interrupt when the NATS based broker is running                                         |
| 2017/12/09 | 59    | Add a compatible `choria_util` agent                                                                               |
| 2017/12/09 | 57    | Create basic MCollective SimpleRPC compatible agents written in Go and compiled in                                 |
| 2017/12/08 | 53    | Adds the `buildinfo` subcommand                                                                                    |
| 2017/12/08 | 52    | Improve cross compile compatibility by using `os.Getuid()` instead of `user.Current()`                             |
| 2017/12/08 |       | Release 0.0.1                                                                                                      |
