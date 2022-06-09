# further documentation https://choria.io/docs/streams/key-value/

# to create a replicated KV bucket
choria kv add CONFIG --replicas 3

# to store a value in the bucket
choria kv put CONFIG username bob

# to read just the value with no additional details
choria kv get CONFIG username --raw

# view an audit trail for a key if history is kept
choria kv history CONFIG username

# to see the bucket status
choria kv status CONFIG

# observe real time changes for an entire bucket
choria kv watch CONFIG

# observe real time changes for all keys below users
choria kv watch CONFIG 'users.>''

# create a bucket backup for CONFIG into backups/CONFIG
choria kv backup CONFIG ./backups/CONFIG

# restore a bucket from a backup
choria kv restore ./backups/CONFIG

# list known buckets
nats kv ls
