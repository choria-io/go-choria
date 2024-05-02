// Copyright (c) 2021-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package external

import (
	"time"

	"github.com/choria-io/go-choria/protocol"
)

type dOpts struct {
	filter      *protocol.Filter
	collective  string
	federations []string
	timeout     time.Duration
	command     string
	do          map[string]string
}

// DiscoverOption configures the broadcast discovery method
type DiscoverOption func(o *dOpts)

// Filter sets the filter to use for the discovery, else a blank one is used
func Filter(f *protocol.Filter) DiscoverOption {
	return func(o *dOpts) {
		o.filter = f
	}
}

// Collective sets the collective to discover in, else main collective is used
func Collective(c string) DiscoverOption {
	return func(o *dOpts) {
		o.collective = c
	}
}

// Federations sets the list of federated collectives to discover in
func Federations(f []string) DiscoverOption {
	return func(o *dOpts) {
		o.federations = f
	}
}

// Timeout sets the discovery timeout, else the configured default is used
func Timeout(t time.Duration) DiscoverOption {
	return func(o *dOpts) {
		o.timeout = t
	}
}

func Command(c string) DiscoverOption {
	return func(o *dOpts) {
		o.command = c
	}
}

// DiscoveryOptions sets the key value pairs that make user supplied discovery options.
//
// Supported options:
//
//	command - The command to execute instead of configured default
//
// All options will be passed to the external command in the request, so other
// command specific options is supported and will be ignored by this code
func DiscoveryOptions(opt map[string]string) DiscoverOption {
	return func(o *dOpts) {
		o.do = opt
	}
}
