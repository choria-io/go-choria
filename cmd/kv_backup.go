// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"sync"

	"github.com/nats-io/jsm.go"
)

type kvBackupCommand struct {
	command

	name   string
	target string
}

func (k *kvBackupCommand) Setup() error {
	if kv, ok := cmdWithFullCommand("kv"); ok {
		k.cmd = kv.Cmd().Command("backup", "Backs up a bucket to a directory")
		k.cmd.Arg("bucket", "The bucket name").Required().StringVar(&k.name)
		k.cmd.Arg("target", "Directory to create the backup in").Required().StringVar(&k.target)
	}

	return nil
}

func (k *kvBackupCommand) Configure() error {
	return commonConfigure()
}

func (k *kvBackupCommand) Run(wg *sync.WaitGroup) error {
	defer wg.Done()

	store, conn, err := c.KVWithConn(ctx, nil, k.name, false)
	if err != nil {
		return err
	}

	status, err := store.Status()
	if err != nil {
		return err
	}

	mgr, err := jsm.New(conn.Nats())
	if err != nil {
		return err
	}

	stream, err := mgr.LoadStream(status.BackingStore())
	if err != nil {
		return err
	}

	fp, err := stream.SnapshotToDirectory(ctx, k.target, jsm.SnapshotHealthCheck())
	if err != nil {
		return err
	}

	fmt.Printf("Created %d bytes backup of bucket %s in %s\n", fp.BytesReceived(), k.name, k.target)

	return nil
}

func init() {
	cli.commands = append(cli.commands, &kvBackupCommand{})
}
