// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package puppetdb

import (
	"sync"
	"time"

	"github.com/choria-io/go-choria/protocol"
)

type dOpts struct {
	filter     *protocol.Filter
	collective string
	discovered []string
	mu         *sync.Mutex
	timeout    time.Duration
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

// Timeout sets the discovery timeout, else the configured default is used
func Timeout(t time.Duration) DiscoverOption {
	return func(o *dOpts) {
		o.timeout = t
	}
}
