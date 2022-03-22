// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package testutil

import (
	"crypto/ed25519"
	"crypto/rsa"
	"encoding/hex"
	"time"

	"github.com/golang-jwt/jwt"
)

func CreateSignedClientJWT(pk *rsa.PrivateKey, claims map[string]interface{}) (string, error) {
	c := map[string]interface{}{
		"exp":      time.Now().UTC().Add(time.Hour).Unix(),
		"nbf":      time.Now().UTC().Add(-1 * time.Minute).Unix(),
		"iat":      time.Now().UTC().Unix(),
		"iss":      "Ginkgo",
		"callerid": "up=ginkgo",
		"sub":      "up=ginkgo",
	}

	for k, v := range claims {
		c[k] = v
	}

	token := jwt.NewWithClaims(jwt.GetSigningMethod("RS512"), jwt.MapClaims(c))
	return token.SignedString(pk)
}

func CreateSignedServerJWT(pk *rsa.PrivateKey, pubK ed25519.PublicKey, claims map[string]interface{}) (string, error) {
	c := map[string]interface{}{
		"exp":        time.Now().UTC().Add(time.Hour).Unix(),
		"nbf":        time.Now().UTC().Add(-1 * time.Minute).Unix(),
		"iat":        time.Now().UTC().Unix(),
		"iss":        "Ginkgo",
		"public_key": hex.EncodeToString(pubK),
		"identity":   "ginkgo.example.net",
		"ou":         "choria",
	}
	for k, v := range claims {
		c[k] = v
	}

	token := jwt.NewWithClaims(jwt.GetSigningMethod("RS512"), jwt.MapClaims(c))

	return token.SignedString(pk)
}
