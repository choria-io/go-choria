// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package nagioswatcher

import (
	"strconv"
	"time"
)

func (w *Watcher) builtinHeartbeat() (state State, output string, err error) {
	return OK, strconv.Itoa(int(time.Now().Unix())), nil
}
