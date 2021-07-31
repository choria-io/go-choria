package cmd

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/nats-io/jsm.go/kv"
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
		k.cmd.Flag("once", "Wait for one value and print it, exit after").BoolVar(&k.once)
		k.cmd.Flag("timeout", "Timeout waiting for values").DurationVar(&k.timeout)
	}

	return nil
}

func (k *kvWatchCommand) Configure() error {
	return commonConfigure()
}

func (k *kvWatchCommand) Run(wg *sync.WaitGroup) error {
	defer wg.Done()

	wctx := ctx
	var cancel func()
	if k.timeout > 0 {
		wctx, cancel = context.WithTimeout(ctx, k.timeout)
		defer cancel()
	}

	store, err := c.KV(ctx, nil, k.name, false)
	if err != nil {
		return err
	}

	watch, err := store.Watch(wctx, k.key)
	if err != nil {
		return err
	}
	defer watch.Close()

	for entry := range watch.Channel() {
		if entry == nil {
			continue
		}

		if k.once {
			if entry.Operation() == kv.DeleteOperation {
				continue
			}

			os.Stdout.Write(entry.Value())
			return nil
		}

		if entry.Operation() == kv.DeleteOperation {
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
