// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"strings"
	"sync"

	"github.com/choria-io/go-choria/internal/util"
)

type kvPurgeCommand struct {
	command
	name  string
	key   string
	force bool
}

func (k *kvPurgeCommand) Setup() error {
	if kv, ok := cmdWithFullCommand("kv"); ok {
		k.cmd = kv.Cmd().Command("purge", "Deletes historical data from a key")
		k.cmd.Arg("bucket", "The bucket name").Required().StringVar(&k.name)
		k.cmd.Arg("key", "The key to purge").Required().StringVar(&k.key)
		k.cmd.Flag("force", "Purge without prompting").Short('f').UnNegatableBoolVar(&k.force)
	}

	return nil
}

func (k *kvPurgeCommand) Configure() error {
	return commonConfigure()
}

func (k *kvPurgeCommand) Run(wg *sync.WaitGroup) error {
	defer wg.Done()

	store, err := c.KV(ctx, nil, k.name, false)
	if err != nil {
		return err
	}

	if !k.force {
		ok, err := util.PromptForConfirmation("Really remove the %s > %s", k.name, k.key)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println("Skipping")
			return nil
		}
	}

	err = store.Purge(k.key)
	if err != nil {
		if strings.Contains(err.Error(), "rollup not permitted") {
			return fmt.Errorf("purge failed, upgrade bucket using 'choria kv upgrade %s'", k.name)
		}
		return err
	}

	return nil
}

func init() {
	cli.commands = append(cli.commands, &kvPurgeCommand{})
}
