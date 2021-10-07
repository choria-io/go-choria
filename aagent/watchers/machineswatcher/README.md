## Machines Watchers for Choria Autonomous Agents

This contains an [Autonomous Agent](https://choria.io/docs/autoagents/) Watcher plugin capable of
managing the typical `/etc/choria/machines` directory via Choria Key-Value Store and the `archive`
watcher.

In effect this allows you to Configuration Manage sets of Autonomous Agents on a fleet where you do not have
other Configuration Management tools or where you just want to manage these out of band.

## Goals

Create an opinionated manager for other machines that will safely and securely set up a server for hosting autonomous
agents using the properties of the `archive` watcher.

If this watcher is used it is one that would be compiled into the Choria binary and configured using KV.

## Autonomous Agent Archives

These archives are prepared as per the instructions in the [archive watcher](../archivewatcher/README.md) with the following hard constraints:

* The checksums file must be `SHA256SUMS` and must be present
* The tar file must create a directory matching the name exactly, `yourmachine-1.2.3.tar.gz` must create `yourmachine`
* Checksums of the `SHA256SUMS` file and the archive must be specified

### Configuring

An Autonomous agent must be created that polls the Key-Value store and then configures the `machines` type watcher:

```yaml
watchers:
  - name: desired_state
    type: kv
    interval: 1m
    state_match: [MANAGE]
    properties:
       bucket: MACHINES
       key: machines
       mode: poll
       bucket_prefix: false
    
  - name: manage_machines
    state_match: [MANAGE]
    type: machines
    interval: 1m
    state_matchin:
      - MANAGE
    properties:
      data_item: machines
      purge_unknown: true
      machine_manage_interval: 1m
      public_key: 64031219d4922eed63a5f567303e98607c632139c01bc9fa4ca2514c2d9d30da
```

Here we set an optional `public_key`, when this is set to a ed25519 public key it will verify and only accept data from the data store that has a valid signature signed using the corresponding private key.

A keypair can be created using the signer command:

```go
go run cmd/mms.go keys
 Public Key: 64031219d4922eed63a5f567303e98607c632139c01bc9fa4ca2514c2d9d30da
Private Key: d8bd4d6392af154e996a18a4ccd5f51931d8e861d42966a677d85fbb598b66d364031219d4922eed63a5f567303e98607c632139c01bc9fa4ca2514c2d9d30da
```

The data can now be created:

```nohighlight
$ cat machines.json
[
 {
   "name": "facts",
     "source": "https://my.example.net/metadata/metadata-machine-1.0.0.tgz",
     "verify": "SHA256SUMS",
     "verify_checksum": "1e85719c6959eb0f2c8f2166e30ae952ccaef2c286f31868ea1d311d3738a339",
     "checksum": "f11ea2005de97bf309bafac46e77c01925307a26675f44f388d4502d2b9d00bf",
     "match": "has_command('facter')"
 }
]
$ go run cmd/mms.go pack machines.json d8bd4d6392af154e996a18a4ccd5f51931d8e861d42966a677d85fbb598b66d364031219d4922eed63a5f567303e98607c632139c01bc9fa4ca2514c2d9d30da > spec.json
$ cat spec.json | choria kv put MACHINES machines -
{"machines":"WwogewogICAibmFtZSI6ICJmYWN0cyIsCiAgICAgInNvdXJjZSI6ICJodHRwczovL215LmV4YW1wbGUubmV0L21ldGFkYXRhL21ldGFkYXRhLW1hY2hpbmUtMS4wLjAudGd6IiwKICAgICAidmVyaWZ5IjogIlNIQTI1NlNVTVMiLAogICAgICJ2ZXJpZnlfY2hlY2tzdW0iOiAiMWU4NTcxOWM2OTU5ZWIwZjJjOGYyMTY2ZTMwYWU5NTJjY2FlZjJjMjg2ZjMxODY4ZWExZDMxMWQzNzM4YTMzOSIsCiAgICAgImNoZWNrc3VtIjogImYxMWVhMjAwNWRlOTdiZjMwOWJhZmFjNDZlNzdjMDE5MjUzMDdhMjY2NzVmNDRmMzg4ZDQ1MDJkMmI5ZDAwYmYiLAogICAgICJtYXRjaCI6ICJoYXNfY29tbWFuZCgnZmFjdGVyJykiCiB9Cl0K","signature":"f06d4a1cfe9ac79d26b5e6646fdfa9d845a5506c9a2fe0a71fb8416f6f7edd253a1eb46363c12ca5f6148b19ab1ed9a5f25c89b09b3360a09b7d054bf4b55204"}
```

After this the machines will be downloaded and maintained. In the `pack` command above the key is optional so the same command can be used to encode the specification without signing. They key can be read from the environment variable `KEY`.

Note the `has_command('facter')` for the `matcher` key, this is a small [expr](https://github.com/antonmedv/expr) expression
that is run on the node to determine if a specific machine should go on a node. The Key-Value is for the entire connected
DC so in order to allow heterogeneous environments machines that should not go on the entire fleet can be limited using matchers.

|Function|Description|
|--------|-----------|
|identity|Regular expression match over the machine identity|
|has_file|Determines if a regular file is present on the machine|
|has_directory|Determines if a directory is present on the machine|
|has_command|Searches `PATH` for a command, note the `PATH` choria runs with is quite limited|

The expression format is the typical used by Choria for example a match might be `identity('^web') && has_command('facter')`
would do pretty much the right thing.

## Compiling Autonomous Agents into Choria

Current `main` of Choria also supports compiling Autonomous Agents into Choria, if you're really paranoid or strict
you would compile the above autonomous agent into Choria and use it to bootstrap others in a trusted manner from a trusted
source allowing just the properties you want to be adjusted via Key-Value Store.

In this mode you can even forgo the entire Key-Value integration and compile urls and all checksums right into the binary
for the truly paranoid.

```go
package metamgr

import (
	"github.com/choria-io/go-choria/aagent/machine"
	"github.com/choria-io/go-choria/aagent/plugin"
)

func ChoriaPlugin() *plugin.MachinePlugin {
	return plugin.NewMachinePlugin("metamgr", &machine.Machine{
		MachineName: "metamgr",
		InitialState: "MANAGE",
		// rest of the autonomous agent
    })
}
```

You can now include this file in the `user_plugins.yaml` and it will be compiled in, see below example.  This way you have
an unmodifiable way to bootstrap a trusted set of Autonomous Agents onto new servers without needing Configuration Management

## Compiling into Choria

Compiling this into Choria is reasonably simple, assuming you already figured out how to compile choria :-)

```yaml
# packager/user_plugins.yaml
archive_watcher: github.com/choria-io/go-choria/aagent/watchers/archivewatcher
machines_watcher: github.com/choria-io/go-choria/aagent/watchers/machineswatcher
machines_manager: github.com/choria-io/go-choria/aagent/watchers/machineswatcher/manager
```

Do `go generate` and recompile, this will include the watcher.

