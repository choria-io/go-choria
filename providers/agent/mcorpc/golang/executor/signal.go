// Copyright (c) 2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"
	"syscall"

	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/providers/agent/mcorpc"
	"github.com/choria-io/go-choria/providers/execution"
)

type SignalRequest struct {
	JobID  string `json:"id"`
	Signal int    `json:"signal"`
}

type SignalResponse struct {
	Pid     int  `json:"pid"`
	Running bool `json:"running"`
}

func signalAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn inter.ConnectorInfo) {
	spool := agent.Config.Choria.ExecutorSpool
	if spool == "" {
		abort(reply, "Executor spool is not configured")
		return
	}

	args := &SignalRequest{}

	if !mcorpc.ParseRequestData(args, req, reply) {
		return
	}

	if args.JobID == "" {
		abort(reply, "ID is required")
	}

	if args.Signal < 0 {
		abort(reply, "Signal is required")
	}

	resp := &SignalResponse{}

	p, err := execution.Load(spool, args.JobID)
	if err != nil {
		abort(reply, "Could not load job: %v", err.Error())
		return
	}

	if proxyAuthorize(p, req, agent) {
		agent.Log.Warnf("Denying %s access to process created by %s#%s based on authorization policy for request %s", req.CallerID, p.Agent, p.Action, req.RequestID)
		abort(reply, "You are not authorized to call this %s#%s", p.Agent, p.Action)
		return
	}

	resp.Running = p.IsRunning()
	if !resp.Running {
		abort(reply, "Job %s is not running", args.JobID)
		return
	}

	resp.Pid, err = p.ParsePid()
	if err != nil {
		abort(reply, "Could not parse pid file: %v", err.Error())
		return
	}

	err = p.Signal(syscall.Signal(args.Signal))
	if err != nil {
		abort(reply, "Could not send signal: %v", err.Error())
		return
	}

	resp.Running = p.IsRunning()

	reply.Data = resp
}
