// Copyright (c) 2019-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package ingest

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/choria-io/go-choria/broker/adapter/stats"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/protocol"
	"github.com/choria-io/go-choria/srvcache"
	"github.com/prometheus/client_golang/prometheus"

	log "github.com/sirupsen/logrus"
)

// Adaptable matches both protocol.Request and protocol.Reply
type Adaptable interface {
	Message() string
	SenderID() string
	Time() time.Time
	RequestID() string
}

type NatsIngest struct {
	topic       string
	proto       string
	name        string
	adapterName string
	group       string

	input chan inter.ConnectorMessage
	work  chan Adaptable

	fw   Framework
	cfg  *config.Config
	log  *log.Entry
	conn inter.Connector
}

type Framework interface {
	Configuration() *config.Config
	MiddlewareServers() (servers srvcache.Servers, err error)
	NewRequestFromTransportJSON(payload []byte, skipvalidate bool) (msg protocol.Request, err error)
	NewReplyFromTransportJSON(payload []byte, skipvalidate bool) (msg protocol.Reply, err error)
}

func New(name string, work chan Adaptable, fw Framework, logger *log.Entry) ([]*NatsIngest, error) {
	prefix := fmt.Sprintf("plugin.choria.adapter.%s.ingest.", name)
	cfg := fw.Configuration()

	instances, err := strconv.Atoi(cfg.Option(prefix+"workers", "10"))
	if err != nil {
		return nil, fmt.Errorf("%s should be a integer number", prefix+"workers")
	}

	topic := cfg.Option(prefix+"topic", "")
	if topic == "" {
		return nil, fmt.Errorf("no ingest topic configured, please set %s", prefix+"topic")
	}

	proto := cfg.Option(prefix+"protocol", "reply")

	workers := []*NatsIngest{}

	if proto == "request" {
		proto = "choria:request"
	} else {
		proto = "choria:reply"
	}

	logger.Infof("Creating NATS Adapter %s for topic %s Ingest with %d instances", name, topic, instances)
	for i := 0; i < instances; i++ {
		iname := fmt.Sprintf("%s.%d", name, i)
		logger.Debugf("Creating NATS Adapter %s %s Ingest instance %d / %d", name, topic, i, instances)

		n := &NatsIngest{
			name:        iname,
			adapterName: name,
			group:       "nats_ingest_" + name,
			topic:       topic,
			work:        work,
			proto:       proto,
			fw:          fw,
			cfg:         fw.Configuration(),
			log:         logger.WithFields(log.Fields{"side": "ingest", "instance": i}),
		}

		workers = append(workers, n)
	}

	return workers, nil
}

func (na *NatsIngest) Connect(ctx context.Context, cm inter.ConnectionManager) error {
	if ctx.Err() != nil {
		return fmt.Errorf("shutdown called")
	}

	var err error

	na.conn, err = cm.NewConnector(ctx, na.fw.MiddlewareServers, fmt.Sprintf("choria adapter %s", na.name), na.log)
	if err != nil {
		return fmt.Errorf("could not start NATS connection: %s", err)
	}

	na.input, err = na.conn.ChanQueueSubscribe(na.name, na.topic, na.group, 1000)
	if err != nil {
		return fmt.Errorf("could not subscribe to %s: %s", na.topic, err)
	}

	return nil
}

func (na *NatsIngest) disconnect() {
	if na.conn != nil {
		na.log.Debugf("Disconnecting from NATS")
		na.conn.Close()
	}
}

func (na *NatsIngest) Receiver(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	bytes := stats.BytesCtr.WithLabelValues(na.name, "input", na.cfg.Identity)
	ectr := stats.ErrorCtr.WithLabelValues(na.name, "input", na.cfg.Identity)
	ctr := stats.ReceivedMsgsCtr.WithLabelValues(na.name, "input", na.cfg.Identity)
	timer := stats.ProcessTime.WithLabelValues(na.name, "input", na.cfg.Identity)
	workqlen := stats.WorkQueueLengthGauge.WithLabelValues(na.adapterName, na.cfg.Identity)

	receiverf := func(cm inter.ConnectorMessage) {
		obs := prometheus.NewTimer(timer)
		defer obs.ObserveDuration()
		defer func() { workqlen.Set(float64(len(na.work))) }()

		rawmsg := cm.Data()
		var msg Adaptable
		var err error

		bytes.Add(float64(len(rawmsg)))

		if na.proto == "choria:request" {
			msg, err = na.fw.NewRequestFromTransportJSON(rawmsg, true)
		} else {
			msg, err = na.fw.NewReplyFromTransportJSON(rawmsg, true)
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
		select {
		case na.work <- msg:
		default:
			na.log.Warn("Work queue is full, discarding message")
			ectr.Inc()
			return
		}

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
