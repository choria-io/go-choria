// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package fs

import (
	"embed"
)

//go:embed ddl
//go:embed client
//go:embed plugin
//go:embed misc
//go:embed completion
//go:embed cheats
var FS embed.FS
