// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package v2

import (
	"errors"

	"github.com/choria-io/go-choria/protocol/stats"
)

const promVersion = "2"

var (
	ErrInvalidJSON   = errors.New("supplied JSON document does not pass schema validation")
	protocolErrorCtr = stats.ProtocolErrorCtr.WithLabelValues(promVersion)
	invalidCtr       = stats.InvalidCtr.WithLabelValues(promVersion)
	// validCtr         = stats.ValidCtr.WithLabelValues(promVersion)
	badJsonCtr = stats.BadJsonCtr.WithLabelValues(promVersion)
)
