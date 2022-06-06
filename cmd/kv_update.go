// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"sync"
)

type kvUpdateCommand struct {
	command
	name  string
	key   string
	value string
	rev   uint64
}

func (k *kvUpdateCommand) Setup() error {
	if kv, ok := cmdWithFullCommand("kv"); ok {
		k.cmd = kv.Cmd().Command("update", "Updates a key with a new value if the previous value matches the given revision")
		k.cmd.Arg("bucket", "The bucket name").Required().StringVar(&k.name)
		k.cmd.Arg("key", "The key to fetch").Required().StringVar(&k.key)
		k.cmd.Arg("value", "The value to store, when empty reads STDIN").StringVar(&k.value)
		k.cmd.Arg("revision", "The revision of the previous value in the bucket").Uint64Var(&k.rev)
	}

	return nil
}

func (k *kvUpdateCommand) Configure() error {
	return commonConfigure()
}

func (k *kvUpdateCommand) Run(wg *sync.WaitGroup) error {
	defer wg.Done()

	store, err := c.KV(ctx, nil, k.name, false)
	if err != nil {
		return err
	}

	rev, err := store.Update(k.key, []byte(k.value), k.rev)
	if err != nil {
		return err
	}

	fmt.Printf("Updated %s, new revision %d\n", k.key, rev)

	return nil
}

func init() {
	cli.commands = append(cli.commands, &kvUpdateCommand{})
}
