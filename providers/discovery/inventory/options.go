// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package inventory

import (
	"github.com/choria-io/go-choria/protocol"
)

type dOpts struct {
	filter     *protocol.Filter
	collective string
	do         map[string]string
	source     string
	noValidate bool
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

// DiscoveryOptions sets the key value pairs that make user supplied discovery options.
//
// Supported options:
//
//   file - set the file to read
func DiscoveryOptions(opt map[string]string) DiscoverOption {
	return func(o *dOpts) {
		o.do = opt
	}
}

// File sets the file to read nodes from
func File(f string) DiscoverOption {
	return func(o *dOpts) {
		o.source = f
	}
}
