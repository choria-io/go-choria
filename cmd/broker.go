// Copyright (c) 2017-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/choria-io/fisk"
	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/internal/util"
	"github.com/nats-io/jsm.go"
	natscli "github.com/nats-io/natscli/cli"
	log "github.com/sirupsen/logrus"
)

type brokerCommand struct {
	command
}

// broker
func (b *brokerCommand) Setup() (err error) {
	b.cmd = cli.app.Command("broker", "Choria Network Broker and Streams Management Utilities").Alias("b")
	b.cmd.Flag("choria-config", "Choria Config file to use").Hidden().PlaceHolder("FILE").ExistingFileVar(&configFile)

	opts, err := natscli.ConfigureInCommand(b.cmd, &natscli.Options{NoCheats: true, Timeout: 5 * time.Second}, false, "cheat", "backup", "restore", "bench", "schema", "errors", "kv", "object", "micro", "context")
	if err != nil {
		return err
	}

	b.cmd.PreAction(func(pc *fisk.ParseContext) error {
		return b.prepareNatsCli(pc, opts)
	})

	return
}

func (b *brokerCommand) prepareNatsCli(pc *fisk.ParseContext, opts *natscli.Options) error {
	cmd := pc.String()
	if cmd == "broker" || cmd == "broker run" {
		return nil
	}

	for _, e := range pc.Elements {
		f, ok := e.Clause.(*fisk.FlagClause)
		if ok {
			if strings.HasPrefix(f.Model().Name, "help") {
				return nil
			}
		}
	}

	err = commonConfigure()
	if err != nil {
		return err
	}

	c, err = choria.NewWithConfig(cfg)
	if err != nil {
		return err
	}

	natscli.SetContext(ctx)
	natscli.SetVersion(build.Version)

	if strings.HasPrefix(cmd, "broker server") && !util.HasPrefix(cmd, "broker server check stream", "broker server check kv", "broker server check jetstream") {
		if cfg.Choria.NetworkSystemUsername == "" || cfg.Choria.NetworkSystemPassword == "" {
			return fmt.Errorf("the %q command needs system username and password set using plugin.choria.network.system.*", cmd)
		}

		cfg.Choria.NatsUser = cfg.Choria.NetworkSystemUsername
		cfg.Choria.NatsPass = cfg.Choria.NetworkSystemPassword
	}

	connLogger := c.Logger("conn")

	cliLogger := log.New()
	// cli does a lot of Printf which is info
	cliLogger.SetLevel(log.InfoLevel)
	cliLogger.SetOutput(connLogger.Logger.Out)
	natscli.SetLogger(cliLogger)

	conn, err := c.NewConnector(ctx, c.MiddlewareServers, "cli", connLogger)
	if err != nil {
		return err
	}

	opts.Conn = conn.Nats()
	opts.InboxPrefix = conn.InboxPrefix()

	opts.JSc, err = opts.Conn.JetStream()
	if err != nil {
		return err
	}

	var jsmOpts []jsm.Option

	if os.Getenv("TRACE") == "1" {
		cliLogger.Warnf("Tracing Choria Streams API interactions due to TRACE environment variable")
		opts.Trace = true
		jsmOpts = append(jsmOpts, jsm.WithTrace())
	}

	opts.Mgr, err = jsm.New(opts.Conn, jsmOpts...)
	if err != nil {
		return err
	}

	ran = true

	return nil
}

func (b *brokerCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	return
}

func (b *brokerCommand) Configure() error {
	return nil
}

func init() {
	cli.commands = append(cli.commands, &brokerCommand{})
}
