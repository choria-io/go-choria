// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package v2

type federationTransportHeader struct {
	RequestID string   `json:"req,omitempty"`
	ReplyTo   string   `json:"reply-to,omitempty"`
	Targets   []string `json:"target,omitempty"`
}
