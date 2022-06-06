// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"sync"
)

type kvCreateCommand struct {
	command
	name  string
	key   string
	value string
}

func (k *kvCreateCommand) Setup() error {
	if kv, ok := cmdWithFullCommand("kv"); ok {
		k.cmd = kv.Cmd().Command("create", "Puts a value into a key only if the key is new or it's last operation was a delete")
		k.cmd.Arg("bucket", "The bucket name").Required().StringVar(&k.name)
		k.cmd.Arg("key", "The key to fetch").Required().StringVar(&k.key)
		k.cmd.Arg("value", "The value to store, when empty reads STDIN").StringVar(&k.value)
	}

	return nil
}

func (k *kvCreateCommand) Configure() error {
	return commonConfigure()
}

func (k *kvCreateCommand) Run(wg *sync.WaitGroup) error {
	defer wg.Done()

	store, err := c.KV(ctx, nil, k.name, false)
	if err != nil {
		return err
	}

	_, err = store.Create(k.key, []byte(k.value))
	if err != nil {
		return err
	}

	fmt.Println(k.value)

	return nil
}

func init() {
	cli.commands = append(cli.commands, &kvCreateCommand{})
}
