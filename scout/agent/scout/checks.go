package agent

import (
	"context"
	"fmt"

	"github.com/choria-io/go-choria/aagent/watchers/nagioswatcher"
	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/providers/agent/mcorpc"
)

type ChecksRequest struct{}

type ChecksResponse struct {
	Checks []*CheckState `json:"checks"`
}

type CheckState struct {
	Name    string                           `json:"name"`
	State   string                           `json:"state"`
	Version string                           `json:"version"`
	Started int64                            `json:"start_time"`
	Status  *nagioswatcher.StateNotification `json:"status"`
}

func checksAction(_ context.Context, _ *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, _ inter.ConnectorInfo) {
	resp := &ChecksResponse{Checks: []*CheckState{}}
	reply.Data = resp

	states, err := agent.ServerInfoSource.MachinesStatus()
	if err != nil {
		abort(fmt.Sprintf("Failed to retrieve states: %s", err), reply)
		return
	}

	for _, m := range states {
		if !m.Scout {
			continue
		}

		state := &CheckState{
			Name:    m.Name,
			State:   m.State,
			Version: m.Version,
			Started: m.StartTimeUTC,
		}

		ss, ok := m.ScoutState.(*nagioswatcher.StateNotification)
		if ok {
			state.Status = ss
		}

		resp.Checks = append(resp.Checks, state)
	}
}
