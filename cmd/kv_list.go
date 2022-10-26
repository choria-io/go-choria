// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"strings"
	"sync"

	"github.com/choria-io/go-choria/internal/util"
	"github.com/nats-io/jsm.go"
)

type kvLSCommand struct {
	command
}

func (k *kvLSCommand) Setup() error {
	if kv, ok := cmdWithFullCommand("kv"); ok {
		k.cmd = kv.Cmd().Command("list", "List buckets").Alias("ls")
	}

	return nil
}

func (k *kvLSCommand) Configure() error {
	return commonConfigure()
}

func (k *kvLSCommand) Run(wg *sync.WaitGroup) error {
	defer wg.Done()

	conn, err := c.NewConnector(ctx, c.MiddlewareServers, "kv manager", c.Logger("kv"))
	if err != nil {
		return err
	}

	mgr, err := jsm.New(conn.Nats())
	if err != nil {
		return err
	}

	found := 0
	table := util.NewUTF8Table("Bucket", "History", "Values")
	mgr.EachStream(&jsm.StreamNamesFilter{Subject: "$KV.>"}, func(s *jsm.Stream) {
		if !jsm.IsKVBucketStream(s.Name()) {
			return
		}

		parts := strings.SplitN(s.Name(), "_", 2)
		if len(parts) != 2 {
			return
		}

		state, err := s.LatestState()
		if err != nil {
			return
		}

		found++
		table.AddRow(parts[1], s.MaxMsgsPerSubject(), state.Msgs)
	})

	if found == 0 {
		fmt.Println("No Key-Value stores found")
		return nil
	}

	fmt.Println(table.Render())

	return nil
}

func init() {
	cli.commands = append(cli.commands, &kvLSCommand{})
}
