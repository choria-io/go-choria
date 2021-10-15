// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package lifecycle

import (
	"time"
)

// Event is event that can be published to the network
type Event interface {
	Protocol() string
	Target() (string, error)
	String() string
	Type() Type
	TypeString() string
	SetIdentity(string)
	Component() string
	Identity() string
	ID() string
	Format() Format
	SetFormat(Format)
	TimeStamp() time.Time
}
