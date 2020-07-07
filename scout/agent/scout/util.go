package agent

import (
	"fmt"

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
	states, err := si.MachinesStatus()
	if err != nil {
		abort(fmt.Sprintf("Failed to retrieve states: %s", err), reply)
		return forced, failed, skipped
	}

	if len(list) == 0 {
		for _, m := range states {
			list = append(list, m.Name)
		}
	}

	for _, m := range states {
		if !stringInStrings(m.Name, list) {
			continue
		}

		if !stringInStrings(transition, m.AvailableTransitions) {
			log.Errorf("Could not transition check %s using %s: not a valid transition", m.Name, transition)
			skipped = append(skipped, m.Name)
			continue
		}

		err = si.MachineTransition(m.Name, "", "", "", transition)
		if err != nil {
			log.Errorf("Could not transition check %s to %s: %s", m.Name, transition, err)
			failed = append(failed, m.Name)
			continue
		}

		forced = append(forced, m.Name)
	}

	if len(failed) > 0 {
		abort("Some checked could not be forced", reply)
	}

	return forced, failed, skipped
}
