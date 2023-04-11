// Copyright (c) 2023, R.I. Pienaar and the Choria Project contributors
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
	iu "github.com/choria-io/go-choria/internal/util"
	"github.com/sirupsen/logrus"
)

type mPluginsPackCommand struct {
	command
	source string
	key    string
	out    string
	force  bool
}

func (r *mPluginsPackCommand) Setup() (err error) {
	if machine, ok := cmdWithFullCommand("machine plugins"); ok {
		r.cmd = machine.Cmd().Command("pack", "Encodes and signs data for the plugins watcher")
		r.cmd.Arg("source", "File containing the plugins definition").Required().ExistingFileVar(&r.source)
		r.cmd.Arg("key", "The ed25519 private key to encode with").StringVar(&r.key)
		r.cmd.Flag("force", "Do not warn about no ed25519 key and support writing empty files").BoolVar(&r.force)
		r.cmd.Flag("output", "Write result to a file").StringVar(&r.out)
	}

	return nil
}

func (r *mPluginsPackCommand) Configure() error {
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

func (r *mPluginsPackCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	data, err := os.ReadFile(r.source)
	if err != nil {
		return err
	}

	var t []watcher.ManagedPlugin
	err = json.Unmarshal(data, &t)
	if err != nil {
		return fmt.Errorf("invalid specification: %v", err)
	}

	if len(t) == 0 && !r.force {
		return fmt.Errorf("no plugins listed in specification, use --force to write an empty list")
	}

	if r.key == "" && !r.force {
		logrus.Warn("No ed25519 private key given, encoding without signing")
	}

	spec := &watcher.Specification{Plugins: data}
	j, err := spec.Encode(r.key)
	if err != nil {
		return err
	}

	if r.out == "" {
		fmt.Println(string(j))
		return nil
	}

	if iu.FileExist(r.out) && !r.force {
		return fmt.Errorf("output file %s exist, use --force to overwrite", r.out)
	}

	return os.WriteFile(r.out, j, 0600)
}

func init() {
	cli.commands = append(cli.commands, &mPluginsPackCommand{})
}
