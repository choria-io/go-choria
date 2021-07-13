package cmd

import (
	"fmt"
	"sync"

	"github.com/fatih/color"
	"github.com/nats-io/jsm.go/kv"
)

type kvWatchCommand struct {
	command
	name string
	key  string
}

func (k *kvWatchCommand) Setup() error {
	if kv, ok := cmdWithFullCommand("kv"); ok {
		k.cmd = kv.Cmd().Command("watch", "Watch a bucket or key for changes")
		k.cmd.Arg("bucket", "The bucket name").Required().StringVar(&k.name)
		k.cmd.Arg("key", "The key to watch").StringVar(&k.key)
	}

	return nil
}

func (k *kvWatchCommand) Configure() error {
	return commonConfigure()
}

func (k *kvWatchCommand) Run(wg *sync.WaitGroup) error {
	defer wg.Done()

	conn, err := c.NewConnector(ctx, c.MiddlewareServers, fmt.Sprintf("kv manager %s", k.name), c.Logger("kv"))
	if err != nil {
		return err
	}

	store, err := kv.NewClient(conn.Nats(), k.name)
	if err != nil {
		return err
	}

	var watch kv.Watch
	if k.key == "" {
		watch, err = store.Watch(ctx, k.key)
	} else {
		watch, err = store.WatchBucket(ctx)
	}
	if err != nil {
		return err
	}
	defer watch.Close()

	for res := range watch.Channel() {
		if res != nil {
			if res.Operation() == kv.DeleteOperation {
				fmt.Printf("[%s] %s %s.%s\n", res.Created().Format("2006-01-02 15:04:05"), color.RedString("DEL"), res.Bucket(), res.Key())
			} else {
				fmt.Printf("[%s] %s %s.%s: %s\n", res.Created().Format("2006-01-02 15:04:05"), color.GreenString("PUT"), res.Bucket(), res.Key(), res.Value())
			}
		}
	}

	return nil
}

func init() {
	cli.commands = append(cli.commands, &kvWatchCommand{})
}
