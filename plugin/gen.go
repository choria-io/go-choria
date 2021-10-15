// Copyright (c) 2018-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

//go:build ignore
// +build ignore

package main

import (
	"os"

	"github.com/choria-io/go-choria/plugin"
)

func main() {
	if !plugin.Generate() {
		os.Exit(1)
	}
}
