// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

type tElectionWatchCommand struct {
	command

	bucket   string
	election string
}

func (w *tElectionWatchCommand) Setup() (err error) {
	if elect, ok := cmdWithFullCommand("election"); ok {
		w.cmd = elect.Cmd().Command("watch", "Watch election activity in real time")
		w.cmd.Flag("bucket", "Use a specific bucket for elections").Default("CHORIA_LEADER_ELECTION").StringVar(&w.bucket)
		w.cmd.Flag("election", "Restrict the watch to a specific election").Default(">").StringVar(&w.election)
	}

	return nil
}

func (w *tElectionWatchCommand) Configure() (err error) {
	return commonConfigure()
}

func (w *tElectionWatchCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	logger := c.Logger("election")

	conn, err := c.NewConnector(ctx, c.MiddlewareServers, fmt.Sprintf("election %s %s", w.bucket, c.Config.Identity), logger)
	if err != nil {
		return err
	}

	fmt.Printf("Listening for leadership events in %s.\n\nLegend: L=Leader, C=Campaign, U=Unknown\n\n", w.bucket)
	prefix := fmt.Sprintf("$KV.%s.", w.bucket)
	sub, err := conn.Nats().SubscribeSync(fmt.Sprintf("%s%s", prefix, w.election))
	if err != nil {
		return fmt.Errorf("could not subscribe for campaigns: %v", err)
	}

	for {
		msg, err := sub.NextMsgWithContext(ctx)
		if err == context.DeadlineExceeded || err == nats.ErrTimeout {
			continue
		} else if err == context.Canceled {
			return nil
		} else if err != nil {
			return fmt.Errorf("could not load leadership events: %v", err)
		}

		election := strings.TrimPrefix(msg.Subject, prefix)
		seq := msg.Header.Get(nats.ExpectedLastSubjSeqHdr)
		switch seq {
		case "0":
			fmt.Printf("[%s] [C] [%s] %s\n", time.Now().Format("15:04:05.000"), election, string(msg.Data))
		case "":
			fmt.Printf("[%s] [U] [%s] %s\n", time.Now().Format("15:04:05.000"), election, string(msg.Data))
		default:
			fmt.Printf("[%s] [L] [%s] %s\n", time.Now().Format("15:04:05.000"), election, string(msg.Data))
		}
	}
}

func init() {
	cli.commands = append(cli.commands, &tElectionWatchCommand{})
}
