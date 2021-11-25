// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package tokens

import (
	"crypto/ed25519"
	"crypto/rsa"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type ServerPermissions struct {
	// Submission enables access to <collective>.submission.in.>
	Submission bool `json:"submission"`

	// Streams allow access to Choria Streams such as reading KV values
	Streams bool `json:"streams"`

	// ServiceHost allows a node to listen for service requests
	ServiceHost bool `json:"service_host"`
}

type ServerClaims struct {
	// ChoriaIdentity is the server identity
	ChoriaIdentity string `json:"identity"`

	// Collectives sets what collectives this server belongs to within the organization
	Collectives []string `json:"collectives"`

	// PublicKey is a ED25519 public key use to sign server nonce and other parts
	PublicKey string `json:"public_key"`

	// Permissions are additional abilities the server will have
	Permissions *ServerPermissions `json:"permissions,omitempty"`

	// OrganizationUnit is currently unused but will indicate the server account a node should belong to, set to 'choria' now
	OrganizationUnit string `json:"ou,omitempty"`

	// AdditionalPublishSubjects are additional subjects the server can publish to facilitate for example custom registration paths
	AdditionalPublishSubjects []string `json:"pub_subjects,omitempty"`

	StandardClaims
}

func NewServerClaims(identity string, collectives []string, org string, perms *ServerPermissions, additionalPublish []string, pk ed25519.PublicKey, issuer string, validity time.Duration) (*ServerClaims, error) {
	if identity == "" {
		return nil, fmt.Errorf("identity is required")
	}

	if len(collectives) == 0 {
		return nil, fmt.Errorf("at least one collective is required")
	}

	if pk == nil {
		return nil, fmt.Errorf("public key is required")
	}

	if org == "" {
		org = "choria"
	}

	if issuer == "" {
		issuer = "choria"
	}

	stdClaims, err := newStandardClaims(issuer, ServerPurpose, validity, false)
	if err != nil {
		return nil, err
	}

	return &ServerClaims{
		ChoriaIdentity:            identity,
		Collectives:               collectives,
		PublicKey:                 hex.EncodeToString(pk),
		Permissions:               perms,
		OrganizationUnit:          org,
		AdditionalPublishSubjects: additionalPublish,
		StandardClaims:            *stdClaims,
	}, nil
}

// UnverifiedIdentityFromServerToken extracts the identity from a server token.
//
// The token is not verified as this is mainly used on servers who might not have
// the signer public key to verify the certificate. This is safe as the signer
// will later verify the token anyway.
//
// An empty identity will result in an error
func UnverifiedIdentityFromServerToken(token string) (*jwt.Token, string, error) {
	claims := &ServerClaims{}
	t, _, err := new(jwt.Parser).ParseUnverified(token, claims)
	if err != nil {
		return nil, "", err
	}

	if !IsServerToken(claims.StandardClaims) {
		return nil, "", fmt.Errorf("not a server token")
	}

	if claims.ChoriaIdentity == "" {
		return nil, "", fmt.Errorf("invalid identity in token")
	}

	return t, claims.ChoriaIdentity, nil
}

func IsServerTokenString(token string) (bool, error) {
	claims := &ServerClaims{}
	_, _, err := new(jwt.Parser).ParseUnverified(token, claims)
	if err != nil {
		return false, err
	}

	return IsServerToken(claims.StandardClaims), nil
}

// IsServerToken determines if this is a server token
func IsServerToken(claims StandardClaims) bool {
	return claims.Purpose == ServerPurpose
}

// ParseServerTokenUnverified parses the server token in an unverified manner.
func ParseServerTokenUnverified(token string) (*ServerClaims, error) {
	claims := &ServerClaims{}
	_, _, err := new(jwt.Parser).ParseUnverified(token, claims)
	if err != nil {
		return nil, err
	}

	if !IsServerToken(claims.StandardClaims) {
		return nil, fmt.Errorf("not a server token")
	}

	return claims, nil
}

// ParseServerToken parses token and verifies it with pk
func ParseServerToken(token string, pk *rsa.PublicKey) (*ServerClaims, error) {
	claims := &ServerClaims{}
	err := ParseToken(token, claims, pk)
	if err != nil {
		return nil, fmt.Errorf("could not parse client id token: %s", err)
	}

	if !IsServerToken(claims.StandardClaims) {
		return nil, fmt.Errorf("not a server token")
	}

	return claims, nil
}

// ParseServerTokenWithKeyfile parses token and verifies it with the RSA Public key in pkFile
func ParseServerTokenWithKeyfile(token string, pkFile string) (*ServerClaims, error) {
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

	return ParseServerToken(token, cert)
}
