// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/jsm.go/kv"
)

type kvAddCommand struct {
	command

	name          string
	history       uint64
	ttl           time.Duration
	replicas      uint
	maxValueSize  int32
	maxBucketSize int64
}

func (k *kvAddCommand) Setup() error {
	if kv, ok := cmdWithFullCommand("kv"); ok {
		k.cmd = kv.Cmd().Command("add", "Adds a new bucket")
		k.cmd.Arg("bucket", "The bucket name").Required().StringVar(&k.name)
		k.cmd.Flag("history", "How many historic values to keep for each key").Default("5").Uint64Var(&k.history)
		k.cmd.Flag("ttl", "Expire values from the bucket after this duration").DurationVar(&k.ttl)
		k.cmd.Flag("replicas", "How many data replicas to store").Default("1").UintVar(&k.replicas)
		k.cmd.Flag("max-value-size", "Maximum size of any value in the bucket").Default("10240").Int32Var(&k.maxValueSize)
		k.cmd.Flag("max-bucket-size", "Maximum size for the entire bucket").Int64Var(&k.maxBucketSize)
	}

	return nil
}

func (k *kvAddCommand) Configure() error {
	return commonConfigure()
}

func (k *kvAddCommand) Run(wg *sync.WaitGroup) error {
	defer wg.Done()

	store, err := c.KV(ctx, nil, k.name, true, kv.WithTTL(k.ttl),
		kv.WithHistory(k.history),
		kv.WithReplicas(k.replicas),
		kv.WithMaxBucketSize(k.maxBucketSize),
		kv.WithMaxValueSize(k.maxValueSize),
		kv.WithLogger(c.Logger("kv")))
	if err != nil {
		return err
	}

	status, err := store.Status()
	if err != nil {
		return err
	}

	fmt.Printf("%s Key-Value Store\n", status.Bucket())
	fmt.Println()
	fmt.Printf("     Bucket Name: %s\n", status.Bucket())
	fmt.Printf("         History: %d\n", status.History())
	fmt.Printf("             TTL: %v\n", status.TTL())
	fmt.Printf(" Max Bucket Size: %d\n", status.MaxBucketSize())
	fmt.Printf("  Max Value Size: %d\n", status.MaxValueSize())

	return nil
}

func init() {
	cli.commands = append(cli.commands, &kvAddCommand{})
}
