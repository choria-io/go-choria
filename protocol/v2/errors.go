// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package v2

import (
	"fmt"
)

var (
	ErrIncorrectProtocol = fmt.Errorf("version 2 protocol requires a ed25519+jwt based security system")
)
