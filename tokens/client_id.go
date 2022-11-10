// Copyright (c) 2021-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package tokens

import (
	"crypto/ed25519"
	"crypto/md5"
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

	// SystemUser allows accessing the Choria Broker system account without verified TLS
	SystemUser bool `json:"system_user,omitempty"`

	// Governor enables access to Governors, cannot make new ones, also requires Streams permission
	Governor bool `json:"governor,omitempty"`

	// OrgAdmin has access to all subjects and broker system account
	OrgAdmin bool `json:"org_admin,omitempty"`

	// FleetManagement enables access to the choria server fleet for RPCs
	FleetManagement bool `json:"fleet_management,omitempty"`

	// SignedFleetManagement requires a user to have a valid signature by an AuthenticationDelegator to interact with the fleet
	SignedFleetManagement bool `json:"signed_fleet_management,omitempty"`

	// ExtendedServiceLifetime allows a token to have a longer than common lifetime, suitable for services users
	ExtendedServiceLifetime bool `json:"service,omitempty"`

	// AuthenticationDelegator has the right to sign requests on behalf of others
	AuthenticationDelegator bool `json:"authentication_delegator,omitempty"`
}

// ClientIDClaims represents a user and all AAA Authenticators should create a JWT using this format
//
// The "purpose" claim should be set to ClientIDPurpose
type ClientIDClaims struct {
	// CallerID is the choria caller id that will be set for this user for AAA purposes, typically provider=caller format
	CallerID string `json:"callerid"`

	// AllowedAgents is a list of agent names or agent.action names this user can perform
	AllowedAgents []string `json:"agents,omitempty"`

	// OrganizationUnit broker account a user should belong to, set to 'choria' now and issuing organization
	OrganizationUnit string `json:"ou,omitempty"`

	// UserProperties is a list of arbitrary properties that can be set for a user, OPA Policies in the token can access these
	UserProperties map[string]string `json:"user_properties,omitempty"`

	// OPAPolicy is a Open Policy Agent document to be used by the signer to limit the users actions
	OPAPolicy string `json:"opa_policy,omitempty"`

	// Permissions sets additional permissions for a client
	Permissions *ClientPermissions `json:"permissions,omitempty"`

	// AdditionalPublishSubjects are additional subjects the client can publish to
	AdditionalPublishSubjects []string `json:"pub_subjects,omitempty"`

	// AdditionalSubscribeSubjects are additional subjects the client can subscribe to
	AdditionalSubscribeSubjects []string `json:"sub_subjects,omitempty"`

	StandardClaims
}

var (
	ErrNotAClientToken       = fmt.Errorf("not a client token")
	ErrInvalidClientCallerID = fmt.Errorf("invalid caller id in token")
)

// UniqueID returns the caller id and unique id used to generate private inboxes
func (c *ClientIDClaims) UniqueID() (id string, uid string) {
	return c.CallerID, fmt.Sprintf("%x", md5.Sum([]byte(c.CallerID)))
}

// NewClientIDClaims generates new ClientIDClaims
func NewClientIDClaims(callerID string, allowedAgents []string, org string, properties map[string]string, opaPolicy string, issuer string, validity time.Duration, perms *ClientPermissions, pk ed25519.PublicKey) (*ClientIDClaims, error) {
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

	if org == "" {
		org = defaultOrg
	}

	stdClaims.PublicKey = pubKey

	return &ClientIDClaims{
		CallerID:         callerID,
		AllowedAgents:    allowedAgents,
		OrganizationUnit: org,
		UserProperties:   properties,
		OPAPolicy:        opaPolicy,
		Permissions:      perms,
		StandardClaims:   *stdClaims,
	}, nil
}

// UnverifiedCallerFromClientIDToken extracts the caller id from a client token.
//
// The token is not verified as this is mainly used on clents who might not have
// the signer public key to verify the certificate. This is safe as the signer
// will later verify the token anyway.
//
// # Further, at the moment, we do not verity the Purpose for backward compatibility
//
// An empty callerid will result in an error
func UnverifiedCallerFromClientIDToken(token string) (*jwt.Token, string, error) {
	claims := &ClientIDClaims{}
	t, _, err := new(jwt.Parser).ParseUnverified(token, claims)
	if err != nil {
		return nil, "", err
	}

	if !IsClientIDToken(claims.StandardClaims) {
		return nil, "", ErrNotAClientToken
	}

	if claims.CallerID == "" {
		return nil, "", ErrInvalidClientCallerID
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
		return nil, ErrNotAClientToken
	}

	return claims, nil
}

// ParseClientIDToken parses token and verifies it with pk
func ParseClientIDToken(token string, pk any, verifyPurpose bool) (*ClientIDClaims, error) {
	claims := &ClientIDClaims{}
	err := ParseToken(token, claims, pk)
	if err != nil {
		return nil, fmt.Errorf("could not parse client id token: %w", err)
	}

	if verifyPurpose && !IsClientIDToken(claims.StandardClaims) {
		return nil, ErrNotAClientToken
	}

	// if we have a tcs we require an issuer expiry to be set and it to not have expired
	if !claims.StandardClaims.verifyIssuerExpiry(claims.TrustChainSignature != "") {
		return nil, jwt.ErrTokenExpired
	}

	return claims, nil
}

// ParseClientIDTokenWithKeyfile parses token and verifies it with the RSA Public key in pkFile, does not support ed25519 public keys in a file
func ParseClientIDTokenWithKeyfile(token string, pkFile string, verifyPurpose bool) (*ClientIDClaims, error) {
	if pkFile == "" {
		return nil, fmt.Errorf("invalid public key file")
	}

	certdat, err := os.ReadFile(pkFile)
	if err != nil {
		return nil, fmt.Errorf("could not read validation certificate: %s", err)
	}

	pk, err := readRSAOrED25519PublicData(certdat)
	if err != nil {
		return nil, err
	}

	return ParseClientIDToken(token, pk, verifyPurpose)
}
