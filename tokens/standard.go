// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package tokens

import (
	"github.com/golang-jwt/jwt/v4"
)

type StandardClaims struct {
	Purpose Purpose `json:"purpose"`

	jwt.RegisteredClaims
}
