// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/server/agents"
)

var metadata = &agents.Metadata{
	Name:        "scout",
	Description: "Choria Scout Agent Management API",
	Author:      "R.I.Pienaar <rip@devco.net>",
	Version:     build.Version,
	License:     "Apache-2.0",
	Timeout:     20,
	URL:         "http://choria.io",
}
