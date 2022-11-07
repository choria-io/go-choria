// Copyright (c) 2021-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package tokens

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rsa"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

const (
	algRS256     = "RS256"
	algRS384     = "RS384"
	algRS512     = "RS512"
	alsEdDSA     = "EdDSA"
	rsaKeyHeader = "-----BEGIN RSA PRIVATE KEY"
	certHeader   = "-----BEGIN CERTIFICATE"
	pkHeader     = "-----BEGIN PUBLIC KEY"
	keyHeader    = "-----BEGIN PRIVATE KEY"
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

	// ServerPurpose indicates a JWT is a ServerClaims JWT
	ServerPurpose Purpose = "choria_server"
)

// MapClaims are free form map claims
type MapClaims jwt.MapClaims

// ParseToken parses token into claims and verify the token is valid using the pk
func ParseToken(token string, claims jwt.Claims, pk any) error {
	if pk == nil {
		return fmt.Errorf("invalid public key")
	}

	_, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		switch t.Method.Alg() {
		case algRS256, algRS512, algRS384:
			pk, ok := pk.(*rsa.PublicKey)
			if !ok {
				return nil, fmt.Errorf("rsa public key required")
			}
			return pk, nil

		case alsEdDSA:
			pk, ok := pk.(ed25519.PublicKey)
			if !ok {
				return nil, fmt.Errorf("ed25519 public key required")
			}

			return pk, nil

		default:
			return nil, fmt.Errorf("unsupported signing method %v in token", t.Method)
		}
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

	if bytes.HasPrefix(keydat, []byte(rsaKeyHeader)) || bytes.HasPrefix(keydat, []byte(keyHeader)) {
		key, err := jwt.ParseRSAPrivateKeyFromPEM(keydat)
		if err != nil {
			return "", fmt.Errorf("could not parse signing key: %s", err)
		}

		return SignToken(claims, key)
	}

	if len(keydat) == ed25519.PrivateKeySize {
		seed, err := hex.DecodeString(string(keydat))
		if err != nil {
			return "", fmt.Errorf("invalid ed25519 seed file: %v", err)
		}

		return SignToken(claims, ed25519.NewKeyFromSeed(seed))
	}

	return "", fmt.Errorf("unsupported key in %v", pkFile)
}

// SignToken signs a JWT using a RSA Private Key
func SignToken(claims jwt.Claims, pk any) (string, error) {
	var stoken string
	var err error

	switch pri := pk.(type) {
	case ed25519.PrivateKey:
		token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
		stoken, err = token.SignedString(pri)

	case *rsa.PrivateKey:
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		stoken, err = token.SignedString(pri)

	default:
		return "", fmt.Errorf("unsupported private key")
	}

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

func readRSAOrED25519PublicData(dat []byte) (any, error) {
	var pk any
	var err error

	if bytes.HasPrefix(dat, []byte(certHeader)) || bytes.HasPrefix(dat, []byte(pkHeader)) {
		pk, err = jwt.ParseRSAPublicKeyFromPEM(dat)
		if err != nil {
			return nil, fmt.Errorf("could not parse validation certificate: %s", err)
		}
	} else {
		edpk, err := hex.DecodeString(string(dat))
		if err != nil {
			return nil, fmt.Errorf("could not parse ed25519 public data: %v", err)
		}
		if len(edpk) != ed25519.PublicKeySize {
			return nil, fmt.Errorf("invalid ed25519 public key size")
		}

		pk = ed25519.PublicKey(edpk)
	}

	return pk, nil
}
