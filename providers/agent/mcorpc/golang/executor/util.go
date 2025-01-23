// Copyright (c) 2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"time"

	"github.com/choria-io/go-choria/providers/agent/mcorpc"
	"github.com/choria-io/go-choria/providers/execution"
)

func abort(reply *mcorpc.Reply, format string, a ...any) {
	reply.Statuscode = mcorpc.Aborted
	reply.Statusmsg = fmt.Sprintf(format, a...)
}

func proxyAuthorize(p *execution.Process, req *mcorpc.Request, agent *mcorpc.Agent) bool {
	processRequest := &mcorpc.Request{
		Agent:            p.Agent,
		Action:           p.Action,
		RequestID:        p.RequestID,
		SenderID:         req.SenderID,
		CallerID:         req.CallerID,
		Collective:       req.Collective,
		TTL:              req.TTL,
		Time:             time.Time{},
		Filter:           req.Filter,
		CallerPublicData: req.CallerPublicData,
		SignerPublicData: req.SignerPublicData,
	}

	return mcorpc.AuthorizeRequest(agent.Choria, processRequest, agent.Config, agent.ServerInfoSource, agent.Log)
}
