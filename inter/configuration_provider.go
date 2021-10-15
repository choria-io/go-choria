// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package inter

import (
	"github.com/choria-io/go-choria/config"
)

// ConfigurationProvider provides runtime Choria configuration
type ConfigurationProvider interface {
	Configuration() *config.Config
}
