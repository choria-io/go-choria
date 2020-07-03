## Entity

An entity represents a single monitored item, it will run the Scout agent and manage itself from JetStream.

## Streams and Subjects

|Stream|Subjects|Description|
|------|--------|-----------|
|SCOUT_TAGS|scout.tags.>|Checks to apply to nodes tagged with a tag|
|SCOUT_CHECKS|scout.check.*|Stream Template creating streams per check holding definitions etc|
|SCOUT_OVERRIDES|scout.overrides.>|Stream holding entity overrides|
|scout_check_*|scout_check_x|Streams created by the SCOUT_CHECKS template holding check definitions|

## Startup

```nohighlight
$ scout run --config /etc/choria
```

Here it will look for a few files:

### `scout.cfg`

For connecting, provisioning etc

```ini
# normal choria config
... 
```

### `tags.json`

Tags identify what groups of checks the entity will run on a node, this is loaded from a stream `SCOUT_TAGS` matching
subjects like `scout.tags.<tag>`

```json
[
  "common",
  "e.example.net"
]
```

These lists loaded from the Stream are lists like:

```json
[
  "check_load",
  "check_puppet"
]
```

These will map to a stream `SCOUT_CHECKS` full of machine definitions or at least an abstraction over them
