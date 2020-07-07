package agent

import (
	"context"
	"fmt"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/providers/agent/mcorpc"
)

type ChecksRequest struct{}

type CheckResponse struct {
	Checks []*CheckState `json:"checks"`
}

type CheckState struct {
	Name    string
	State   string
	Version string
	Started int64
}

func checksAction(_ context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, _ choria.ConnectorInfo) {
	resp := &CheckResponse{Checks: []*CheckState{}}
	reply.Data = resp

	states, err := agent.ServerInfoSource.MachinesStatus()
	if err != nil {
		abort(fmt.Sprintf("Failed to retrieve states: %s", err), reply)
		return
	}

	for _, m := range states {
		check := &CheckState{
			Name:    m.Name,
			State:   m.State,
			Version: m.Version,
			Started: m.StartTimeUTC,
		}

		resp.Checks = append(resp.Checks, check)
	}
}
