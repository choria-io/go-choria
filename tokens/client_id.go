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

type ClientPermissions struct {
	// StreamsAdmin enables full access to Choria Streams for all APIs
	StreamsAdmin bool `json:"streams_admin,omitempty"`

	// StreamsUser enables user level access to Choria Streams, no stream admin features
	StreamsUser bool `json:"streams_user,omitempty"`

	// EventsViewer allows viewing lifecycle and auto agent events
	EventsViewer bool `json:"events_viewer,omitempty"`

	// ElectionUser allows using leader elections
	ElectionUser bool `json:"election_user,omitempty"`

	// OrgAdmin has access to all subjects
	OrgAdmin bool `json:"org_admin,omitempty"`

	// ExtendedServiceLifetime allows a token to have a longer than common life time, suitable for services users
	ExtendedServiceLifetime bool `json:"service,omitempty"`
}

// ClientIDClaims represents a user and all AAA Authenticators should create a JWT using this format
//
// The "purpose" claim should be set to ClientIDPurpose
type ClientIDClaims struct {
	// CallerID is the choria caller id that will be set for this user for AAA purposes, typically provider=caller format
	CallerID string `json:"callerid"`

	// AllowedAgents is a list of agent names or agent.action names this user can perform
	AllowedAgents []string `json:"agents,omitempty"`

	// OrganizationUnit is currently unused but will indicate the server account a user should belong to, set to 'choria' now
	OrganizationUnit string `json:"ou,omitempty"`

	// UserProperties is a list of arbitrary properties that can be set for a user, OPA Policies in the token can access these
	UserProperties map[string]string `json:"user_properties,omitempty"`

	// OPAPolicy is a Open Policy Agent document to be used by the signer to limit the users actions
	OPAPolicy string `json:"opa_policy,omitempty"`

	// Permissions sets additional permissions for a client
	Permissions *ClientPermissions `json:"permissions,omitempty"`

	// PublicKey is a ED25519 public key that will be used to sign requests and the server nonce
	PublicKey string `json:"public_key,omitempty"`

	StandardClaims
}

// NewClientIDClaims generates new ClientIDClaims
func NewClientIDClaims(callerID string, allowedAgents []string, org string, properties map[string]string, opaPolicy string, issuer string, validity time.Duration, perms *ClientPermissions, pk ed25519.PublicKey) (*ClientIDClaims, error) {
	if issuer == "" {
		issuer = "Choria"
	}

	if callerID == "" {
		return nil, fmt.Errorf("caller id is required")
	}

	stdClaims, err := newStandardClaims(issuer, ClientIDPurpose, validity, false)
	if err != nil {
		return nil, err
	}

	pubKey := ""
	if pk != nil {
		pubKey = hex.EncodeToString(pk)
	}

	return &ClientIDClaims{
		CallerID:         callerID,
		AllowedAgents:    allowedAgents,
		OrganizationUnit: org,
		UserProperties:   properties,
		OPAPolicy:        opaPolicy,
		Permissions:      perms,
		PublicKey:        pubKey,
		StandardClaims:   *stdClaims,
	}, nil
}

// UnverifiedCallerFromClientIDToken extracts the caller id from a client token.
//
// The token is not verified as this is mainly used on clents who might not have
// the signer public key to verify the certificate. This is safe as the signer
// will later verify the token anyway.
//
// Further, at the moment, we do not verity the Purpose for backward compatibility
//
// An empty callerid will result in a error
func UnverifiedCallerFromClientIDToken(token string) (*jwt.Token, string, error) {
	claims := &ClientIDClaims{}
	t, _, err := new(jwt.Parser).ParseUnverified(token, claims)
	if err != nil {
		return nil, "", err
	}

	if !IsClientIDToken(claims.StandardClaims) {
		return nil, "", fmt.Errorf("not a client id token")
	}

	if claims.CallerID == "" {
		return nil, "", fmt.Errorf("invalid caller id in token")
	}

	return t, claims.CallerID, nil
}

// IsClientIDTokenString calls IsClientIDToken on the token in a string
func IsClientIDTokenString(token string) (bool, error) {
	claims := &ClientIDClaims{}
	_, _, err := new(jwt.Parser).ParseUnverified(token, claims)
	if err != nil {
		return false, err
	}

	return IsClientIDToken(claims.StandardClaims), nil
}

// IsClientIDToken determines if this is a client identifying token
func IsClientIDToken(claims StandardClaims) bool {
	return claims.Purpose == ClientIDPurpose
}

// ParseClientIDTokenUnverified parses the client token in an unverified manner.
func ParseClientIDTokenUnverified(token string) (*ClientIDClaims, error) {
	claims := &ClientIDClaims{}
	_, _, err := new(jwt.Parser).ParseUnverified(token, claims)
	if err != nil {
		return nil, err
	}

	if !IsClientIDToken(claims.StandardClaims) {
		return nil, fmt.Errorf("token is not a client id token")
	}

	return claims, nil
}

// ParseClientIDToken parses token and verifies it with pk
func ParseClientIDToken(token string, pk *rsa.PublicKey, verifyPurpose bool) (*ClientIDClaims, error) {
	claims := &ClientIDClaims{}
	err := ParseToken(token, claims, pk)
	if err != nil {
		return nil, fmt.Errorf("could not parse client id token: %s", err)
	}

	if verifyPurpose {
		if !IsClientIDToken(claims.StandardClaims) {
			return nil, fmt.Errorf("not a client id token")
		}
	}

	return claims, nil
}

// ParseClientIDTokenWithKeyfile parses token and verifies it with the RSA Public key in pkFile
func ParseClientIDTokenWithKeyfile(token string, pkFile string, verifyPurpose bool) (*ClientIDClaims, error) {
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

	return ParseClientIDToken(token, cert, verifyPurpose)
}
