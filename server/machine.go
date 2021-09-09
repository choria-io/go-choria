package server

import (
	"context"
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
