// Copyright (c) 2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"
	"time"

	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/providers/agent/mcorpc"
	"github.com/choria-io/go-choria/providers/execution"
)

type ListRequest struct {
	Action    string `json:"action"`
	Agent     string `json:"agent"`
	Before    int64  `json:"before"`
	Caller    string `json:"caller"`
	Command   string `json:"command"`
	Completed bool   `json:"completed"`
	Identity  string `json:"identity"`
	RequestID string `json:"requestid"`
	Running   bool   `json:"running"`
	Since     int64  `json:"since"`
}

type ListMatched struct {
	Action        string    `json:"action"`
	Agent         string    `json:"agent"`
	Command       string    `json:"command"`
	ID            string    `json:"id"`
	Identity      string    `json:"identity"`
	RequestID     string    `json:"requestid"`
	StartTime     time.Time `json:"start"`
	TerminateTime time.Time `json:"terminate"`
}
type ListResponse struct {
	Jobs []ListMatched `json:"jobs"`
}

func listAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn inter.ConnectorInfo) {
	spool := agent.Config.Choria.ExecutorSpool
	if spool == "" {
		abort(reply, "Executor spool is not configured")
		return
	}

	args := &ListRequest{}
	if !mcorpc.ParseRequestData(args, req, reply) {
		return
	}

	resp := &ListResponse{}

	matched, err := execution.List(spool, &execution.ListQuery{
		Action:    args.Action,
		Agent:     args.Agent,
		Before:    time.Unix(args.Before, 0),
		Caller:    args.Caller,
		Command:   args.Command,
		Completed: args.Completed,
		Identity:  args.Identity,
		RequestID: args.RequestID,
		Running:   args.Running,
		Since:     time.Unix(args.Since, 0),
	})
	if err != nil {
		abort(reply, "Could not list jobs: %v", err)
		return
	}

	for _, job := range matched {
		resp.Jobs = append(resp.Jobs, ListMatched{
			Action:        job.Action,
			Agent:         job.Agent,
			Command:       job.Command,
			ID:            job.ID,
			Identity:      job.Identity,
			RequestID:     job.RequestID,
			StartTime:     job.StartTime,
			TerminateTime: job.TerminateTime,
		})
	}

	reply.Data = resp
}
