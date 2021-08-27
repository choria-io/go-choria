package agent

import (
	"context"

	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/providers/agent/mcorpc"
)

type ResumeRequest struct {
	Checks []string `json:"checks"`
}

type ResumeReply struct {
	TransitionedChecks []string `json:"transitioned"`
	FailedChecks       []string `json:"failed"`
	SkippedChecks      []string `json:"skipped"`
}

func resumeAction(_ context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, _ inter.ConnectorInfo) {
	resp := &ResumeReply{[]string{}, []string{}, []string{}}
	reply.Data = resp

	args := &ResumeRequest{}
	if !mcorpc.ParseRequestData(args, req, reply) {
		return
	}

	resp.TransitionedChecks, resp.FailedChecks, resp.SkippedChecks = transitionSelectedChecks(args.Checks, resumeTransition, agent.ServerInfoSource, reply)
}
