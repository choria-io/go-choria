// Copyright (c) 2021-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

//go:build !windows
// +build !windows

package cmd

import (
	"sync"
)

func (r *serverRunCommand) platformRun(wg *sync.WaitGroup) (err error) {
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
