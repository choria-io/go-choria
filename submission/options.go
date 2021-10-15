// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package submission

type spoolOpts struct {
	maxSize  int
	spoolDir string
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
