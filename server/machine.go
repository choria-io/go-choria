package server

import (
	"context"
	"sync"

	"github.com/choria-io/go-choria/aagent"
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

	srv.machines, err = aagent.New(srv.cfg.Choria.MachineSourceDir, srv)
	if err != nil {
		return err
	}

	return srv.machines.ManageMachines(ctx, wg)
}
