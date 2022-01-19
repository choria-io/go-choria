// Copyright (c) 2020-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"sync"

	"github.com/choria-io/go-choria/server"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/windows/svc"
)

type winServiceWrapper struct {
	instance *server.Instance
	cmd      *serverRunCommand
}

func (w *winServiceWrapper) Execute(args []string, changes <-chan svc.ChangeRequest, status chan<- svc.Status) (bool, uint32) {
	status <- svc.Status{State: svc.StartPending}

	var err error

	w.instance, err = w.cmd.prepareInstance()
	if err != nil {
		log.Errorf("failed to create instance of the server: %q", err)
		return false, 1
	}

	wg.Add(1)
	go w.instance.Run(ctx, wg)

	status <- svc.Status{
		State:   svc.Running,
		Accepts: svc.AcceptStop | svc.AcceptShutdown,
	}

loop:
	for change := range changes {
		switch change.Cmd {
		case svc.Interrogate:
			status <- change.CurrentStatus

		case svc.Stop, svc.Shutdown:
			cancel()

			break loop
		default:
			log.Warnf("Unexpected service control sequence received: %q", change.Cmd)
		}
	}

	status <- svc.Status{State: svc.StopPending}

	return false, 0
}

func (r *serverRunCommand) platformRun(wg *sync.WaitGroup) (err error) {
	interactive, err := svc.IsWindowsService()
	if err != nil {
		return err
	}

	if interactive {
		instance, err := r.prepareInstance()
		if err != nil {
			return err
		}

		wg.Add(1)
		if r.serviceHost {
			return instance.RunServiceHost(ctx, wg)
		}

		return instance.Run(ctx, wg)
	}

	return svc.Run("choria-server", &winServiceWrapper{cmd: r})
}
