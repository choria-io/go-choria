package cmd

import (
	"fmt"
	"sync"

	"github.com/choria-io/go-choria/internal/util"
	"github.com/nats-io/jsm.go/kv"
)

type kvDelCommand struct {
	command
	name  string
	key   string
	force bool
}

func (k *kvDelCommand) Setup() error {
	if kv, ok := cmdWithFullCommand("kv"); ok {
		k.cmd = kv.Cmd().Command("del", "Deletes a key")
		k.cmd.Arg("bucket", "The bucket name").Required().StringVar(&k.name)
		k.cmd.Arg("key", "The key to delete").Required().StringVar(&k.key)
		k.cmd.Flag("force", "Force delete without prompting").Short('f').BoolVar(&k.force)
	}

	return nil
}

func (k *kvDelCommand) Configure() error {
	return commonConfigure()
}

func (k *kvDelCommand) Run(wg *sync.WaitGroup) error {
	defer wg.Done()

	conn, err := c.NewConnector(ctx, c.MiddlewareServers, fmt.Sprintf("kv manager %s", k.name), c.Logger("kv"))
	if err != nil {
		return err
	}

	store, err := kv.NewBucket(conn.Nats(), k.name)
	if err != nil {
		return err
	}

	if !k.force {
		ok, err := util.PromptForConfirmation("Really remove the %s key from bucket %s", k.key, k.name)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println("Skipping")
			return nil
		}
	}

	return store.Delete(k.key)
}

func init() {
	cli.commands = append(cli.commands, &kvDelCommand{})
}
