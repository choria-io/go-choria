## Archive Watcher for Choria Autonomous Agents

This repository contains an [Autonomous Agent](https://choria.io/docs/autoagents/) Watcher plugin capable of
downloading `tar.gz` archives from an HTTP(S) repository, verify them continuously and repair them on unexpected
changes.

## Archive Watcher

### Goals

This is built in a way that would allow the initial Autonomous Agent to be compiled into the Choria binary, meaning no-one
can modify its behaviour or requirements for checksum verification etc.

The compiled-in Autonomous Agent then reads configuration about URLs and checksums to download from a Key-Value store that
sets the URL to fetch, the checksum of the downloaded archive AND the checksum of the `SHA256SUMS` file in the archive.

This way should someone on the server change any downloaded file or tries to edit the `SHA256SUMS` file would trigger a
remediation via re-download.

The download concurrency can be controlled using a [Choria Governor](https://choria.io/docs/streams/governor/) ensuring that
on a large network the webservers are not overwhelmed.

Every event such as verification failing, new files downloaded etc publish [CloudEvents](https://cloudevents.io/) into
[Choria Streams](https://choria.io/docs/streams/governor/) that can be viewed real time using `choria tool event`.

Today it's focussed on small single directory archives - like Choria Autonomous Agents - and the verification is restricted
to a single directory.

### Preparing an archive

This supports GZipped Tar files only, we have a typical Choria Autonomous Agent here:

```nohighlight
metadata
├── machine.yaml
├── gather.sh
└── SHA256SUMS
```

The `SHA256SUMS` file was made using `sha256sum * > SHA256SUMS`.

We tar up this archive and again get another SHA256 for it:

```nohighlight
$ cd metadata
$ sha256sum * > SHA256SUMS
$ cd -
$ tar -cvzf metadata-machine-1.0.0.tgz metadata
$ sha256sum metadata-machine-1.0.0.tgz metadata/SHA256SUMS
f11ea2005de97bf309bafac46e77c01925307a26675f44f388d4502d2b9d00bf  metadata-machine-1.0.0.tgz
1e85719c6959eb0f2c8f2166e30ae952ccaef2c286f31868ea1d311d3738a339  metadata/SHA256SUMS
```

Place this file on any webserver of your choice. Note these checksums for later.

### Configuration

First we'll create a Key-Value store to configure this Autonomous Agent, since we're creating one that introspect the machine
for some metadata we call it `METADATA`:

```nohighlight
$ choria kv add METADATA --replicas 3
```

We then place our initial configuration in the bucket:

```nohighlight
$ choria kv put METADATA machine \
'{
  "source": "https://my.example.net/metadata/metadata-machine-1.0.0.tgz",
  "checksum": "f11ea2005de97bf309bafac46e77c01925307a26675f44f388d4502d2b9d00bf",
  "verify_checksum": "1e85719c6959eb0f2c8f2166e30ae952ccaef2c286f31868ea1d311d3738a339"
}'
```

The `source` is where to get the file, `checksum` is the SHA256 sum of the `metadata-machine-1.0.0.tgz` and the `verify_checksum`
is the SHA256 sum of the `SHA256SUMS` file that's inside `metadata-machine-1.0.0.tgz`

Now we arrange for this data to be placed on each node and subsequent changes to be monitored using the [KV Watcher](https://choria.io/docs/autoagents/watcher_reference/#key-value-store):

```yaml
watchers:
  - name: data
    type: kv
    interval: 55s
    state_match: [MANAGE]
    properties:
      bucket: METADATA
      key: machine
      mode: poll
      bucket_prefix: false
```

Finally, we set up our metadata manager to fetch and maintain the metadata gathering Autonomous Agent:

```yaml
watchers:
  - name: download
    state_match: [MANAGE]
    type: archive
    interval: 1m
    properties:
      source: '{{ lookup "data.machine.source" "" }}'
      checksum: '{{ lookup "data.machine.checksum" "" }}'
      verify_checksum: '{{ lookup "data.machine.verify_checksum" "" }}'
      username: artifacts
      password: toomanysecrets
      target: /etc/choria/machines
      creates: metadata
      verify: SHA256SUMS
```

This will:

* Every minute
    * Checks that the `/etc/choria/machines/metadata` directory exist
    * Verify the checksum of `/etc/choria/machines/metadata/SHA256SUMS`
    * Verify the checksum of every file in `/etc/choria/machines/metadata` using the `SHA256SUMS` file
    * If verification failed, downloads the file:
        * Into a temporary directory
        * Verifies the checksum of the `tar.gz`
        * Extract it, verifies it makes `metadata`
        * Verify every file in it based on `SHA256SUMS` after first verifying `SHA256SUMS` is legit
        * Remove the existing files in `/etc/choria/machines/metadata`
        * Replace them with the new files

## Compiling into Choria

This plugin requires custom-builds of Choria. Update the `user-plugins.yaml`:

```yaml
# packager/user_plugins.yaml
archive_watcher: github.com/choria-io/go-choria/aagent/watchers/archivewatcher
```
And run `go generate`, the plugin will be compiled in.
