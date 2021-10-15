// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

// generated code; DO NOT EDIT

package aaa_signerclient

import (
	_ "embed"

	"github.com/choria-io/go-choria/providers/agent/mcorpc/ddl/agent"
)

//go:embed ddl.json
var rawDDL []byte

// DDLBytes is the raw JSON encoded DDL file for the agent
func DDLBytes() ([]byte, error) {
	return rawDDL, nil
}

// DDL is a parsed and loaded DDL for the agent
func DDL() (*agent.DDL, error) {
	return agent.NewFromBytes(rawDDL)
}
