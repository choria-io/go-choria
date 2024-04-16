// Copyright (c) 2019-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/choria-io/go-choria/aagent/machine"
	"github.com/choria-io/go-choria/aagent/notifiers/console"
	"github.com/choria-io/go-choria/aagent/watchers"
	"github.com/choria-io/go-choria/config"
	"github.com/sirupsen/logrus"
)

type mRunCommand struct {
	command
	sourceDir string
	factsFile string
	dataFile  string
	connect   bool
}

func (r *mRunCommand) Setup() (err error) {
	if machine, ok := cmdWithFullCommand("machine"); ok {
		r.cmd = machine.Cmd().Command("run", "Runs an autonomous agent locally")
		r.cmd.Arg("source", "Directory containing the machine definition").Required().ExistingDirVar(&r.sourceDir)
		r.cmd.Flag("facts", "JSON format facts file to supply to the machine as run time facts").ExistingFileVar(&r.factsFile)
		r.cmd.Flag("data", "JSON format data file to supply to the machine as run time data").ExistingFileVar(&r.dataFile)
		r.cmd.Flag("connect", "Connects to the Choria Broker when running the autonomous agent").UnNegatableBoolVar(&r.connect)
	}

	return nil
}

func (r *mRunCommand) Configure() error {
	if r.connect {
		return commonConfigure()
	}

	if debug {
		logrus.SetOutput(os.Stdout)
		logrus.SetLevel(logrus.DebugLevel)
		logrus.Debug("Logging at debug level due to CLI override")
	}

	cfg, err = config.NewDefaultConfig()
	if err != nil {
		return err
	}

	cfg.Choria.SecurityProvider = "file"
	cfg.DisableSecurityProviderVerify = true

	return err
}

func (r *mRunCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	if debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	m, err := machine.FromDir(r.sourceDir, watchers.New(ctx))
	if err != nil {
		return err
	}

	m.SetIdentity("cli")
	m.RegisterNotifier(&console.Notifier{})
	m.SetMainCollective(cfg.MainCollective)

	if r.factsFile != "" {
		facts, err := os.ReadFile(r.factsFile)
		if err != nil {
			return err
		}

		m.SetFactSource(func() json.RawMessage { return facts })
	}

	if r.dataFile != "" {
		dat := make(map[string]any)
		df, err := os.ReadFile(r.dataFile)
		if err != nil {
			return err
		}

		err = json.Unmarshal(df, &dat)
		if err != nil {
			return err
		}

		for k, v := range dat {
			err = m.DataPut(k, v)
			if err != nil {
				return err
			}
		}
	}

	if r.connect {
		conn, err := c.NewConnector(ctx, c.MiddlewareServers, "machine run", c.Logger("machine"))
		if err != nil {
			return err
		}

		m.SetConnection(conn)
	}

	<-m.Start(ctx, wg)

	<-ctx.Done()

	return nil
}

func init() {
	cli.commands = append(cli.commands, &mRunCommand{})
}
