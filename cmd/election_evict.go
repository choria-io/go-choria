// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"sync"

	"github.com/AlecAivazis/survey/v2"
)

type tElectionEvictCommand struct {
	command

	bucket   string
	election string
	force    bool
}

func (w *tElectionEvictCommand) Setup() (err error) {
	if elect, ok := cmdWithFullCommand("election"); ok {
		w.cmd = elect.Cmd().Command("evict", "Evict the current leader from an election")
		w.cmd.Arg("election", "Restrict the watch to a specific election").Required().StringVar(&w.election)
		w.cmd.Flag("bucket", "Use a specific bucket for elections").Default("CHORIA_LEADER_ELECTION").StringVar(&w.bucket)
		w.cmd.Flag("force", "Force eviction without confirmation").UnNegatableBoolVar(&w.force)
	}

	return nil
}

func (w *tElectionEvictCommand) Configure() (err error) {
	return commonConfigure()
}

func (w *tElectionEvictCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	if !w.force {
		ans := false
		err = survey.AskOne(&survey.Confirm{
			Message: fmt.Sprintf("Evict the current leader from election %s in bucket %s", w.election, w.bucket),
			Default: ans,
		}, &ans)
		if err != nil {
			return err
		}
		if !ans {
			return nil
		}
	}

	logger := c.Logger("election")

	conn, err := c.NewConnector(ctx, c.MiddlewareServers, fmt.Sprintf("election %s %s", w.bucket, c.Config.Identity), logger)
	if err != nil {
		return err
	}

	js, err := conn.Nats().JetStream()
	if err != nil {
		return err
	}

	kv, err := js.KeyValue(w.bucket)
	if err != nil {
		return fmt.Errorf("cannot access KV Bucket %s: %v", w.bucket, err)
	}

	err = kv.Delete(w.election)
	if err != nil {
		return err
	}

	fmt.Printf("Evicted the leader from election %s in bucket %s\n", w.election, w.bucket)

	return nil
}

func init() {
	cli.commands = append(cli.commands, &tElectionEvictCommand{})
}
