// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/choria-io/go-choria/internal/util"
	"golang.org/x/term"
)

type kvGetCommand struct {
	command
	name string
	key  string
	raw  bool
}

func (k *kvGetCommand) Setup() error {
	if kv, ok := cmdWithFullCommand("kv"); ok {
		k.cmd = kv.Cmd().Command("get", "Get a value")
		k.cmd.Arg("bucket", "The bucket name").Required().StringVar(&k.name)
		k.cmd.Arg("key", "The key to fetch").Required().StringVar(&k.key)
		k.cmd.Flag("raw", "Show only the value").UnNegatableBoolVar(&k.raw)
	}

	return nil
}

func (k *kvGetCommand) Configure() error {
	return commonConfigure()
}

func (k *kvGetCommand) Run(wg *sync.WaitGroup) error {
	defer wg.Done()

	store, err := c.KV(ctx, nil, k.name, false)
	if err != nil {
		return err
	}

	entry, err := store.Get(k.key)
	if err != nil {
		return err
	}

	if k.raw {
		os.Stdout.Write(entry.Value())
		return nil
	}

	fmt.Printf("%s > %s sequence %d created @ %s\n\n", entry.Bucket(), entry.Key(), entry.Revision(), entry.Created().Format(time.RFC822))

	width, _, err := term.GetSize(0)
	if err != nil {
		width = 80
	}

	pv := util.Base64IfNotPrintable(entry.Value())
	lpv := len(pv)

	if lpv > width {
		fmt.Printf("Showing first %d characters, use --raw for full value\n\n", width)
		fmt.Println(pv[:width])
	} else {
		fmt.Println(util.Base64IfNotPrintable(entry.Value()))
	}

	return nil
}

func init() {
	cli.commands = append(cli.commands, &kvGetCommand{})
}
