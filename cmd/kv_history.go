// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"sync"
	"time"

	"github.com/choria-io/go-choria/internal/util"
	"github.com/nats-io/nats.go"
)

type kvHistoryCommand struct {
	command
	name string
	key  string
}

func (k *kvHistoryCommand) Setup() error {
	if kv, ok := cmdWithFullCommand("kv"); ok {
		k.cmd = kv.Cmd().Command("history", "View the history for a specific key")
		k.cmd.Arg("bucket", "The bucket name").Required().StringVar(&k.name)
		k.cmd.Arg("key", "The key to retrieve history for").Required().StringVar(&k.key)
	}

	return nil
}

func (k *kvHistoryCommand) Configure() error {
	return commonConfigure()
}

func (k *kvHistoryCommand) Run(wg *sync.WaitGroup) error {
	defer wg.Done()

	store, err := c.KV(ctx, nil, k.name, false)
	if err != nil {
		return err
	}

	history, err := store.History(k.key)
	if err != nil {
		return err
	}

	table := util.NewUTF8Table("Seq", "Operation", "Time", "Length", "Value")
	for _, r := range history {
		val := util.Base64IfNotPrintable(r.Value())
		if len(val) > 40 {
			val = fmt.Sprintf("%s...%s", val[0:15], val[len(val)-15:])
		}

		var op string

		switch r.Operation() {
		case nats.KeyValuePurge:
			op = "PURGE"
		case nats.KeyValueDelete:
			op = "DELETE"
		case nats.KeyValuePut:
			op = "PUT"
		default:
			op = r.Operation().String()
		}

		table.AddRow(r.Revision(), op, r.Created().Format(time.RFC822), len(r.Value()), val)
	}

	fmt.Println(table.Render())

	return nil
}

func init() {
	cli.commands = append(cli.commands, &kvHistoryCommand{})
}
