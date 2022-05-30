// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/alecthomas/kingpin"
	machines "github.com/choria-io/go-choria/aagent/watchers/machineswatcher"
	"github.com/sirupsen/logrus"
)

var (
	managed string
	key     string
	force   bool
)

func main() {
	app := kingpin.New("mms", "Manage machine specifications")

	app.Command("keys", "Create a key ed25519 pair").Action(keysAction)

	sign := app.Command("pack", "Packs and optionally signs a specification").Action(signAction)
	sign.Arg("machines", "A file holding JSON data describing machines to manage").Required().ExistingFileVar(&managed)
	sign.Arg("key", "The ED25519 private key to encode with").Envar("KEY").StringVar(&key)
	sign.Flag("force", "Do not warn about no ed25519 key").BoolVar(&force)

	kingpin.MustParse(app.Parse(os.Args[1:]))
}

func keysAction(_ *kingpin.ParseContext) error {
	pub, pri, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}

	fmt.Printf(" Public Key: %s\n", hex.EncodeToString(pub))
	fmt.Printf("Private Key: %s\n", hex.EncodeToString(pri))

	return nil
}

func signAction(_ *kingpin.ParseContext) error {
	data, err := os.ReadFile(managed)
	if err != nil {
		return err
	}

	spec := machines.Specification{
		Machines: data,
	}

	if key != "" {
		pk, err := hex.DecodeString(key)
		if err != nil {
			return err
		}

		spec.Signature = hex.EncodeToString(ed25519.Sign(pk, data))
	} else if !force {
		logrus.Warn("No ed25519 private key given encoding without signing")
	}

	j, err := json.Marshal(spec)
	if err != nil {
		return err
	}

	fmt.Println(string(j))

	return nil
}
