// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package submission

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/choria-io/go-choria/inter"
	"github.com/nats-io/jsm.go/api"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
)

const (
	defaultTTL             = 7 * 24 * time.Hour
	maxTTL                 = 31 * 24 * time.Hour
	defaultTimeout         = 2 * time.Second
	defaultMaxSpoolEntries = 1000
)

type StoreType int

const (
	Unknown   StoreType = 0
	Directory StoreType = 1
)

type Store interface {
	NewMessage() *Message
	StartPoll(context.Context, *sync.WaitGroup, func([]*Message) error) error
	Complete(*Message) error
	Discard(*Message) error
	IncrementTries(*Message) error
	Submit(msg *Message) error
}

type Submitter interface {
	Submit(msg *Message) error
}

type Spool struct {
	opts   *spoolOpts
	store  Store
	conn   inter.RawNATSConnector
	prefix string
	log    *logrus.Entry
}

func New(collective string, identity string, store StoreType, log *logrus.Entry, opts ...Option) (*Spool, error) {
	if collective == "" {
		return nil, fmt.Errorf("collective is required")
	}

	if identity == "" {
		return nil, fmt.Errorf("identity is unknown")
	}

	sopts := &spoolOpts{maxSize: defaultMaxSpoolEntries}
	for _, opt := range opts {
		opt(sopts)
	}

	spool := &Spool{
		log:    log.WithField("component", "submission"),
		prefix: fmt.Sprintf("%s.submission.in.", collective),
		opts:   sopts,
	}

	switch store {
	case Directory:
		st, err := NewDirectorySpool(sopts.spoolDir, sopts.maxSize, identity, spool.log)
		if err != nil {
			return nil, err
		}

		spool.store = st

	default:
		return nil, fmt.Errorf("unknown store type %v", store)
	}

	return spool, nil
}

func NewFromChoria(fw inter.Framework, store StoreType) (*Spool, error) {
	cfg := fw.Configuration()

	seed, _ := fw.SignerSeedFile()
	token, _ := fw.SignerTokenFile()

	return New(cfg.MainCollective, cfg.Identity, store, fw.Logger("submission"),
		WithSpoolDirectory(cfg.Choria.SubmissionSpool),
		WithMaxSpoolEntries(cfg.Choria.SubmissionSpoolMaxSize),
		WithSeedFile(seed),
		WithTokenFile(token))
}

func (s *Spool) Submit(msg *Message) error {
	return s.store.Submit(msg)
}

func (s *Spool) NewMessage() *Message {
	return s.store.NewMessage()
}

func (s *Spool) publishReliable(ctx context.Context, msg *nats.Msg, m *Message) (*api.JSPubAckResponse, error) {
	pctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	res, err := s.conn.RequestRawMsgWithContext(pctx, msg)
	if err != nil {
		return nil, fmt.Errorf("request failed: %s", err)
	}

	var ack api.JSPubAckResponse
	err = json.Unmarshal(res.Data, &ack)
	if err != nil {
		return nil, fmt.Errorf("invalid ack: %s", err)
	}

	if ack.Error != nil {
		return nil, fmt.Errorf("publish failed: %s", err)
	}

	s.store.Discard(m)

	return &ack, nil
}

func (s *Spool) Run(ctx context.Context, wg *sync.WaitGroup, conn inter.RawNATSConnector) {
	defer wg.Done()

	s.conn = conn

	wg.Add(1)
	s.store.StartPoll(ctx, wg, func(msgs []*Message) error {
		skipReliable := false

		for _, m := range msgs {
			msg, err := m.NatsMessage(s.prefix, s.opts.seedFile, s.opts.tokenFile)
			if err != nil {
				switch err {
				case ErrMessageMaxTries:
					s.log.Infof("Discarding max attempted message %s", m.ID)
				case ErrMessageExpired:
					s.log.Infof("Discarding expired message %s", m.ID)
				default:
					s.log.Errorf("Unknown error processing message, discarding %s: %s", m.ID, err)
				}
				s.store.Discard(m)

				continue
			}

			// always do 1 attempt to publish unreliable messages
			if !m.Reliable {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}
				
				err = s.conn.PublishRawMsg(msg)
				if err != nil {
					s.log.Errorf("Could not publish unreliable message %s, discarding: %s", m.ID, err)
				}
				s.store.Discard(m)
				continue
			}

			// if any reliable message fails we skip all future reliable messages to preserve order of reliable messages
			if skipReliable {
				continue
			}

			ack, err := s.publishReliable(ctx, msg, m)
			if err != nil {
				s.log.Errorf("Publishing reliable message %s to %s failed, skipping remaining reliable messages: %s", m.ID, msg.Subject, err)
				s.store.IncrementTries(m)
				skipReliable = true
				continue
			}

			s.log.Debugf("Published message %s to stream %s with sequence %d duplicate=%v", m.ID, ack.Stream, ack.Sequence, ack.Duplicate)
		}

		return nil
	})

	<-ctx.Done()
}
