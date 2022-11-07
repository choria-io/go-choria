// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package testutil

import (
	"crypto/ed25519"
	"crypto/rsa"
	"encoding/hex"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

func CreateSignedClientJWT(pk any, claims map[string]any) (string, error) {
	c := map[string]any{
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

	var alg string
	switch pk.(type) {
	case ed25519.PrivateKey:
		alg = "EdDSA"
	case *rsa.PrivateKey:
		alg = "RS512"
	}

	token := jwt.NewWithClaims(jwt.GetSigningMethod(alg), jwt.MapClaims(c))
	return token.SignedString(pk)
}

func CreateSignedServerJWT(pk any, pubK []byte, claims map[string]any) (string, error) {
	c := map[string]any{
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

	var alg string
	switch pk.(type) {
	case ed25519.PrivateKey:
		alg = "EdDSA"
	case *rsa.PrivateKey:
		alg = "RS512"
	}

	token := jwt.NewWithClaims(jwt.GetSigningMethod(alg), jwt.MapClaims(c))

	return token.SignedString(pk)
}
