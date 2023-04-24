// Copyright (c) 2021-2023, R.I. Pienaar and the Choria Project contributors
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

	"github.com/choria-io/fisk"
	machines "github.com/choria-io/go-choria/aagent/watchers/machineswatcher"
	iu "github.com/choria-io/go-choria/internal/util"
	"github.com/sirupsen/logrus"
)

var (
	managed string
	key     string
	force   bool
)

func main() {
	app := fisk.New("mms", "Manage machine specifications")

	app.Command("keys", "Create a key ed25519 pair").Action(keysAction)

	sign := app.Command("pack", "Packs and optionally signs a specification").Action(signAction)
	sign.Arg("machines", "A file holding JSON data describing machines to manage").Required().ExistingFileVar(&managed)
	sign.Arg("key", "The ED25519 private key to encode with").Envar("KEY").StringVar(&key)
	sign.Flag("force", "Do not warn about no ed25519 key").BoolVar(&force)

	verify := app.Command("verify", "Verifies a signature made using pack").Action(verifyAction)
	verify.Arg("source", "The signed artifact to validate").ExistingFileVar(&managed)
	verify.Arg("key", "The ed25519 public key to verify with").StringVar(&key)

	fisk.MustParse(app.Parse(os.Args[1:]))
}

func verifyAction(_ *fisk.ParseContext) error {
	data, err := os.ReadFile(managed)
	if err != nil {
		return err
	}

	var spec machines.Specification
	err = json.Unmarshal(data, &spec)
	if err != nil {
		return err
	}

	var pk ed25519.PublicKey
	if iu.FileExist(key) {
		kdat, err := os.ReadFile(key)
		if err != nil {
			return err
		}
		pk, err = hex.DecodeString(string(kdat))
		if err != nil {
			return fmt.Errorf("invalid public key data")
		}
	} else {
		pk, err = hex.DecodeString(key)
		if err != nil {
			return err
		}
	}

	sig, err := hex.DecodeString(spec.Signature)
	if err != nil {
		return fmt.Errorf("invalid signature: %v", err)
	}

	ok, err := iu.Ed25519Verify(pk, spec.Machines, sig)
	if err != nil {
		return err
	}

	if !ok {
		fmt.Printf("Data %s does not have a valid signature for key %s", managed, hex.EncodeToString(pk))
		os.Exit(1)
	}

	fmt.Printf("Data file %s is signed using %s\n", managed, hex.EncodeToString(pk))

	return nil
}

func keysAction(_ *fisk.ParseContext) error {
	pub, pri, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}

	fmt.Printf(" Public Key: %s\n", hex.EncodeToString(pub))
	fmt.Printf("Private Key: %s\n", hex.EncodeToString(pri))

	return nil
}

func signAction(_ *fisk.ParseContext) error {
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
