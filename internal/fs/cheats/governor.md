# further documentation https://choria.io/docs/streams/governor/

# to create governor with 10 slots and 1 minute timeout
choria governor add cron 10 1m

# to view the configuration and state
choria governor view cron

# to reset the governor, clearing all slots
choria governor reset cron

# to run long-job.sh when a slot is available, giving up after 20 minutes without a slot
choria governor run cron --max-wait 20m long-job.sh

# to run a cron job across a pool of machines once only per hour
choria governor add cron 1 59m 3
choria governor run cron --max-wait 10s --max-per-period long-job.sh

# list known governors
choria governor list
