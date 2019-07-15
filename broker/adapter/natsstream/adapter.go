package natsstream

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/choria-io/go-choria/broker/adapter/stats"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-config"
	log "github.com/sirupsen/logrus"
)

// matches both protocol.Request and protocol.Reply
type adaptable interface {
	Message() string
	SenderID() string
	Time() time.Time
	RequestID() string
}

// NatStream is an adapter that connects a NATS topic with messages
// sent from Choria in its usual transport protocol to a NATS
// Streaming topic.
//
// On the stream the messages will be JSON format with keys
// body, sender and time.  Body is a base64 encoded string
//
// Configure the adapters:
//   # required
//   plugin.choria.adapters = discovery
//   plugin.choria.adapter.discovery.type = nats_stream
//   plugin.choria.adapter.discovery.queue_len = 1000 # default
//
// Configure the stream:
//
//   plugin.choria.adapter.discovery.stream.servers = stan1:4222,stan2:4222
//   plugin.choria.adapter.discovery.stream.clusterid = prod
//   plugin.choria.adapter.discovery.stream.topic = discovery # default
//   plugin.choria.adapter.discovery.stream.workers = 10 # default
//
// Configure the NATS ingest:
//
//    plugin.choria.adapter.discovery.ingest.topic = mcollective.broadcast.agent.discovery
//    plugin.choria.adapter.discovery.ingest.protocol = request # or reply
//    plugin.choria.adapter.discovery.ingest.workers = 10 # default
type NatStream struct {
	streams []*stream
	ingests []*nats

	work chan adaptable

	log *log.Entry
}

var framework *choria.Framework
var cfg *config.Config

func Create(name string, choria *choria.Framework) (adapter *NatStream, err error) {
	framework = choria
	cfg = choria.Configuration()

	s := fmt.Sprintf("plugin.choria.adapter.%s.queue_len", name)
	worklen, err := strconv.Atoi(cfg.Option(s, "1000"))
	if err != nil {
		return nil, fmt.Errorf("%s should be a integer number", s)
	}

	stats.WorkQueueCapacityGauge.WithLabelValues(name, cfg.Identity).Set(float64(worklen))

	adapter = &NatStream{
		log:  log.WithFields(log.Fields{"component": "nats_stream_adapter", "name": name}),
		work: make(chan adaptable, worklen),
	}

	adapter.streams, err = newStream(name, adapter.work, adapter.log)
	if err != nil {
		return nil, fmt.Errorf("Could not create adapter %s: %s", name, err)
	}

	adapter.ingests, err = newIngest(name, adapter.work, adapter.log)
	if err != nil {
		return nil, fmt.Errorf("Could not create adapter %s: %s", name, err)
	}

	return adapter, err
}

func (sa *NatStream) Init(ctx context.Context, cm choria.ConnectionManager) (err error) {
	for _, worker := range sa.streams {
		if ctx.Err() != nil {
			return fmt.Errorf("Shutdown called")
		}

		err = worker.connect(ctx, cm)
		if err != nil {
			return fmt.Errorf("Failure during initial NATS Streaming connections: %s", err)
		}
	}

	for _, worker := range sa.ingests {
		if ctx.Err() != nil {
			return fmt.Errorf("Shutdown called")
		}

		err = worker.connect(ctx, cm)
		if err != nil {
			return fmt.Errorf("Failure during NATS initial connections: %s", err)
		}
	}

	return nil
}

func (sa *NatStream) Process(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	for _, worker := range sa.streams {
		wg.Add(1)
		go worker.publisher(ctx, wg)
	}

	for _, worker := range sa.ingests {
		wg.Add(1)
		go worker.receiver(ctx, wg)
	}

	return
}
