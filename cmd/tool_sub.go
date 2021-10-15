// Copyright (c) 2018-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"crypto/md5"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/choria-io/go-choria/inter"
)

type tSubCommand struct {
	command
	subject string
	raw     bool
}

func (s *tSubCommand) Setup() (err error) {
	if tool, ok := cmdWithFullCommand("tool"); ok {
		s.cmd = tool.Cmd().Command("sub", "Subscribe to middleware topics")
		s.cmd.Arg("subject", "The subject to subscribe to").StringVar(&s.subject)
		s.cmd.Flag("raw", "Display raw messages one per line without timestamps").BoolVar(&s.raw)
	}

	return nil
}

func (s *tSubCommand) Configure() error {
	return commonConfigure()
}

func (s *tSubCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	if s.subject == "" {
		s.subject = fmt.Sprintf("%s.reply.%s.%s", c.Config.MainCollective, fmt.Sprintf("%x", md5.Sum([]byte(c.CallerID()))), strings.Replace(c.UniqueID(), "-", "", -1))
	}

	log := c.Logger("sub")
	conn, err := c.NewConnector(ctx, c.MiddlewareServers, c.Certname(), log)
	if err != nil {
		return fmt.Errorf("cannot connect: %s", err)
	}

	if !s.raw {
		fmt.Printf("Waiting for messages from topic %s on %s\n", s.subject, conn.ConnectedServer())
	}

	msgs := make(chan inter.ConnectorMessage, 100)

	err = conn.QueueSubscribe(ctx, c.UniqueID(), s.subject, "", msgs)
	if err != nil {
		return fmt.Errorf("could not subscribe to %s: %s", s.subject, err)
	}

	for {
		select {
		case m := <-msgs:
			if s.raw {
				fmt.Println(string(m.Data()))
				continue
			}

			if m.Subject() == s.subject {
				fmt.Printf("---- %s\n%s\n\n", time.Now().Format("15:04:05"), string(m.Data()))
			} else {
				fmt.Printf("---- %s on topic %s\n%s\n\n", time.Now().Format("15:04:05"), m.Subject(), string(m.Data()))
			}

		case <-ctx.Done():
			return nil
		}
	}
}

func init() {
	cli.commands = append(cli.commands, &tSubCommand{})
}
