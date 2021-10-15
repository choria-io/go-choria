// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

//go:build !windows
// +build !windows

package choria

func (fw *Framework) openLogfile() error {
	return fw.commonLogOpener()
}
