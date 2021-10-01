// Copyright (c) 2017-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package nagioswatcher

func (w *Watcher) watchUsingGoss() (state State, output string, err error) {
	return UNKNOWN, "UNKNOWN: goss is not supported on windows", nil
}
