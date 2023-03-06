// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package provision

import (
	"context"
	"fmt"
	"time"

	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/lifecycle"
	"github.com/choria-io/go-choria/providers/agent/mcorpc"
	"github.com/choria-io/go-updater"
)

type ReleaseUpdateRequest struct {
	Token      string `json:"token"`
	Repository string `json:"repository"`
	Version    string `json:"version"`
}

var updaterf func(...updater.Option) error

func releaseUpdateAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn inter.ConnectorInfo) {
	if !agent.Choria.ProvisionMode() {
		abort("Cannot update a server that is not in provisioning mode", reply)
		return
	}

	if !agent.Choria.Configuration().Choria.ProvisionAllowUpdate {
		abort("Version updates is not enabled in the configuration", reply)
		return
	}

	mu.Lock()
	defer mu.Unlock()

	args := ReleaseUpdateRequest{}
	if !mcorpc.ParseRequestData(&args, req, reply) {
		return
	}

	if !checkToken(args.Token, reply) {
		return
	}

	opts := []updater.Option{
		updater.Version(args.Version),
		updater.SourceRepo(args.Repository),
		updater.Logger(agent.Log),
	}

	err := versionUpdater()(opts...)
	if err != nil {
		if err := updater.RollbackError(err); err != nil {
			abort(fmt.Sprintf("Update to version %s failed, rollback also failed, server in broken state: %s", args.Version, err), reply)
			return
		}

		abort(fmt.Sprintf("Update to version %s failed, release rolled back: %s", args.Version, err), reply)
		return
	}

	err = agent.ServerInfoSource.NewEvent(lifecycle.Upgraded, lifecycle.NewVersion(args.Version))
	if err != nil {
		agent.Log.Errorf("Could not publish upgraded event: %s", err)
	}

	reply.Data = Reply{"Restarting Choria Server after 2s"}
	agent.Log.Warnf("Restarting server via request %s from %s (%s) with splay 2s", req.RequestID, req.CallerID, req.SenderID)

	go restartCb(2*time.Second, agent.ServerInfoSource, agent.Log)
}

func versionUpdater() func(...updater.Option) error {
	if updaterf != nil {
		return updaterf
	}

	return updater.Apply
}
