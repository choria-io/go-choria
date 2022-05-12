// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package inter

import (
	"gopkg.in/alecthomas/kingpin.v2"
)

// FlagApp is a kingpin command or app
type FlagApp interface {
	Flag(name, help string) *kingpin.FlagClause
	Command(name, help string) *kingpin.CmdClause
}
