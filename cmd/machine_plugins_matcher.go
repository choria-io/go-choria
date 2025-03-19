// Copyright (c) 2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	watcher "github.com/choria-io/go-choria/aagent/watchers/pluginswatcher"
	"github.com/choria-io/go-choria/config"
	"github.com/sirupsen/logrus"
)

type mPluginsMatcherCommand struct {
	command
	facts    string
	identity string
	matcher  string
}

func init() {
	cli.commands = append(cli.commands, &mPluginsMatcherCommand{})
}

func (r *mPluginsMatcherCommand) Setup() (err error) {
	if machine, ok := cmdWithFullCommand("machine plugins"); ok {
		r.cmd = machine.Cmd().Command("matcher", "Tests a plugins matcher expression")
		r.cmd.Arg("matcher", "The matcher expression").Required().StringVar(&r.matcher)
		r.cmd.Flag("identity", "Node identity").StringVar(&r.identity)
		r.cmd.Flag("facts", "File holding JSON format facts").ExistingFileVar(&r.facts)
	}

	return nil
}

func (r *mPluginsMatcherCommand) Configure() error {
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

func (r *mPluginsMatcherCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	var facts json.RawMessage

	if r.facts != "" {
		facts, err = os.ReadFile(r.facts)
		if err != nil {
			return err
		}
	}

	matched, err := watcher.IsNodeMatch(facts, r.identity, r.matcher, logrus.New())
	if err != nil {
		return err
	}

	if matched {
		fmt.Printf("Matched plugin matcher: %s\n", r.matcher)
	} else {
		fmt.Printf("Failed to match plugin matcher: %s\n", r.matcher)
	}

	return nil
}
