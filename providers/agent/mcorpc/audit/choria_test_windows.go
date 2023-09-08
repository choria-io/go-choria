// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0
//go:build windows
// +build windows

package audit

import (
	"os"
)

func checkFileGid(stat os.FileInfo, group string) {}
