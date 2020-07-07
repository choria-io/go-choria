package agent

import (
	"context"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/providers/agent/mcorpc"
)

type MaintenanceRequest struct {
	Checks []string `json:"checks"`
}

type MaintenanceReply struct {
	TransitionedChecks []string `json:"transitioned"`
	FailedChecks       []string `json:"failed"`
	SkippedChecks      []string `json:"skipped"`
}

func maintenanceAction(_ context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, _ choria.ConnectorInfo) {
	resp := &MaintenanceReply{[]string{}, []string{}, []string{}}
	reply.Data = resp

	args := &MaintenanceRequest{}
	if !mcorpc.ParseRequestData(args, req, reply) {
		return
	}

	resp.TransitionedChecks, resp.FailedChecks, resp.SkippedChecks = transitionSelectedChecks(args.Checks, maintenanceTransition, agent.ServerInfoSource, reply)
}
