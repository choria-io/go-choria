// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package lifecycle

import (
	"time"

	"github.com/choria-io/go-choria/internal/util"
)

var mockTime int64
var mockID string

func timeStamp() int64 {
	if mockTime != 0 {
		return mockTime
	}

	return time.Now().UTC().Unix()
}

func eventID() string {
	if mockID != "" {
		return mockID
	}

	return util.UniqueID()
}
