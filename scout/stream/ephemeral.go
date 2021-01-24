package stream

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"sync"
	"time"

	"github.com/nats-io/jsm.go"
	"github.com/nats-io/jsm.go/api"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/backoff"
	"github.com/choria-io/go-choria/internal/util"
)

type Ephemeral struct {
	ctx       context.Context
	cancel    func()
	stream    *jsm.Stream
	cons      *jsm.Consumer
	opts      *api.ConsumerConfig
	log       *logrus.Entry
	interval  time.Duration
	resumeSeq uint64

	sync.Mutex
}

func NewEphemeral(stream *jsm.Stream, interval time.Duration, log *logrus.Entry, opts ...jsm.ConsumerOption) (e *Ephemeral, err error) {
	if stream == nil {
		return nil, fmt.Errorf("stream is required")
	}

	e = &Ephemeral{
		stream:   stream,
		interval: interval,
	}

	if log == nil {
		logger := logrus.New()
		logger.SetOutput(ioutil.Discard)
		e.log = logrus.NewEntry(logger)
	} else {
		e.log = log.WithFields(logrus.Fields{"stream": stream.Name()})
	}

	e.opts, err = jsm.NewConsumerConfiguration(jsm.DefaultConsumer, opts...)
	if err != nil {
		return nil, err
	}

	if e.opts.MaxAckPending == 0 || e.opts.MaxAckPending > 100 {
		e.opts.MaxAckPending = 100
	}

	if e.opts.AckPolicy == api.AckNone {
		return nil, fmt.Errorf("ack policy has to be all or explicit")
	}

	e.ctx, e.cancel = context.WithCancel(context.Background())

	err = e.start()
	if err != nil {
		e.Close()
		return nil, err
	}

	return e, nil
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

	e.resumeSeq = meta.StreamSequence() + 1
}

func (e *Ephemeral) Consumer() *jsm.Consumer {
	e.Lock()
	defer e.Unlock()

	return e.cons
}
func (e *Ephemeral) start() error {
	return e.manageConsumer()
}

func (e *Ephemeral) Close() {
	e.Lock()
	cancel := e.cancel
	e.Unlock()

	if cancel != nil {
		cancel()
	}
}

func (e *Ephemeral) createConsumer() (cons *jsm.Consumer, err error) {
	e.Lock()
	opts := *e.opts
	rseq := e.resumeSeq
	e.Unlock()

	if rseq > 0 {
		opts.OptStartSeq = rseq
		opts.DeliverPolicy = api.DeliverByStartSequence
		opts.OptStartTime = nil
	}

	cons, err = e.stream.NewConsumerFromDefault(opts)
	if err != nil {
		e.log.Errorf("Could not create consumer on stream %q: %s", e.stream.Name(), err)
		err = backoff.TwentySec.For(e.ctx, func(try int) error {
			e.log.Warnf("Trying to create consumer on stream %q, try %d", e.stream.Name(), try)
			cons, err = e.stream.NewConsumerFromDefault(opts)
			return err
		})
	}

	e.Lock()
	e.cons = cons
	e.Unlock()

	return cons, err
}

func (e *Ephemeral) manageConsumer() error {
	cons, err := e.createConsumer()
	if err != nil {
		return err
	}

	go func() {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		splay := time.Duration(r.Intn(int(e.interval)))
		err = util.InterruptibleSleep(e.ctx, splay)
		if err != nil {
			return
		}

		ticker := time.NewTicker(e.interval)
		defer ticker.Stop()

		for {
			select {
			case <-e.ctx.Done():
				return
			case <-ticker.C:
				_, err = cons.State()
				if err != nil {
					e.log.Warnf("Ephemeral consumer %q > %q has failed, recreating", cons.StreamName(), cons.Name())
					cons, err = e.createConsumer()
					if err != nil {
						return
					}
				}
			}
		}
	}()

	return nil
}
