// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/internal/util"
)

type jWTKeyPairCommand struct {
	file  string
	force bool
	command
}

func (k *jWTKeyPairCommand) Setup() (err error) {
	if jwt, ok := cmdWithFullCommand("jwt"); ok {
		k.cmd = jwt.Cmd().Command("keys", "Create an Ed25519 keypair").Alias("k")
		k.cmd.Arg("seed-file", "The Seed file to act on").Required().StringVar(&k.file)
		k.cmd.Flag("force", "Force overwrite existing seed file").Short('f').BoolVar(&k.force)
	}

	return nil
}

func (k *jWTKeyPairCommand) Configure() error {
	cfg, err = config.NewDefaultConfig()
	if err != nil {
		return fmt.Errorf("could not create default configuration: %s", err)
	}

	cfg.DisableSecurityProviderVerify = true
	cfg.Choria.SecurityProvider = "file"

	return nil
}

func (k *jWTKeyPairCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	if choria.FileExist(k.file) && !k.force {
		ok, err := util.PromptForConfirmation("Really overwrite %s", k.file)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println("Skipping")
			return nil
		}
	}

	pub, _, err := choria.Ed25519KeyPairToFile(k.file)
	if err != nil {
		return err
	}

	fmt.Printf("Public Key: %s\n\n", hex.EncodeToString(pub))
	fmt.Printf("Ed25519 seed saved in %s\n", k.file)

	return nil
}

func init() {
	cli.commands = append(cli.commands, &jWTKeyPairCommand{})
}
