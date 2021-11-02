// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"sync"

	"github.com/nats-io/jsm.go"
	"github.com/nats-io/nats.go"
)

type kvUpgradeCommand struct {
	command

	name string
}

func (k *kvUpgradeCommand) Setup() error {
	if kv, ok := cmdWithFullCommand("kv"); ok {
		k.cmd = kv.Cmd().Command("upgrade", "Upgrades a bucket configuration for >= Choria 0.25")
		k.cmd.Arg("bucket", "The bucket name").Required().StringVar(&k.name)
	}

	return nil
}

func (k *kvUpgradeCommand) Configure() error {
	return commonConfigure()
}

func (k *kvUpgradeCommand) Run(wg *sync.WaitGroup) error {
	defer wg.Done()

	store, conn, err := c.KVWithConn(ctx, nil, k.name, false)
	if err != nil {
		return err
	}

	status, err := store.Status()
	if err != nil {
		return err
	}

	nfo := status.(*nats.KeyValueBucketStatus).StreamInfo()
	if nfo.Config.AllowRollup {
		fmt.Printf("Configuration for %s is already up to date\n", k.name)
		return nil
	}

	mgr, err := jsm.New(conn.Nats())
	if err != nil {
		return err
	}

	stream, err := mgr.LoadStream(nfo.Config.Name)
	if err != nil {
		return err
	}

	err = stream.UpdateConfiguration(stream.Configuration(), jsm.AllowRollup())
	if err != nil {
		return err
	}

	fmt.Printf("Configuration for %s has been updated\n", k.name)

	return nil
}

func init() {
	cli.commands = append(cli.commands, &kvUpgradeCommand{})
}
