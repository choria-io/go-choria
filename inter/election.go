// Copyright (c) 2017-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package inter

import "context"

// Election is a NATS Key-Value Store based Leader Election system
type Election interface {
	// Start starts the election, interrupted by context. Blocks until stopped.
	Start(ctx context.Context) error
	// Stop stops the election
	Stop()
	// IsLeader determines if we are currently the leader
	IsLeader() bool
}
