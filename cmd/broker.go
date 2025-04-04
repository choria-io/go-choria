// Copyright (c) 2017-2025, R.I. Pienaar and the Choria Project contributors
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
	"github.com/nats-io/jsm.go"
	"github.com/nats-io/nats.go/jetstream"
	natscli "github.com/nats-io/natscli/cli"
	"github.com/nats-io/natscli/options"
	log "github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
)

type brokerCommand struct {
	command
	timeout time.Duration
}

// broker
func (b *brokerCommand) Setup() (err error) {
	b.cmd = cli.app.Command("broker", "Choria Network Broker and Streams Management Utilities").Alias("b")
	b.cmd.Flag("choria-config", "Choria Config file to use").Hidden().PlaceHolder("FILE").ExistingFileVar(&configFile)
	b.cmd.Flag("connect-timeout", "Connection timeout").Default("5s").DurationVar(&b.timeout)

	opts, err := natscli.ConfigureInCommand(b.cmd, &options.Options{NoCheats: true, Timeout: 5 * time.Second}, false,
		"audit", "run", "cheat", "rtt", "latency", "backup", "restore", "bench", "schema", "errors", "kv", "object", "micro", "context", "auth", "service")
	if err != nil {
		return err
	}

	b.cmd.PreAction(func(pc *fisk.ParseContext) error {
		return b.prepareNatsCli(pc, opts)
	})

	return
}

func (b *brokerCommand) prepareNatsCli(pc *fisk.ParseContext, opts *options.Options) error {
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

	should := []bool{
		strings.HasPrefix(cmd, "broker top"),
		strings.HasPrefix(cmd, "broker server") && (!strings.HasPrefix(cmd, "broker server check stream") &&
			!strings.HasPrefix(cmd, "broker server check kv") &&
			!strings.HasPrefix(cmd, "broker server check jetstream") &&
			!strings.HasPrefix(cmd, "broker server check consumer") &&
			!strings.HasPrefix(cmd, "broker server check connection") &&
			!strings.HasPrefix(cmd, "broker server check request") &&
			!strings.HasPrefix(cmd, "broker server check credential") &&
			!strings.HasPrefix(cmd, "broker server check message")),
	}

	if slices.Contains(should, true) {
		if cfg.Choria.NetworkSystemUsername == "" || cfg.Choria.NetworkSystemPassword == "" {
			return fmt.Errorf("the %q command needs system username and password set using plugin.choria.network.system.*", cmd)
		}

		cfg.Choria.NatsUser = cfg.Choria.NetworkSystemUsername
		cfg.Choria.NatsPass = cfg.Choria.NetworkSystemPassword
	}

	connLogger := c.Logger("conn")

	cliLogger := log.New()
	if debug {
		cliLogger.SetLevel(log.DebugLevel)
		connLogger.Logger.SetLevel(log.DebugLevel)
	} else {
		// cli does a lot of Printf which is info
		cliLogger.SetLevel(log.InfoLevel)
	}
	cliLogger.SetOutput(connLogger.Logger.Out)
	natscli.SetLogger(cliLogger)

	var to *time.Timer
	if b.timeout > 0 {
		to = time.AfterFunc(b.timeout, func() {
			connLogger.Warnf("Initial connection timeout, shutting down, adjust using --connect-timeout")
		})
	}

	conn, err := c.NewConnector(ctx, c.MiddlewareServers, "cli", connLogger)
	if err != nil {
		return err
	}

	if b != nil {
		to.Stop()
	}

	opts.Conn = conn.Nats()
	opts.InboxPrefix = conn.InboxPrefix()

	opts.JSc, err = jetstream.New(opts.Conn)
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
