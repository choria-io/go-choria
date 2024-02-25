// Copyright (c) 2024, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

//go:build !windows

package execwatcher

import (
	"os/exec"
	"syscall"
)

func disown(cmd *exec.Cmd) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}

	return nil
}
