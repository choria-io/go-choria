package natsstream

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/choria-io/go-choria/broker/adapter/stats"
	"github.com/choria-io/go-srvcache"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/choria-io/go-choria/choria"
	log "github.com/sirupsen/logrus"
)

type nats struct {
	servers func() (srvcache.Servers, error)
	topic   string
	proto   string
	name    string
	group   string

	input chan *choria.ConnectorMessage
	work  chan adaptable

	log  *log.Entry
	conn choria.Connector
}

func newIngest(name string, work chan adaptable, logger *log.Entry) ([]*nats, error) {
	prefix := fmt.Sprintf("plugin.choria.adapter.%s.ingest.", name)

	instances, err := strconv.Atoi(cfg.Option(prefix+"workers", "10"))
	if err != nil {
		return nil, fmt.Errorf("%s should be a integer number", prefix+"workers")
	}

	topic := cfg.Option(prefix+"topic", "")
	if topic == "" {
		return nil, fmt.Errorf("No ingest topic configured, please set %s", prefix+"topic")
	}

	_, err = framework.MiddlewareServers()
	if err != nil {
		return nil, fmt.Errorf("Could not resolve initial server list: %s", err)
	}

	proto := cfg.Option(prefix+"protocol", "reply")

	workers := []*nats{}

	if proto == "request" {
		proto = "choria:request"
	} else {
		proto = "choria:reply"
	}

	for i := 0; i < instances; i++ {
		iname := fmt.Sprintf("%s.%d", name, i)
		logger.Infof("Creating NATS Streaming Adapter %s %s Ingest instance %d / %d", name, topic, i, instances)

		n := &nats{
			name:    iname,
			group:   "nats_ingest_" + name,
			topic:   topic,
			work:    work,
			servers: framework.MiddlewareServers,
			proto:   proto,
			log:     logger.WithFields(log.Fields{"side": "ingest", "instance": i}),
		}

		workers = append(workers, n)
	}

	return workers, nil
}

func (na *nats) connect(ctx context.Context, cm choria.ConnectionManager) error {
	if ctx.Err() != nil {
		return fmt.Errorf("Shutdown called")
	}

	var err error

	na.conn, err = cm.NewConnector(ctx, na.servers, fmt.Sprintf("choria adapter %s", na.name), na.log)
	if err != nil {
		return fmt.Errorf("Could not start NATS connection: %s", err)
	}

	na.input, err = na.conn.ChanQueueSubscribe(na.name, na.topic, na.group, 1000)
	if err != nil {
		return fmt.Errorf("Could not subscribe to %s: %s", na.topic, err)
	}

	return nil
}

func (na *nats) disconnect() {
	if na.conn != nil {
		na.log.Info("Disconnecting from NATS")
		na.conn.Close()
	}
}

func (na *nats) receiver(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	bytes := stats.BytesCtr.WithLabelValues(na.name, "input", cfg.Identity)
	ectr := stats.ErrorCtr.WithLabelValues(na.name, "input", cfg.Identity)
	ctr := stats.ReceivedMsgsCtr.WithLabelValues(na.name, "input", cfg.Identity)
	timer := stats.ProcessTime.WithLabelValues(na.name, "input", cfg.Identity)

	receiverf := func(cm *choria.ConnectorMessage) {
		obs := prometheus.NewTimer(timer)
		defer obs.ObserveDuration()

		rawmsg := cm.Data
		var msg adaptable
		var err error

		bytes.Add(float64(len(rawmsg)))

		if na.proto == "choria:request" {
			msg, err = framework.NewRequestFromTransportJSON(rawmsg, true)
		} else {
			msg, err = framework.NewReplyFromTransportJSON(rawmsg, true)
		}

		if err != nil {
			na.log.Warnf("Could not process message, discarding: %s", err)
			ectr.Inc()
			return
		}

		// If the work queue is full, perhaps due to the other side
		// being slow or disconnected when we get full we will block
		// and that will cause NATS to disconnect us as a slow consumer
		//
		// Since slow consumer disconnects discards a load of messages
		// anyway we might as well discard them here and avoid all the
		// disconnect/reconnect noise
		//
		// Essentially the NATS -> NATS Stream bridge functions as a
		// broadcast to ordered queue bridge and by it's nature this
		// side has to be careful to handle when the other side gets
		// into a bad place.  The work channel has 1000 capacity so
		// this gives us a good buffer to weather short lived storms
		if len(na.work) == cap(na.work) {
			na.log.Warn("Work queue is full, discarding message")
			ectr.Inc()
			return
		}

		na.work <- msg

		ctr.Inc()
	}

	for {
		select {
		case cm := <-na.input:
			receiverf(cm)

		case <-ctx.Done():
			na.disconnect()

			return
		}
	}
}
