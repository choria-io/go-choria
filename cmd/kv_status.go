// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"sync"

	"github.com/nats-io/nats.go"
)

type kvStatusCommand struct {
	command
	name string
}

func (k *kvStatusCommand) Setup() error {
	if kv, ok := cmdWithFullCommand("kv"); ok {
		k.cmd = kv.Cmd().Command("status", "View the status of a bucket").Alias("info")
		k.cmd.Arg("bucket", "The bucket name").Required().StringVar(&k.name)
	}

	return nil
}

func (k *kvStatusCommand) Configure() error {
	return commonConfigure()
}

func (k *kvStatusCommand) Run(wg *sync.WaitGroup) error {
	defer wg.Done()

	store, err := c.KV(ctx, nil, k.name, false)
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
	fmt.Printf("    Values Stored: %d\n", status.Values())
	fmt.Printf("          History: %d\n", status.History())
	fmt.Printf("              TTL: %v\n", status.TTL())
	fmt.Printf("  Max Bucket Size: %d\n", nfo.Config.MaxBytes)
	fmt.Printf("   Max Value Size: %d\n", nfo.Config.MaxMsgSize)
	fmt.Printf(" Storage Replicas: %d\n", nfo.Config.Replicas)

	return nil
}

func init() {
	cli.commands = append(cli.commands, &kvStatusCommand{})
}
