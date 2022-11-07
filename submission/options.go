// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package submission

type spoolOpts struct {
	maxSize   int
	spoolDir  string
	seedFile  string
	tokenFile string
}

type Option func(o *spoolOpts)

// WithSpoolDirectory sets the path to the directory for the Directory store
func WithSpoolDirectory(d string) Option {
	return func(o *spoolOpts) {
		o.spoolDir = d
	}
}

// WithMaxSpoolEntries sets the maximum amount of entries allow in the spool, new entries will be rejected
func WithMaxSpoolEntries(max int) Option {
	return func(o *spoolOpts) {
		o.maxSize = max
	}
}

// WithSeedFile sets the ed25519 seed to use which will enable signed messages
func WithSeedFile(seed string) Option {
	return func(o *spoolOpts) {
		o.seedFile = seed
	}
}

// WithTokenFile sets the JWT file to use, when set will set it as a header in signed messages
func WithTokenFile(token string) Option {
	return func(o *spoolOpts) {
		o.tokenFile = token
	}
}
