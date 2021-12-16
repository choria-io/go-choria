// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"sync"

	"github.com/choria-io/go-choria/internal/util"
	"github.com/choria-io/go-choria/providers/kv"
)

type kvDelCommand struct {
	command
	name  string
	key   string
	force bool
}

func (k *kvDelCommand) Setup() error {
	if kv, ok := cmdWithFullCommand("kv"); ok {
		k.cmd = kv.Cmd().Command("del", "Deletes a key or bucket").Alias("rm")
		k.cmd.Arg("bucket", "The bucket name").Required().StringVar(&k.name)
		k.cmd.Arg("key", "The key to delete, will preserve history").StringVar(&k.key)
		k.cmd.Flag("force", "Force delete without prompting").Short('f').BoolVar(&k.force)
	}

	return nil
}

func (k *kvDelCommand) Configure() error {
	return commonConfigure()
}

func (k *kvDelCommand) deleteKey() error {
	store, err := c.KV(ctx, nil, k.name, false)
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

func (k *kvDelCommand) deleteBucket() error {
	store, conn, err := c.KVWithConn(ctx, nil, k.name, false)
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

	return kv.DeleteKV(conn.Nats(), store)
}

func (k *kvDelCommand) Run(wg *sync.WaitGroup) error {
	defer wg.Done()

	if k.key != "" {
		return k.deleteKey()
	}

	return k.deleteBucket()
}

func init() {
	cli.commands = append(cli.commands, &kvDelCommand{})
}
