// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"github.com/choria-io/go-choria/logger"
	"github.com/choria-io/go-choria/plugin"
	"github.com/choria-io/go-choria/providers/agent/mcorpc"
	"github.com/choria-io/go-choria/server"
	"github.com/choria-io/go-choria/server/agents"
)

var (
	log logger.Logrus
)

const (
	forceTransition       = "FORCE_CHECK"
	maintenanceTransition = "MAINTENANCE"
	resumeTransition      = "RESUME"
)

func New(mgr server.AgentManager) (agents.Agent, error) {
	log = mgr.Logger()

	agent := mcorpc.New("scout", metadata, mgr.Choria(), mgr.Logger())

	agent.SetActivationChecker(activationCheck(mgr))

	agent.MustRegisterAction("checks", checksAction)
	agent.MustRegisterAction("trigger", triggerAction)
	agent.MustRegisterAction("maintenance", maintenanceAction)
	agent.MustRegisterAction("resume", resumeAction)
	agent.MustRegisterAction("goss_validate", gossValidateAction)

	// TODO: info action showing machine info - facts and inventory like response

	return agent, nil
}

func ChoriaPlugin() plugin.Pluggable {
	return mcorpc.NewChoriaAgentPlugin(metadata, New)
}
