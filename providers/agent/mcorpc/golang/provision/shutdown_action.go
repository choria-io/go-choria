// Copyright (c) 2022-2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package provision

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/providers/agent/mcorpc"
)

type ShutdownRequest struct {
	Token string `json:"token"`
}

func shutdownAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn inter.ConnectorInfo) {
	mu.Lock()
	defer mu.Unlock()

	if !agent.Choria.ProvisionMode() && build.ProvisionToken == "" {
		abort("Cannot shutdown a server that is not in provisioning mode or with no token set", reply)
		return
	}

	args := &ShutdownRequest{}
	if !mcorpc.ParseRequestData(args, req, reply) {
		return
	}

	if !checkToken(args.Token, reply) {
		return
	}

	splay := time.Duration(rand.IntN(10)) * time.Second
	agent.Log.Warnf("Shutting server down via request %s from %s (%s) with splay %v", req.RequestID, req.CallerID, req.SenderID, splay)

	go shutdownCb(splay, agent.ServerInfoSource, agent.Log)

	reply.Data = Reply{fmt.Sprintf("Shutting Choria Server down after %v", splay)}
}
