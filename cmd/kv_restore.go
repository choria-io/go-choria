// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/nats-io/jsm.go"
	"github.com/tidwall/gjson"
)

type kvRestoreCommand struct {
	command

	source string
}

func (k *kvRestoreCommand) Setup() error {
	if kv, ok := cmdWithFullCommand("kv"); ok {
		k.cmd = kv.Cmd().Command("restore", "Restores a backup from a directory")
		k.cmd.Arg("source", "Directory holding the backup").Required().ExistingDirVar(&k.source)
	}

	return nil
}

func (k *kvRestoreCommand) Configure() error {
	return commonConfigure()
}

func (k *kvRestoreCommand) Run(wg *sync.WaitGroup) error {
	defer wg.Done()

	bj, err := os.ReadFile(filepath.Join(k.source, "backup.json"))
	if err != nil {
		return fmt.Errorf("could not read backup configuration: %s", err)
	}

	res := gjson.GetBytes(bj, "config.name")
	if !res.Exists() {
		return fmt.Errorf("cannot determine bucket name from backup")
	}

	conn, err := c.NewConnector(ctx, c.MiddlewareServers, fmt.Sprintf("kv %s", c.CallerID()), c.Logger("kv"))
	if err != nil {
		return err
	}

	mgr, err := jsm.New(conn.Nats())
	if err != nil {
		return err
	}

	rp, _, err := mgr.RestoreSnapshotFromDirectory(ctx, res.String(), k.source)
	if err != nil {
		return err
	}

	fmt.Printf("Restored %d bytes backup from %s\n", rp.BytesSent(), k.source)

	return nil
}

func init() {
	cli.commands = append(cli.commands, &kvRestoreCommand{})
}
