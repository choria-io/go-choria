// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"github.com/choria-io/go-choria/protocol/stats"
)

const promVersion = "1"

var (
	protocolErrorCtr = stats.ProtocolErrorCtr.WithLabelValues(promVersion)
	invalidCtr       = stats.InvalidCtr.WithLabelValues(promVersion)
	validCtr         = stats.ValidCtr.WithLabelValues(promVersion)
	badJsonCtr       = stats.BadJsonCtr.WithLabelValues(promVersion)
)
