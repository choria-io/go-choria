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

type mPluginsVerifyCommand struct {
	command
	source string
	key    string
}

func init() {
	cli.commands = append(cli.commands, &mPluginsVerifyCommand{})
}

func (r *mPluginsVerifyCommand) Setup() (err error) {
	if machine, ok := cmdWithFullCommand("machine plugins"); ok {
		r.cmd = machine.Cmd().Command("verify", "Verifies a file made using pack is signed correctly")
		r.cmd.Arg("source", "The signed artifact to validate").ExistingFileVar(&r.source)
		r.cmd.Arg("key", "The ed25519 public key to verify with").StringVar(&r.key)
	}

	return nil
}

func (r *mPluginsVerifyCommand) Configure() error {
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

func (r *mPluginsVerifyCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	data, err := os.ReadFile(r.source)
	if err != nil {
		return err
	}

	var spec watcher.Specification
	err = json.Unmarshal(data, &spec)
	if err != nil {
		return err
	}

	var pk ed25519.PublicKey
	if iu.FileExist(r.key) {
		kdat, err := os.ReadFile(r.key)
		if err != nil {
			return err
		}
		pk, err = hex.DecodeString(string(kdat))
		if err != nil {
			return fmt.Errorf("invalid public key data")
		}
	} else {
		pk, err = hex.DecodeString(r.key)
		if err != nil {
			return err
		}
	}

	ok, err := spec.VerifySignature(pk)
	if err != nil {
		return err
	}

	if !ok {
		fmt.Printf("Data %s does not have a valid signature for key %s", r.source, hex.EncodeToString(pk))
		fmt.Println()
		return fmt.Errorf("verification failed")
	}

	fmt.Printf("Data file %s is signed using %s\n", r.source, hex.EncodeToString(pk))
	return nil
}
