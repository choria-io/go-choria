// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"

	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/providers/agent/mcorpc"
)

type TriggerRequest struct {
	Checks []string `json:"checks"`
}

type TriggerReply struct {
	TransitionedChecks []string `json:"transitioned"`
	FailedChecks       []string `json:"failed"`
	SkippedChecks      []string `json:"skipped"`
}

func triggerAction(_ context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, _ inter.ConnectorInfo) {
	resp := &TriggerReply{[]string{}, []string{}, []string{}}
	reply.Data = resp

	args := &TriggerRequest{}
	if !mcorpc.ParseRequestData(args, req, reply) {
		return
	}

	resp.TransitionedChecks, resp.FailedChecks, resp.SkippedChecks = transitionSelectedChecks(args.Checks, forceTransition, agent.ServerInfoSource, reply)
}
