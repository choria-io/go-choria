package agent

import (
	"fmt"

	"github.com/choria-io/go-choria/aagent"
	"github.com/choria-io/go-choria/providers/agent/mcorpc"
	"github.com/choria-io/go-choria/server"
	"github.com/choria-io/go-choria/server/agents"
)

// only activate when not in provisioning mode
func activationCheck(mgr server.AgentManager) func() bool {
	return func() bool {
		return !mgr.Choria().ProvisionMode()
	}
}

func abort(msg string, reply *mcorpc.Reply) {
	reply.Statuscode = mcorpc.Aborted
	reply.Statusmsg = msg
}

func stringInStrings(s string, ss []string) bool {
	for _, i := range ss {
		if s == i {
			return true
		}
	}

	return false
}

func transitionSelectedChecks(list []string, transition string, si agents.ServerInfoSource, reply *mcorpc.Reply) (forced []string, failed []string, skipped []string) {
	forced = []string{}
	failed = []string{}
	skipped = []string{}

	states, err := si.MachinesStatus()
	if err != nil {
		abort(fmt.Sprintf("Failed to retrieve states: %s", err), reply)
		return forced, failed, skipped
	}

	// user asked for none, implies all
	all := len(list) == 0

	statemap := make(map[string]aagent.MachineState)
	for _, m := range states {
		if !m.Scout {
			continue
		}

		statemap[m.Name] = m
		if all {
			list = append(list, m.Name)
		}
	}

	for _, n := range list {
		m, ok := statemap[n]
		if !ok {
			log.Warnf("Cannot transition %s using %s, unknown check", n, transition)
			failed = append(failed, n)
			continue
		}

		if !stringInStrings(transition, m.AvailableTransitions) {
			log.Warnf("Could not transition check %s using %s: not a valid transition", m.Name, transition)
			skipped = append(skipped, m.Name)
			continue
		}

		err = si.MachineTransition(m.Name, "", "", "", transition)
		if err != nil {
			log.Errorf("Failed to transition check %s to %s: %s", m.Name, transition, err)
			failed = append(failed, m.Name)
			continue
		}

		forced = append(forced, m.Name)
	}

	if !all && (len(forced)+len(skipped) != len(list)) {
		abort(fmt.Sprintf("Some checks could not be transitioned to %s", transition), reply)
	}

	return forced, failed, skipped
}
