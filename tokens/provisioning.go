// Copyright (c) 2021-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package tokens

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type ProvisioningClaims struct {
	Token        string    `json:"cht"`
	Secure       bool      `json:"chs"`
	URLs         string    `json:"chu,omitempty"`
	SRVDomain    string    `json:"chsrv,omitempty"`
	ProvDefault  bool      `json:"chpd"`
	ProvRegData  string    `json:"chrd,omitempty"`
	ProvFacts    string    `json:"chf,omitempty"`
	ProvNatsUser string    `json:"chusr,omitempty"`
	ProvNatsPass string    `json:"chpwd,omitempty"`
	Extensions   MapClaims `json:"extensions"`

	StandardClaims
}

// NewProvisioningClaims generates new ProvisioningClaims
func NewProvisioningClaims(secure bool, byDefault bool, token string, user string, password string, urls []string, srvDomain string, registrationDataFile string, factsDataFile string, issuer string, validity time.Duration) (*ProvisioningClaims, error) {
	if issuer == "" {
		issuer = "Choria"
	}

	if srvDomain == "" && len(urls) == 0 {
		return nil, fmt.Errorf("srv domain or urls required")
	}

	stdClaims, err := newStandardClaims(issuer, ProvisioningPurpose, validity, true)
	if err != nil {
		return nil, err
	}

	return &ProvisioningClaims{
		Secure:         secure,
		ProvDefault:    byDefault,
		Token:          token,
		ProvNatsUser:   user,
		ProvNatsPass:   password,
		URLs:           strings.Join(urls, ","),
		SRVDomain:      srvDomain,
		ProvRegData:    registrationDataFile,
		ProvFacts:      factsDataFile,
		StandardClaims: *stdClaims,
	}, nil
}

// IsProvisioningToken determines if this is a provisioning token
func IsProvisioningToken(claims StandardClaims) bool {
	if claims.Subject == string(ProvisioningPurpose) {
		return true
	}

	return claims.Purpose == ProvisioningPurpose
}

// ParseProvisioningToken parses token and verifies it with pk
func ParseProvisioningToken(token string, pk any) (*ProvisioningClaims, error) {
	claims := &ProvisioningClaims{}
	err := ParseToken(token, claims, pk)
	if err != nil {
		return nil, fmt.Errorf("could not parse provisioner token: %s", err)
	}

	if !IsProvisioningToken(claims.StandardClaims) {
		return nil, fmt.Errorf("not a provisioning token")
	}

	return claims, nil
}

// ParseProvisioningTokenWithKeyfile parses token and verifies it with the RSA Public key in pkFile, does not support ed25519
func ParseProvisioningTokenWithKeyfile(token string, pkFile string) (*ProvisioningClaims, error) {
	if pkFile == "" {
		return nil, fmt.Errorf("invalid public key file")
	}

	certdat, err := os.ReadFile(pkFile)
	if err != nil {
		return nil, fmt.Errorf("could not read validation certificate: %s", err)
	}

	cert, err := jwt.ParseRSAPublicKeyFromPEM(certdat)
	if err != nil {
		return nil, fmt.Errorf("could not parse validation certificate: %s", err)
	}

	return ParseProvisioningToken(token, cert)
}

// ParseProvisionTokenUnverified parses the provisioning token in an unverified manner.
//
// This is intended to be used for nodes to figure out their settings, they will go try them
// and if nothings there no biggie.  The broker and provisioner WILL validate this token so
// parsing it unverified there is about equivalent to just a configuration file, which is the
// intended purpose of this token and function.
func ParseProvisionTokenUnverified(token string) (*ProvisioningClaims, error) {
	claims := &ProvisioningClaims{}
	_, _, err := new(jwt.Parser).ParseUnverified(token, claims)
	if err != nil {
		return nil, err
	}

	if !IsProvisioningToken(claims.StandardClaims) {
		return nil, fmt.Errorf("token is not a provisioning token")
	}

	return claims, nil
}
