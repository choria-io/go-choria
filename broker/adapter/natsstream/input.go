package natsstream

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/choria-io/go-choria/statistics"

	"github.com/choria-io/go-choria/choria"
	log "github.com/sirupsen/logrus"
)

type nats struct {
	servers func() ([]choria.Server, error)
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

	instances, err := strconv.Atoi(config.Option(prefix+"workers", "10"))
	if err != nil {
		return nil, fmt.Errorf("%s should be a integer number", prefix+"workers")
	}

	topic := config.Option(prefix+"topic", "")
	if topic == "" {
		return nil, fmt.Errorf("No ingest topic configured, please set %s", prefix+"topic")
	}

	_, err = framework.MiddlewareServers()
	if err != nil {
		return nil, fmt.Errorf("Could not resolve initial server list: %s", err.Error())
	}

	servers := func() ([]choria.Server, error) {
		return framework.MiddlewareServers()
	}

	proto := config.Option(prefix+"protocol", "reply")

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
			servers: servers,
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
		return fmt.Errorf("Could not start NATS connection: %s", err.Error())
	}

	na.input, err = na.conn.ChanQueueSubscribe(na.name, na.topic, na.group, 1000)
	if err != nil {
		return fmt.Errorf("Could not subscribe to %s: %s", na.topic, err.Error())
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

	ctr := statistics.Counter(fmt.Sprintf("adapter.%s.received", na.name))
	ectr := statistics.Counter(fmt.Sprintf("adapter.%s.err", na.name))
	bytes := statistics.Counter(fmt.Sprintf("adapter.%s.bytes", na.name))
	timer := statistics.Timer(fmt.Sprintf("adapter.%s.time", na.name))

	for {
		select {
		case cm := <-na.input:
			timer.Time(func() {
				rawmsg := cm.Data
				var msg adaptable
				var err error

				bytes.Inc(int64(len(rawmsg)))

				if na.proto == "choria:request" {
					msg, err = framework.NewRequestFromTransportJSON(rawmsg, true)
				} else {
					msg, err = framework.NewReplyFromTransportJSON(rawmsg)
				}

				if err != nil {
					na.log.Warnf("Could not process message, discarding: %s", err.Error())
					ectr.Inc(1)
					return
				}

				na.work <- msg

				ctr.Inc(1)
			})
		case <-ctx.Done():
			na.disconnect()

			return
		}
	}
}
