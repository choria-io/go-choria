// Copyright (c) 2021-2024, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package kv

import (
	"fmt"
	"time"

	"github.com/nats-io/jsm.go"
	"github.com/nats-io/jsm.go/api"
	"github.com/nats-io/nats.go"
)

// Option configures a KV bucket
type Option func(*options)

type options struct {
	name          string
	description   string
	maxValSize    int32
	history       uint8
	ttl           time.Duration
	maxBucketSize int64
	replicas      int
	direct        bool
}

func WithTTL(ttl time.Duration) Option {
	return func(o *options) { o.ttl = ttl }
}

func WithHistory(h uint8) Option {
	return func(o *options) { o.history = h }
}

func WithReplicas(r int) Option {
	return func(o *options) { o.replicas = r }
}

func WithMaxBucketSize(s int64) Option {
	return func(o *options) { o.maxBucketSize = s }
}

func WithMaxValueSize(s int32) Option {
	return func(o *options) { o.maxValSize = s }
}

func WithoutDirectAccess() Option {
	return func(o *options) { o.direct = false }
}

func DeleteKV(nc *nats.Conn, kv nats.KeyValue) error {
	status, err := kv.Status()
	if err != nil {
		return err
	}
	nfo := status.(*nats.KeyValueBucketStatus).StreamInfo()

	js, err := nc.JetStream()
	if err != nil {
		return err
	}

	return js.DeleteStream(nfo.Config.Name)
}

func LoadKV(nc *nats.Conn, name string) (nats.KeyValue, error) {
	js, err := nc.JetStream()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Choria Streams: %s", err)
	}

	return js.KeyValue(name)
}

func NewKV(nc *nats.Conn, name string, create bool, opts ...Option) (nats.KeyValue, error) {
	opt := &options{
		name:        name,
		replicas:    1,
		direct:      true,
		description: "Choria Streams Key-Value Bucket",
	}

	for _, o := range opts {
		o(opt)
	}

	kv, err := LoadKV(nc, name)
	if err == nil {
		return kv, nil
	}

	if !create {
		return nil, fmt.Errorf("failed to load Choria Key-Value store %s: %s", name, err)
	}

	mgr, err := jsm.New(nc)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Choria Streams: %s", err)
	}

	cfg := api.StreamConfig{
		Name:          fmt.Sprintf("KV_%s", name),
		Subjects:      []string{fmt.Sprintf("$KV.%s.>", name)},
		Retention:     api.LimitsPolicy,
		MaxMsgsPer:    int64(opt.history),
		MaxBytes:      opt.maxBucketSize,
		MaxAge:        opt.ttl,
		Replicas:      opt.replicas,
		AllowDirect:   opt.direct,
		MaxConsumers:  -1,
		MaxMsgs:       -1,
		MaxMsgSize:    -1,
		Storage:       api.FileStorage,
		Discard:       api.DiscardNew,
		Duplicates:    2 * time.Minute,
		RollupAllowed: true,
		DenyDelete:    true,
	}

	if cfg.Duplicates > cfg.MaxAge {
		cfg.Duplicates = cfg.MaxAge
	}

	_, err = mgr.NewStreamFromDefault(cfg.Name, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create key-value bucket: %s", err)
	}

	return LoadKV(nc, name)
}
