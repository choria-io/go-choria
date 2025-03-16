// Copyright (c) 2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/providers/agent/mcorpc"
	"github.com/choria-io/go-choria/server"
	"github.com/choria-io/go-choria/server/agents"
)

var metadata = &agents.Metadata{
	Name:        "executor",
	Description: "Choria Process Executor Management",
	Author:      "R.I.Pienaar <rip@devco.net>",
	Version:     build.Version,
	License:     build.License,
	Timeout:     20,
	URL:         "https://choria.io",
}

func New(mgr server.AgentManager) (*mcorpc.Agent, error) {
	log := mgr.Logger()
	agent := mcorpc.New("executor", metadata, mgr.Choria(), log)

	agent.SetActivationChecker(func() bool {
		return mgr.Choria().Configuration().Choria.ExecutorEnabled
	})

	agent.MustRegisterAction("status", statusAction)
	agent.MustRegisterAction("signal", signalAction)
	agent.MustRegisterAction("list", listAction)

	return agent, nil
}
