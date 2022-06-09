# further documentation https://choria.io/docs/concepts/cli/

# request the status of a service
choria req service status service=httpd

# restrict the query to a subset of nodes, see https://choria.io/docs/concepts/discovery/
choria req service status service=httpd -C /apache/

# restart services in a batched manner
choria req service restart service=httpd --batch 10 --batch-sleep 30

# filter replies, list host names where the service is not up
choria req service status service=httpd --filter-replies 'ok() && data("status")!="running"' --senders

# get results in JSON format
choria req service status service=httpd --json

# show only failed responses
choria req service status service=httpd --display failed

# target nodes listed in a file, reporting absent nodes
choria req service status service=httpd --nodes nodes.txt
