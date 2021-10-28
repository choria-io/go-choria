// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package kv

import (
	"fmt"
	"time"

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

func NewKV(nc *nats.Conn, name string, create bool, opts ...Option) (nats.KeyValue, error) {
	opt := &options{
		name:        name,
		replicas:    1,
		description: "Choria Streams Key-Value Bucket",
	}

	for _, o := range opts {
		o(opt)
	}

	js, err := nc.JetStream()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Choria Streams: %s", err)
	}

	kv, err := js.KeyValue(name)
	if err == nil {
		return kv, nil
	}

	if !create {
		return nil, fmt.Errorf("failed to load Choria Key-Value store %s: %s", name, err)
	}

	return js.CreateKeyValue(&nats.KeyValueConfig{
		Bucket:       name,
		Description:  opt.description,
		MaxValueSize: opt.maxValSize,
		History:      opt.history,
		TTL:          opt.ttl,
		MaxBytes:     opt.maxBucketSize,
		Storage:      nats.FileStorage,
		Replicas:     opt.replicas,
	})
}
