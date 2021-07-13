package cmd

import (
	"fmt"
	"sync"

	"github.com/choria-io/go-choria/internal/util"
	"github.com/nats-io/jsm.go/kv"
)

type kvPurgeCommand struct {
	command
	name  string
	force bool
}

func (k *kvPurgeCommand) Setup() error {
	if kv, ok := cmdWithFullCommand("kv"); ok {
		k.cmd = kv.Cmd().Command("purge", "Remove all keys from a bucket")
		k.cmd.Arg("bucket", "The bucket name").Required().StringVar(&k.name)
		k.cmd.Flag("force", "Purge without prompting").Short('f').BoolVar(&k.force)
	}

	return nil
}

func (k *kvPurgeCommand) Configure() error {
	return commonConfigure()
}

func (k *kvPurgeCommand) Run(wg *sync.WaitGroup) error {
	defer wg.Done()

	conn, err := c.NewConnector(ctx, c.MiddlewareServers, "kv manager", c.Logger("kv"))
	if err != nil {
		return err
	}

	store, err := kv.NewBucket(conn.Nats(), k.name)
	if err != nil {
		return err
	}

	if !k.force {
		ok, err := util.PromptForConfirmation("Really remove the %s bucket", k.name)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println("Skipping")
			return nil
		}
	}

	return store.Purge()
}

func init() {
	cli.commands = append(cli.commands, &kvPurgeCommand{})
}
