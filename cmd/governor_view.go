// Copyright (c) 2021-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"sync"
	"time"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/internal/util"
	"github.com/nats-io/jsm.go"
)

type tGovViewCommand struct {
	command
	name string
}

func (g *tGovViewCommand) Setup() (err error) {
	if gov, ok := cmdWithFullCommand("governor"); ok {
		g.cmd = gov.Cmd().Command("view", "View the status of a Governor").Alias("info").Alias("show").Alias("v").Alias("s")
		g.cmd.Arg("name", "The name for the Governor to managed").Required().StringVar(&g.name)
	}

	return nil
}

func (g *tGovViewCommand) Configure() (err error) {
	return commonConfigure()
}

func (g *tGovViewCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	gov, conn, err := c.NewGovernorManager(ctx, g.name, 0, 0, 1, false, nil)
	if err != nil {
		return err
	}

	fmt.Printf("Configuration for Governor %s\n", gov.Name())
	fmt.Println()
	fmt.Printf("       Capacity: %d\n", gov.Limit())
	fmt.Printf("        Expires: %v\n", gov.MaxAge())
	fmt.Printf("       Replicas: %d\n", gov.Replicas())

	nfo, err := gov.Stream().Information()
	if err != nil {
		return err
	}

	fmt.Printf("  Active Leases: %d\n", nfo.State.Msgs)
	fmt.Println()

	if nfo.State.Msgs > 0 {
		fmt.Println()
		table := util.NewUTF8Table("ID", "Process Name", "Age")

		sub, err := conn.Nats().SubscribeSync(choria.Inbox(cfg.MainCollective, cfg.Identity))
		if err != nil {
			return err
		}
		defer sub.Unsubscribe()

		_, err = gov.Stream().NewConsumer(jsm.DeliverySubject(sub.Subject), jsm.DeliverAllAvailable(), jsm.AcknowledgeNone())
		if err != nil {
			return err
		}

		for {
			msg, err := sub.NextMsg(time.Second)
			if err != nil {
				return err
			}

			meta, err := jsm.ParseJSMsgMetadata(msg)
			if err != nil {
				continue
			}

			table.AddRow(meta.StreamSequence(), string(msg.Data), fmt.Sprintf("%v", time.Since(meta.TimeStamp()).Round(time.Millisecond)))
			if meta.Pending() == 0 {
				break
			}
		}

		fmt.Println(table.Render())
	}

	return nil
}

func init() {
	cli.commands = append(cli.commands, &tGovViewCommand{})
}
