// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package inter

import (
	"github.com/choria-io/fisk"
)

// FlagApp is a fisk command or app
type FlagApp interface {
	Flag(name, help string) *fisk.FlagClause
	Command(name, help string) *fisk.CmdClause
}
