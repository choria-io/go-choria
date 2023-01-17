// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package testutil

import (
	"crypto/ed25519"
	"crypto/rsa"
	"encoding/hex"
	"os"
	"path/filepath"
	"time"

	iu "github.com/choria-io/go-choria/internal/util"
	"github.com/choria-io/tokens"
	"github.com/golang-jwt/jwt/v4"
)

func CreateChoriaTokenAndKeys(targetDir string, tokenSignerFile string, public ed25519.PublicKey, create func(pubK ed25519.PublicKey) (jwt.Claims, error)) (tokenFile string, signedToken string, pubFile string, priFile string, err error) {
	if public == nil {
		priFile = filepath.Join(targetDir, "seed")
		pubFile = filepath.Join(targetDir, "public")

		public, _, err = iu.Ed25519KeyPairToFile(priFile)
		if err != nil {
			return "", "", "", "", err
		}

		err = os.WriteFile(pubFile, []byte(hex.EncodeToString(public)), 0644)
		if err != nil {
			return "", "", "", "", err
		}
	}

	claims, err := create(public)
	if err != nil {
		return "", "", "", "", err
	}

	signed, err := tokens.SignTokenWithKeyFile(claims, tokenSignerFile)
	if err != nil {
		return "", "", "", "", err
	}

	tokenFile = filepath.Join(targetDir, "jwt")
	err = os.WriteFile(tokenFile, []byte(signed), 0644)
	if err != nil {
		return "", "", "", "", err
	}

	return tokenFile, signed, pubFile, priFile, nil
}

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
