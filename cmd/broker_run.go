// Copyright (c) 2022-2023, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/choria-io/go-choria/broker/adapter"
	"github.com/choria-io/go-choria/broker/federation"
	"github.com/choria-io/go-choria/broker/network"
	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/statistics"
	log "github.com/sirupsen/logrus"
)

type brokerRunCommand struct {
	command

	disableTLS       bool
	disableTLSVerify bool
	pidFile          string

	server     *network.Server
	federation *federation.FederationBroker
}

// broker run
func (r *brokerRunCommand) Setup() (err error) {
	if broker, ok := cmdWithFullCommand("broker"); ok {
		r.cmd = broker.Cmd().Command("run", "Runs a Choria Network Broker instance")
		r.cmd.Flag("config", "Config file to use").PlaceHolder("FILE").StringVar(&configFile)
		r.cmd.Flag("disable-tls", "Disables TLS").Hidden().Default("false").UnNegatableBoolVar(&r.disableTLS)
		r.cmd.Flag("disable-ssl-verification", "Disables SSL Verification").Hidden().Default("false").UnNegatableBoolVar(&r.disableTLSVerify)
		r.cmd.Flag("pid", "Write running PID to a file").StringVar(&r.pidFile)
	}

	return
}

func (r *brokerRunCommand) Configure() error {
	if configFile == "" {
		return fmt.Errorf("configuration file required")
	}

	cfg, err = config.NewSystemConfig(configFile, true)
	if err != nil {
		return fmt.Errorf("could not parse configuration: %s", err)
	}

	cfg.ApplyBuildSettings(bi)

	return nil
}

func (r *brokerRunCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	net := cfg.Choria.BrokerNetwork
	federation := cfg.Choria.BrokerFederation
	adapters := cfg.Choria.BrokerAdapters
	cfg.SoftShutdownTimeout = cfg.Choria.NetworkSoftShutdownTimeout

	if !net && !federation && len(adapters) == 0 {
		return fmt.Errorf("all broker features are disabled")
	}

	log.Infof("Choria Broker version %s starting with config %s", bi.Version(), c.Config.ConfigFile)

	if r.pidFile != "" {
		err := os.WriteFile(r.pidFile, []byte(fmt.Sprintf("%d", os.Getpid())), 0644)
		if err != nil {
			return fmt.Errorf("could not write PID: %s", err)
		}
	}

	if r.disableTLS {
		c.Config.DisableTLS = true
		log.Warn("Running with TLS disabled, not compatible with production use.")
	}

	if r.disableTLSVerify {
		c.Config.DisableTLSVerify = true
		log.Warn("Running with TLS Verification disabled, not compatible with production use.")
	}

	if net {
		log.Info("Starting Network Broker")
		if err = r.runBroker(ctx, wg); err != nil {
			return fmt.Errorf("starting the network broker failed: %s", err)
		}
	}

	if federation {
		log.Infof("Starting Federation Broker on cluster %s", c.Config.Choria.FederationCluster)
		if err = r.runFederation(ctx, wg); err != nil {
			return fmt.Errorf("starting the federation broker failed: %s", err)
		}
	}

	if len(adapters) > 0 {
		log.Info("Starting Protocol Adapters")

		wg.Add(1)
		go r.runAdapters(ctx, wg)
	}

	r.startStats()

	return
}

func (r *brokerRunCommand) runAdapters(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	cfg, err := config.NewSystemConfig(cfg.ConfigFile, true)
	if err != nil {
		log.Errorf("Failed to configure Protocol Adapters: %v", err)
		return
	}

	fw, err := choria.NewWithConfig(cfg)
	if err != nil {
		log.Errorf("Failed to run Protocol Adapters: %v", err)
		return
	}

	// when we are running in-process with a broker, we connect to that broker
	if r.server != nil {
		fw.SetInProcessConnProvider(r.server)
	}

	err = adapter.RunAdapters(ctx, fw, wg)
	if err != nil {
		log.Errorf("Failed to run Protocol Adapters: %s", err)
	}
}

func (r *brokerRunCommand) runFederation(ctx context.Context, wg *sync.WaitGroup) (err error) {
	r.federation, err = federation.NewFederationBroker(c.Config.Choria.FederationCluster, c)
	if err != nil {
		return fmt.Errorf("could not set up Choria Federation Broker: %s", err)
	}

	wg.Add(1)
	go r.federation.Start(ctx, wg)

	return
}

func (r *brokerRunCommand) runBroker(ctx context.Context, wg *sync.WaitGroup) (err error) {
	r.server, err = network.NewServer(c, bi, debug)
	if err != nil {
		return fmt.Errorf("could not set up Choria Network Broker: %s", err)
	}

	wg.Add(1)
	go r.server.Start(ctx, wg)

	return
}

func (r *brokerRunCommand) startStats() {
	var handler http.Handler

	if r.server != nil {
		for {
			if r.server.Started() {
				break
			}

			time.Sleep(100 * time.Millisecond)
		}

		handler = r.server.HTTPHandler()
	} else {
		handler = nil
	}

	statistics.Start(c, handler)
}

func init() {
	cli.commands = append(cli.commands, &brokerRunCommand{})
}
