// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"github.com/choria-io/go-choria/internal/fs"
)

// CachedDDL is a parsed and loaded DDL for the agent a
// TODO: remove
func CachedDDL(a string) (*DDL, error) {
	ddlj, err := fs.FS.ReadFile("ddl/cache/agent/" + a + ".json")
	if err != nil {
		return nil, err
	}

	return NewFromBytes(ddlj)
}
