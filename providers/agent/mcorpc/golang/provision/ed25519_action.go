// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package provision

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/providers/agent/mcorpc"
)

type ED25519Request struct {
	Token string `json:"token"`
	Nonce string `json:"nonce"`
}

type ED25519Reply struct {
	PublicKey string `json:"public_key"`
	Directory string `json:"directory"`
	Signature string `json:"signature"`
}

func ed25519Action(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn inter.ConnectorInfo) {
	mu.Lock()
	defer mu.Unlock()

	if !agent.Choria.ProvisionMode() {
		abort("Cannot reconfigure a server that is not in provisioning mode", reply)
		return
	}

	if agent.Config.ConfigFile == "" {
		abort("Cannot determine where to store secure data, no configure file given", reply)
		return
	}

	args := ED25519Request{}
	if !mcorpc.ParseRequestData(&args, req, reply) {
		return
	}

	if !checkToken(args.Token, reply) {
		return
	}

	secureDir := filepath.Join(filepath.Dir(agent.Config.ConfigFile), "secure")
	keyFile := filepath.Join(secureDir, "server.key")

	agent.Log.Infof("Creating a new ED25519 key pair in %s", secureDir)

	err := os.MkdirAll(secureDir, 0700)
	if err != nil {
		abort(fmt.Sprintf("Could not create secure directory %s: %s", secureDir, err), reply)
		return
	}

	pubK, priK, err := choria.Ed25519KeyPair()
	if err != nil {
		abort(fmt.Sprintf("Could not create keypair: %s", err), reply)
		return
	}

	err = os.WriteFile(keyFile, []byte(hex.EncodeToString(priK.Seed())), 0600)
	if err != nil {
		abort(fmt.Sprintf("Could not write key %s: %s", keyFile, err), reply)
		return
	}

	sig, err := choria.Ed25519Sign(priK, []byte(args.Nonce))
	if err != nil {
		abort(fmt.Sprintf("Could not sign the nonce: %s", err), reply)
		return
	}

	reply.Data = &ED25519Reply{
		PublicKey: hex.EncodeToString(pubK),
		Directory: secureDir,
		Signature: string(sig),
	}
}
