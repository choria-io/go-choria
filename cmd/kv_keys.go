// Copyright (c) 2017-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"sort"
	"sync"

	iu "github.com/choria-io/go-choria/internal/util"
)

type kvKeysCommand struct {
	command

	name string
	json bool
}

func (k *kvKeysCommand) Setup() error {
	if kv, ok := cmdWithFullCommand("kv"); ok {
		k.cmd = kv.Cmd().Command("keys", "List the keys in a bucket")
		k.cmd.Arg("bucket", "The bucket name").Required().StringVar(&k.name)
		k.cmd.Arg("json", "Produce the list in JSON format").BoolVar(&k.json)
	}

	return nil
}

func (k *kvKeysCommand) Configure() error {
	return commonConfigure()
}

func (k *kvKeysCommand) Run(wg *sync.WaitGroup) error {
	defer wg.Done()

	store, err := c.KV(ctx, nil, k.name, false)
	if err != nil {
		return err
	}

	keys, err := store.Keys()
	if err != nil {
		return err
	}

	sort.Strings(keys)

	if k.json {
		return iu.DumpJSONIndent(keys)
	}

	for _, key := range keys {
		fmt.Println(key)
	}

	return nil
}

func init() {
	cli.commands = append(cli.commands, &kvKeysCommand{})
}
