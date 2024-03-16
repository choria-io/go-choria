// Copyright (c) 2019-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"github.com/choria-io/go-choria/providers/autoagents/plugins"
	"os"
	"sync"

	"github.com/choria-io/go-choria/aagent"
	"github.com/choria-io/go-choria/internal/util"
)

// StartMachine starts the choria machine instances
func (srv *Instance) StartMachine(ctx context.Context, wg *sync.WaitGroup) (err error) {
	if srv.fw.ProvisionMode() {
		return
	}

	if srv.cfg.Choria.MachineSourceDir == "" {
		srv.log.Info("Choria Autonomous Agent source directory not configured, skipping initialization")
		return nil
	}

	if !util.FileIsDir(srv.cfg.Choria.MachineSourceDir) {
		srv.log.Warnf("Choria Autonomous Agent source directory configured as %s but it does not exist, creating empty directory", srv.cfg.Choria.MachineSourceDir)
		err := os.MkdirAll(srv.cfg.Choria.MachineSourceDir, 0700)
		if err != nil {
			return err
		}
	}

	srv.machines, err = aagent.New(srv.cfg.Choria.MachineSourceDir, srv)
	if err != nil {
		return err
	}

	return srv.machines.ManageMachines(ctx, wg)
}

// StartInternalMachines starts built-in autonomous agents
func (srv *Instance) StartInternalMachines(ctx context.Context) (err error) {
	if !srv.cfg.Choria.AutonomousAgentsDownload {
		return
	}

	srv.log.Info("Starting built-in Autonomous Agent Plugin Manager")

	m, err := plugins.ChoriaPlugin(srv.cfg)
	if err != nil {
		return err
	}

	return srv.machines.LoadPlugin(ctx, m)
}
