// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package stream

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/jsm.go"
	"github.com/nats-io/jsm.go/api"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/backoff"
)

type Ephemeral struct {
	stream *jsm.Stream
	conn   *nats.Conn
	seen   time.Time
	cfg    *api.ConsumerConfig
	q      chan *nats.Msg
	ctx    context.Context
	cancel func()
	sub    *nats.Subscription
	cons   *jsm.Consumer
	log    *logrus.Entry

	resumeSequence uint64

	sync.Mutex
}

func NewEphemeral(ctx context.Context, nc *nats.Conn, stream *jsm.Stream, interval time.Duration, q chan *nats.Msg, log *logrus.Entry, opts ...jsm.ConsumerOption) (*Ephemeral, error) {
	eph := &Ephemeral{
		stream: stream,
		conn:   nc,
		q:      q,
		log:    log,
	}

	var err error
	eph.cfg, err = jsm.NewConsumerConfiguration(jsm.DefaultConsumer, opts...)
	if err != nil {
		return nil, err
	}

	if eph.cfg.MaxAckPending == 0 || eph.cfg.MaxAckPending > 100 {
		eph.cfg.MaxAckPending = 100
	}

	if eph.cfg.AckPolicy == api.AckNone {
		return nil, fmt.Errorf("ack policy has to be all or explicit")
	}

	eph.cfg.Heartbeat = interval
	eph.cfg.FlowControl = true

	eph.ctx, eph.cancel = context.WithCancel(ctx)

	return eph, eph.start()
}

func (e *Ephemeral) start() error {
	go func() {
		err := e.manage()
		if err != nil {
			e.log.Errorf("Managed ephemeral failed: %s", err)
		}
	}()

	return nil
}

func (e *Ephemeral) manage() error {
	msgq := make(chan *nats.Msg, 1000)

	e.log.Debugf("Creating consumer")
	err := e.createConsumer(msgq)
	if err != nil {
		return err
	}

	ticker := time.NewTicker(e.cfg.Heartbeat + 2*time.Second)
	defer ticker.Stop()

	for {
		select {
		case msg := <-msgq:
			e.markLastSeen()

			// handle and discard the keep alive messages, process flow control and unstuck stalled consumers
			if len(msg.Data) == 0 && msg.Header.Get("Status") == "100" {
				if msg.Reply != "" {
					msg.Respond(nil)
				} else if stalled := msg.Header.Get("Nats-Consumer-Stalled"); stalled != "" {
					e.conn.Publish(stalled, nil)
				}

				continue
			}

			e.q <- msg

		case <-e.ctx.Done():
			close(msgq)
			return nil

		case <-ticker.C:
			e.log.Debugf("Checking consumer %s state", e.cons.Name())

			e.Lock()
			cons := e.cons
			seen := e.seen
			e.Unlock()

			since := time.Since(seen)
			if since > e.cfg.Heartbeat {
				e.log.Warnf("Consumer failed, last seen %v", since)
				cons.Delete()
				err = e.createConsumer(msgq)
				if err != nil {
					e.log.Warnf("Consumer creation failed: %s", err)
					return err
				}
			}
		}
	}
}

func (e *Ephemeral) markLastSeen() {
	e.Lock()
	e.seen = time.Now()
	e.Unlock()
}

func (e *Ephemeral) SetResumeSequence(m *nats.Msg) {
	if m == nil {
		return
	}

	if e == nil {
		return
	}

	meta, _ := jsm.ParseJSMsgMetadata(m)
	if meta == nil {
		return
	}

	e.Lock()
	defer e.Unlock()

	e.resumeSequence = meta.StreamSequence() + 1
}

func (e *Ephemeral) createConsumer(msgq chan *nats.Msg) error {
	e.Lock()
	defer e.Unlock()

	var err error

	return backoff.TwentySec.For(e.ctx, func(i int) error {
		if e.sub != nil {
			e.log.Debugf("Unsubscribing from inbox %s", e.sub.Subject)
			e.sub.Unsubscribe()
		}

		if e.cons != nil {
			e.log.Debugf("Deleting existing consumer")
			e.cons.Delete()
		}

		e.sub, err = e.conn.ChanSubscribe(e.conn.NewRespInbox(), msgq)
		if err != nil {
			e.log.Warnf("Subscription failed on try %d: %s", i, err)
			return err
		}
		e.log.Debugf("Subscribed to %s", e.sub.Subject)

		e.cfg.DeliverSubject = e.sub.Subject
		if e.resumeSequence != 0 {
			e.cfg.OptStartSeq = e.resumeSequence
			e.cfg.DeliverPolicy = api.DeliverByStartSequence
			e.cfg.OptStartTime = nil
		}

		e.log.Debugf("Creating consumer using configuration: %#v", e.cfg)

		e.cons, err = e.stream.NewConsumerFromDefault(*e.cfg)
		e.conn.Flush()
		if err != nil {
			e.log.Warnf("Creating consumer failed: %s", err)
			return err
		}
		e.seen = time.Now()
		e.log.Debugf("Created new consumer %s", e.cons.Name())

		return nil
	})
}

func (e *Ephemeral) Close() {
	e.Lock()
	cancel := e.cancel
	sub := e.sub
	cons := e.cons
	e.Unlock()

	if cancel != nil {
		cancel()
	}

	if sub != nil {
		sub.Unsubscribe()
	}

	if cons != nil {
		cons.Delete()
	}
}
