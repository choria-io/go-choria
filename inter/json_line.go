// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package inter

import (
	"encoding/json"
)

// JsonLineOutput is a line format for json lines output from the cli
type JsonLineOutput struct {
	Kind             string          `json:"k"`
	ProtocolReply    json.RawMessage `json:"pr,omitempty"`
	RPCReply         json.RawMessage `json:"rr,omitempty"`
	Aggregates       json.RawMessage `json:"agg,omitempty"`
	Stats            json.RawMessage `json:"stat,omitempty"`
	Discovered       int             `json:"count,omitempty"`
	DiscoverySeconds float64         `json:"dt,omitempty"`
	DiscoveryMethod  string          `json:"dm,omitempty"`
	Error            string          `json:"err,omitempty"`
}

const (
	JsonLineErrorKind      = "error"
	JsonLineResultKind     = "result"
	JsonLineSummariesKind  = "summaries"
	JsonLineStatsKind      = "stats"
	JsonLineDiscoveredKind = "discovery"
)
