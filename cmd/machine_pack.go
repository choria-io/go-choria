// Copyright (c) 2023, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	watcher "github.com/choria-io/go-choria/aagent/watchers/pluginswatcher"
	"github.com/choria-io/go-choria/config"
	iu "github.com/choria-io/go-choria/internal/util"
	"github.com/sirupsen/logrus"
)

type mPackCommand struct {
	command
	machines string
	key      string
	out      string
	force    bool
}

func (r *mPackCommand) Setup() (err error) {
	if machine, ok := cmdWithFullCommand("machine"); ok {
		r.cmd = machine.Cmd().Command("plugins", "Encodes and signs data for the plugins watcher")
		r.cmd.Arg("source", "File containing the plugins definition").Required().ExistingFileVar(&r.machines)
		r.cmd.Arg("key", "The ed25519 private key to encode with").StringVar(&r.key)
		r.cmd.Flag("force", "Do not warn about no ed25519 key and support writing empty files").BoolVar(&r.force)
		r.cmd.Flag("output", "Write result to a file").StringVar(&r.out)
	}

	return nil
}

func (r *mPackCommand) Configure() error {
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

func (r *mPackCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	data, err := os.ReadFile(r.machines)
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

	spec := watcher.Specification{Plugins: data}

	if r.key != "" {
		var key []byte
		if iu.FileExist(r.key) {
			key, err = os.ReadFile(r.key)
		} else {
			key, err = hex.DecodeString(r.key)
		}
		if err != nil {
			return err
		}

		spec.Signature = hex.EncodeToString(ed25519.Sign(key, data))
	} else if !r.force {
		logrus.Warn("No ed25519 private key given, encoding without signing")
	}

	j, err := json.Marshal(spec)
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
	cli.commands = append(cli.commands, &mPackCommand{})
}
