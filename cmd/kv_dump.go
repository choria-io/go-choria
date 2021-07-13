package cmd

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/nats-io/jsm.go/kv"
)

type kvDumpCommand struct {
	command
	name string
}

func (k *kvDumpCommand) Setup() error {
	if kv, ok := cmdWithFullCommand("kv"); ok {
		k.cmd = kv.Cmd().Command("dump", "Dumps all values for a bucket")
		k.cmd.Arg("bucket", "The bucket name").Required().StringVar(&k.name)
	}

	return nil
}

func (k *kvDumpCommand) Configure() error {
	return commonConfigure()
}

func (k *kvDumpCommand) Run(wg *sync.WaitGroup) error {
	defer wg.Done()

	_, store, err := c.KV(ctx, k.name)
	if err != nil {
		return err
	}

	vals := make(map[string]kv.Entry)
	watch, err := store.WatchBucket(ctx)
	if err != nil {
		return err
	}
	defer watch.Close()

	for entry := range watch.Channel() {
		if entry == nil {
			break
		}

		switch entry.Operation() {
		case kv.PutOperation:
			vals[entry.Key()] = entry
		case kv.DeleteOperation:
			delete(vals, entry.Key())
		}

		if entry.Delta() == 0 {
			break
		}
	}

	j, err := json.MarshalIndent(vals, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(j))

	return nil
}

func init() {
	cli.commands = append(cli.commands, &kvDumpCommand{})
}
