// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package tokens

import (
	"crypto/rsa"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// Purpose indicates what kind of token a JWT is and helps us parse it into the right data structure
type Purpose string

const (
	// UnknownPurpose is a JWT that does not have a purpose set
	UnknownPurpose Purpose = ""

	// ClientIDPurpose indicates a JWT is a ClientIDClaims JWT
	ClientIDPurpose Purpose = "choria_client_id"

	// ProvisioningPurpose indicates a JWT is a ProvisioningClaims JWT
	ProvisioningPurpose Purpose = "choria_provisioning"
)

// ParseToken parses token into claims and verify the token is valid using the pk
func ParseToken(token string, claims jwt.Claims, pk *rsa.PublicKey) error {
	if pk == nil {
		return fmt.Errorf("invalid public key")
	}

	_, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unsupported signing method in token")
		}

		return pk, nil
	})

	return err
}

// ParseTokenUnverified parses token into claims and DOES not verify the token validity in any way
func ParseTokenUnverified(token string) (jwt.MapClaims, error) {
	parser := new(jwt.Parser)
	claims := new(jwt.MapClaims)
	_, _, err := parser.ParseUnverified(token, claims)
	return *claims, err
}

// TokenPurpose parses, without validating, token and checks for a Purpose field in it
func TokenPurpose(token string) Purpose {
	parser := new(jwt.Parser)
	claims := StandardClaims{}
	parser.ParseUnverified(token, &claims)

	if claims.Purpose == UnknownPurpose {
		if claims.RegisteredClaims.Subject == string(ProvisioningPurpose) {
			return ProvisioningPurpose
		}
	}

	return claims.Purpose
}

// TokenPurposeBytes called TokenPurpose with a bytes input
func TokenPurposeBytes(token []byte) Purpose {
	return TokenPurpose(string(token))
}

// SignTokenWithKeyFile signs a JWT using a RSA Private Key in PEM format
func SignTokenWithKeyFile(claims jwt.Claims, pkFile string) (string, error) {
	keydat, err := os.ReadFile(pkFile)
	if err != nil {
		return "", fmt.Errorf("could not read signing key: %s", err)
	}

	key, err := jwt.ParseRSAPrivateKeyFromPEM(keydat)
	if err != nil {
		return "", fmt.Errorf("could not parse signing key: %s", err)
	}

	return SignToken(claims, key)
}

// SignToken signs a JWT using a RSA Private Key
func SignToken(claims jwt.Claims, pk *rsa.PrivateKey) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	stoken, err := token.SignedString(pk)
	if err != nil {
		return "", fmt.Errorf("could not sign token using key: %s", err)
	}

	return stoken, nil
}

// SaveAndSignTokenWithKeyFile signs a token using SignTokenWithKeyFile and saves it to outFile
func SaveAndSignTokenWithKeyFile(claims jwt.Claims, pkFile string, outFile string, perm os.FileMode) error {
	token, err := SignTokenWithKeyFile(claims, pkFile)
	if err != nil {
		return err
	}

	return os.WriteFile(outFile, []byte(token), perm)
}

func newStandardClaims(issuer string, purpose Purpose, validity time.Duration, setSubject bool) (*StandardClaims, error) {
	now := jwt.NewNumericDate(time.Now().UTC())
	claims := &StandardClaims{
		Purpose: purpose,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			IssuedAt:  now,
			NotBefore: now,
		},
	}

	if validity > 0 {
		claims.RegisteredClaims.ExpiresAt = jwt.NewNumericDate(now.Add(validity))
	}

	if setSubject {
		claims.Subject = string(purpose)
	}

	return claims, nil
}
