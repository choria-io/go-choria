// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"sync"
	"time"

	"github.com/choria-io/go-choria/providers/kv"
	"github.com/nats-io/nats.go"
)

type kvAddCommand struct {
	command

	name          string
	history       uint8
	ttl           time.Duration
	replicas      uint
	maxValueSize  int32
	maxBucketSize int64
	direct        bool
}

func (k *kvAddCommand) Setup() error {
	if kv, ok := cmdWithFullCommand("kv"); ok {
		k.cmd = kv.Cmd().Command("add", "Adds a new bucket")
		k.cmd.Arg("bucket", "The bucket name").Required().StringVar(&k.name)
		k.cmd.Flag("history", "How many historic values to keep for each key").Default("5").Uint8Var(&k.history)
		k.cmd.Flag("ttl", "Expire values from the bucket after this duration").DurationVar(&k.ttl)
		k.cmd.Flag("replicas", "How many data replicas to store").Default("1").UintVar(&k.replicas)
		k.cmd.Flag("max-value-size", "Maximum size of any value in the bucket").Default("10240").PlaceHolder("BYTES").Int32Var(&k.maxValueSize)
		k.cmd.Flag("max-bucket-size", "Maximum size for the entire bucket").PlaceHolder("BYTES").Int64Var(&k.maxBucketSize)
		k.cmd.Flag("direct", "Allow optimized direct access to bucket contents").Default("true").BoolVar(&k.direct)
	}

	return nil
}

func (k *kvAddCommand) Configure() error {
	return commonConfigure()
}

func (k *kvAddCommand) Run(wg *sync.WaitGroup) error {
	defer wg.Done()

	opts := []kv.Option{
		kv.WithTTL(k.ttl),
		kv.WithHistory(k.history),
		kv.WithReplicas(int(k.replicas)),
		kv.WithMaxBucketSize(k.maxBucketSize),
		kv.WithMaxValueSize(k.maxValueSize),
	}

	if !k.direct {
		opts = append(opts, kv.WithoutDirectAccess())
	}

	store, err := c.KV(ctx, nil, k.name, true, opts...)
	if err != nil {
		return err
	}

	status, err := store.Status()
	if err != nil {
		return err
	}

	nfo := status.(*nats.KeyValueBucketStatus).StreamInfo()

	fmt.Printf("%s Key-Value Store\n", status.Bucket())
	fmt.Println()
	fmt.Printf("      Bucket Name: %s\n", status.Bucket())
	fmt.Printf("          History: %d\n", status.History())
	fmt.Printf("              TTL: %v\n", status.TTL())
	if nfo.Config.MaxBytes == -1 {
		fmt.Printf("  Max Bucket Size: unlimited\n")
	} else {
		fmt.Printf("  Max Bucket Size: %d\n", nfo.Config.MaxBytes)
	}
	if nfo.Config.MaxMsgSize == -1 {
		fmt.Printf("   Max Value Size: unlimited\n")
	} else {
		fmt.Printf("   Max Value Size: %d\n", nfo.Config.MaxMsgSize)
	}
	fmt.Printf(" Storage Replicas: %d\n", nfo.Config.Replicas)
	fmt.Printf("       Direct Get: %t\n", nfo.Config.AllowDirect)

	return nil
}

func init() {
	cli.commands = append(cli.commands, &kvAddCommand{})
}
