// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"sync"

	"github.com/choria-io/go-choria/internal/util"
)

type kvRMCommand struct {
	command

	name  string
	force bool
}

func (k *kvRMCommand) Setup() error {
	if kv, ok := cmdWithFullCommand("kv"); ok {
		k.cmd = kv.Cmd().Command("rm", "Removes a bucket")
		k.cmd.Arg("bucket", "The bucket name").Required().StringVar(&k.name)
		k.cmd.Flag("force", "Remove without prompting").Short('f').BoolVar(&k.force)
	}

	return nil
}

func (k *kvRMCommand) Configure() error {
	return commonConfigure()
}

func (k *kvRMCommand) Run(wg *sync.WaitGroup) error {
	defer wg.Done()

	store, err := c.KV(ctx, nil, k.name, false)
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

	return store.Destroy()
}

func init() {
	cli.commands = append(cli.commands, &kvRMCommand{})
}
