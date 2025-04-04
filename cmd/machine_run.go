// Copyright (c) 2019-2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/json"
	"fmt"
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
	sourceDirs []string
	factsFile  string
	dataFile   string
	connect    bool
	machines   map[string]*machine.Machine
	mu         sync.Mutex
}

func (r *mRunCommand) Setup() (err error) {
	r.machines = make(map[string]*machine.Machine)

	if machine, ok := cmdWithFullCommand("machine"); ok {
		r.cmd = machine.Cmd().Command("run", "Runs an autonomous agent locally")
		r.cmd.Arg("source", "Directories containing the machine definitions").Required().ExistingDirsVar(&r.sourceDirs)
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

	for _, sourceDir := range r.sourceDirs {
		m, err := machine.FromDir(sourceDir, watchers.New(ctx))
		if err != nil {
			return err
		}

		r.mu.Lock()
		r.machines[m.MachineName] = m
		r.mu.Unlock()

		m.SetIdentity("cli")
		m.RegisterNotifier(&console.Notifier{})
		m.SetMainCollective(cfg.MainCollective)
		m.SetExternalMachineNotifier(r.notifyMachinesAfterTransition)
		m.SetExternalMachineStateQuery(r.machineStateLookup)

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
	}

	<-ctx.Done()

	return nil
}

func (r *mRunCommand) notifyMachinesAfterTransition(event *machine.TransitionNotification) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, m := range r.machines {
		logrus.Infof("Notifying %s about event %s from %s", m.MachineName, event.Transition, event.Machine)
		go m.ExternalEventNotify(event)
	}
}

func (r *mRunCommand) machineStateLookup(name string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, m := range r.machines {
		if m.Name() == name {
			return m.State(), nil
		}
	}

	return "", fmt.Errorf("could not find machine matching ame='%s'", name)
}

func init() {
	cli.commands = append(cli.commands, &mRunCommand{})
}
