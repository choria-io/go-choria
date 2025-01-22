// Copyright (c) 2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"
	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/providers/agent/mcorpc"
	"github.com/choria-io/go-choria/providers/execution"
	"time"
)

type StatusRequest struct {
	JobID string `json:"id"`
}

type StatusResponse struct {
	Started       bool      `json:"started"`
	StartTime     time.Time `json:"start_time"`
	TerminateTime time.Time `json:"terminate_time"`
	ExitCode      int       `json:"exit_code"`
	Running       bool      `json:"running"`
	Agent         string    `json:"agent"`
	Action        string    `json:"action"`
	RequestID     string    `json:"requestid"`
	Caller        string    `json:"caller"`
	Pid           int       `json:"pid"`
	StdoutBytes   int       `json:"stdout_bytes"`
	StderrBytes   int       `json:"stderr_bytes"`
}

func statusAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn inter.ConnectorInfo) {
	spool := agent.Config.Choria.ExecutorSpool

	if spool == "" {
		abort(reply, "Executor spool is not configured")
		return
	}

	args := &StatusRequest{}
	if !mcorpc.ParseRequestData(args, req, reply) {
		return
	}

	p, err := execution.Load(spool, args.JobID)
	if err != nil {
		reply.Statuscode = mcorpc.Aborted
		reply.Statusmsg = err.Error()
		abort(reply, "Could not load job: %v", err)
		return
	}

	resp := &StatusResponse{
		Running:       p.IsRunning(),
		StartTime:     p.StartTime,
		TerminateTime: p.TerminateTime,
		Agent:         p.Agent,
		Action:        p.Action,
		RequestID:     p.RequestID,
		Caller:        p.Caller,
		Pid:           -1,
		ExitCode:      -1,
	}

	resp.Started, err = p.HasStarted()
	if err != nil {
		abort(reply, "Could not check if job is started: %v", err)
		return
	}

	if resp.Started {
		resp.Pid, err = p.ParsePid()
		if err != nil {
			abort(reply, "Could not parse pid: %v", err)
			return
		}
	}

	if !resp.Running && resp.Started {
		resp.ExitCode, err = p.ParseExitCode()
		if err != nil {
			abort(reply, "Could not parse exit code: %v", err)
			return
		}

		b, _ := p.Stderr()
		resp.StderrBytes = len(b)

		b, _ = p.Stdout()
		resp.StdoutBytes = len(b)
	}

	reply.Data = resp
}
