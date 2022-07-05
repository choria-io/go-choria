// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/nats-io/nats.go"
)

type kvWatchCommand struct {
	command
	name    string
	key     string
	once    bool
	timeout time.Duration
}

func (k *kvWatchCommand) Setup() error {
	if kv, ok := cmdWithFullCommand("kv"); ok {
		k.cmd = kv.Cmd().Command("watch", "Watch a bucket or key for changes")
		k.cmd.Arg("bucket", "The bucket name").Required().StringVar(&k.name)
		k.cmd.Arg("key", "The key to watch").StringVar(&k.key)
		k.cmd.Flag("once", "Wait for one value and print it, exit after").UnNegatableBoolVar(&k.once)
		k.cmd.Flag("timeout", "Timeout waiting for values").DurationVar(&k.timeout)
	}

	return nil
}

func (k *kvWatchCommand) Configure() error {
	return commonConfigure()
}

func (k *kvWatchCommand) Run(wg *sync.WaitGroup) error {
	defer wg.Done()

	wctx, cancel := context.WithCancel(ctx)
	defer cancel()

	store, err := c.KV(wctx, nil, k.name, false)
	if err != nil {
		return err
	}

	watch, err := store.Watch(k.key)
	if err != nil {
		return err
	}
	defer watch.Stop()

	if k.timeout > 0 {
		go func() {
			wctx, cancel = context.WithTimeout(ctx, k.timeout)
			defer cancel()
			<-wctx.Done()
			watch.Stop()
		}()
	}

	for entry := range watch.Updates() {
		if entry == nil {
			continue
		}

		if k.once {
			if entry.Operation() == nats.KeyValueDelete {
				continue
			}

			os.Stdout.Write(entry.Value())
			return nil
		}

		if entry.Operation() == nats.KeyValueDelete {
			fmt.Printf("[%s] %s %s.%s\n", entry.Created().Format("2006-01-02 15:04:05"), color.RedString("DEL"), entry.Bucket(), entry.Key())
		} else {
			fmt.Printf("[%s] %s %s.%s: %s\n", entry.Created().Format("2006-01-02 15:04:05"), color.GreenString("PUT"), entry.Bucket(), entry.Key(), entry.Value())
		}
	}

	if wctx.Err() != nil {
		if k.once {
			return fmt.Errorf("timeout")
		}
	}

	return nil
}

func init() {
	cli.commands = append(cli.commands, &kvWatchCommand{})
}
