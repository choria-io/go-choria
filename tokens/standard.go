// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package tokens

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	iu "github.com/choria-io/go-choria/internal/util"
	"github.com/golang-jwt/jwt/v4"
	"github.com/segmentio/ksuid"
)

type StandardClaims struct {
	// Purpose indicates the type of JWT for type discovery
	Purpose Purpose `json:"purpose"`

	// TrustChainSignature is a structure that helps to verify a chain of trust to a org issuer
	TrustChainSignature string `json:"tcs,omitempty"`

	// PublicKey is a ED25519 public key associated with this token
	PublicKey string `json:"public_key,omitempty"`

	// IssuerExpiresAt is the expiry time of the issuer, if set will be checked in addition to the expiry time of the token itself
	IssuerExpiresAt *jwt.NumericDate `json:"issexp,omitempty"`

	jwt.RegisteredClaims
}

// ExpireTime determines the expiry time based on issuer expiry and token expiry
func (s *StandardClaims) ExpireTime() time.Time {
	var iexp, exp time.Time
	if s.IssuerExpiresAt != nil {
		iexp = s.IssuerExpiresAt.Time
	}
	if s.ExpiresAt != nil {
		exp = s.ExpiresAt.Time
	}

	if iexp.IsZero() {
		return exp
	}

	if exp.IsZero() {
		return iexp
	}

	if iexp.Before(exp) {
		return iexp
	}

	return exp
}

// IsExpired checks if the token has expired
func (s *StandardClaims) IsExpired() bool {
	return time.Now().After(s.ExpireTime())
}

// AddOrgIssuerData adds the data that a Chain Issuer needs to be able to issue clients in an Org managed by an Issuer
func (c *StandardClaims) AddOrgIssuerData(priK ed25519.PrivateKey) error {
	dat, err := c.OrgIssuerChainData()
	if err != nil {
		return err
	}

	sig, err := iu.Ed25519Sign(priK, dat)
	if err != nil {
		return err
	}

	c.SetOrgIssuer(priK.Public().(ed25519.PublicKey))
	c.SetChainIssuerTrustSignature(sig)

	return nil
}

// AddChainIssuerData adds the data that a Signed token needs from a Chain Issuer in an Org managed by an Issuer
func (c *StandardClaims) AddChainIssuerData(chainIssuer *ClientIDClaims, prik ed25519.PrivateKey) error {
	err := c.SetChainIssuer(chainIssuer)
	if err != nil {
		return err
	}

	udat, err := c.ChainIssuerData(chainIssuer.TrustChainSignature)
	if err != nil {
		return err
	}

	usig, err := iu.Ed25519Sign(prik, udat)
	if err != nil {
		return err
	}

	c.SetChainUserTrustSignature(chainIssuer, usig)

	return nil
}

// true if not expired
func (c *StandardClaims) verifyIssuerExpiry(req bool) bool {
	// org issuer tokens has a tcs but the org issuer has no expiry time so we can skip
	if !strings.HasPrefix(c.Issuer, ChainIssuerPrefix) {
		return !req
	}

	// without a tcs this isn't a chained token so there's no point in validating
	if c.TrustChainSignature == "" {
		return !req
	}

	if c.IssuerExpiresAt == nil {
		return !req
	}

	return !c.IsExpired()
}

// IsChainedIssuer determines if this is a token capable of issuing users as part of a chain
// without verify being true one can not be 100% certain it's valid to do that but its a strong hint
func (c *StandardClaims) IsChainedIssuer(verify bool) bool {
	if len(c.TrustChainSignature) == 0 {
		return false
	}

	if !strings.HasPrefix(c.Issuer, OrgIssuerPrefix) {
		return false
	}

	if !verify {
		return true
	}

	dat, err := c.OrgIssuerChainData()
	if err != nil {
		return false
	}

	pubK, err := hex.DecodeString(strings.TrimPrefix(c.Issuer, OrgIssuerPrefix))
	if err != nil {
		return false
	}

	sig, err := hex.DecodeString(c.TrustChainSignature)
	if err != nil {
		return false
	}

	ok, _ := iu.Ed25519Verify(pubK, dat, sig)

	return ok
}

// OrgIssuerChainData creates data that the org issuer would sign and embed in the token as TrustChainSignature.
// See AddOrgIssuerData for a one-shot way to set the needed data when you have access to the private key.
func (c *StandardClaims) OrgIssuerChainData() ([]byte, error) {
	if c.ID == "" {
		return nil, fmt.Errorf("no token id set")
	}
	if c.PublicKey == "" {
		return nil, fmt.Errorf("no public key set")
	}

	return []byte(fmt.Sprintf("%s.%s", c.ID, c.PublicKey)), nil
}

// SetOrgIssuer sets the issuer field for users issued by the Org Issuer
// See AddOrgIssuerData for a one-shot way to set the needed data when you have access to the private key.
func (c *StandardClaims) SetOrgIssuer(pk ed25519.PublicKey) {
	c.Issuer = fmt.Sprintf("%s%s", OrgIssuerPrefix, hex.EncodeToString(pk))
}

// SetChainIssuer used by Login Handlers that create users in a chain to set an appropriate issuer on created users
// See AddChainIssuerData for a one-shot way to set the needed data when you have access to the private key.
func (c *StandardClaims) SetChainIssuer(ci *ClientIDClaims) error {
	if ci.ID == "" {
		return fmt.Errorf("id not set")
	}
	if ci.PublicKey == "" {
		return fmt.Errorf("public key not set")
	}

	c.Issuer = fmt.Sprintf("%s%s.%s", ChainIssuerPrefix, ci.ID, ci.PublicKey)
	c.IssuerExpiresAt = ci.ExpiresAt

	return nil
}

// ChainIssuerData is the data that should be signed on a user to create a chain of trust between Org Issuer, Client Login Handler and Client.
//
// The Issuer should already be set using SetChainIssuer()
//
// See AddChainIssuerData for a one-shot way to set the needed data when you have access to the private key.
func (c *StandardClaims) ChainIssuerData(chainSig string) ([]byte, error) {
	if c.ID == "" {
		return nil, fmt.Errorf("id not set")
	}
	if c.Issuer == "" {
		return nil, fmt.Errorf("issuer not set")
	}
	if !strings.HasPrefix(c.Issuer, ChainIssuerPrefix) {
		return nil, fmt.Errorf("invalid issuer prefix")
	}

	parts := strings.Split(strings.TrimPrefix(c.Issuer, ChainIssuerPrefix), ".")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid issuer data")
	}

	return []byte(fmt.Sprintf("%s.%s", c.ID, chainSig)), nil
}

// SetChainUserTrustSignature sets the TrustChainSignature for a user issued by a ChainIssuer like AAA Login Server
//
// See AddChainIssuerData for a one-shot way to set the needed data when you have access to the private key.
func (c *StandardClaims) SetChainUserTrustSignature(h *ClientIDClaims, sig []byte) {
	c.TrustChainSignature = fmt.Sprintf("%s.%s", h.TrustChainSignature, hex.EncodeToString(sig))
}

// SetChainIssuerTrustSignature sets the TrustChainSignature for a user who may issue others like a AAA Login Server
//
// See AddChainIssuerData for a one-shot way to set the needed data when you have access to the private key.
func (c *StandardClaims) SetChainIssuerTrustSignature(sig []byte) {
	c.TrustChainSignature = hex.EncodeToString(sig)
}

// ParseChainIssuerData extract the chain verifier based signature and metadata from a token
func (c *StandardClaims) ParseChainIssuerData() (id string, pk ed25519.PublicKey, tcs string, sig []byte, err error) {
	issuerChainData := strings.TrimPrefix(c.Issuer, ChainIssuerPrefix)
	parts := strings.Split(issuerChainData, ".")
	if len(parts) != 2 {
		return "", nil, "", nil, fmt.Errorf("invalid issuer content")
	}

	if len(parts[0]) == 0 {
		return "", nil, "", nil, fmt.Errorf("invalid id in issuer")
	}
	if len(parts[1]) == 0 {
		return "", nil, "", nil, fmt.Errorf("invalid public key in issuer")
	}

	id = parts[0]
	pks := parts[1]

	hPubk, err := hex.DecodeString(pks)
	if err != nil {
		return "", nil, "", nil, fmt.Errorf("invalid public key in issuer data")
	}

	parts = strings.Split(c.TrustChainSignature, ".")
	if len(parts) != 2 {
		return "", nil, "", nil, fmt.Errorf("invalid trust chain signature")
	}
	if len(parts[0]) == 0 || len(parts[1]) == 0 {
		return "", nil, "", nil, fmt.Errorf("invalid trust chain signature")
	}
	tcs = parts[0]
	sig, err = hex.DecodeString(parts[1])
	if err != nil {
		return "", nil, "", nil, fmt.Errorf("invalid signature in chain signature: %w", err)
	}

	return id, hPubk, tcs, sig, err
}

func (c *StandardClaims) verifyIssuerRequiredClaims() error {
	if c.Issuer == "" {
		return fmt.Errorf("no issuer set")
	}
	if c.PublicKey == "" {
		return fmt.Errorf("no public key set")
	}
	if c.TrustChainSignature == "" {
		return fmt.Errorf("no trust chain signature set")
	}
	if c.ID == "" {
		return fmt.Errorf("id not set")
	}
	if c.IssuedAt == nil || c.IssuedAt.IsZero() {
		return fmt.Errorf("no issued time set")
	}
	if c.ExpiresAt == nil || c.ExpiresAt.IsZero() {
		return fmt.Errorf("no expires set")
	}
	kid, err := ksuid.Parse(c.ID)
	if err != nil {
		return fmt.Errorf("invalid ksuid format")
	}
	if !c.IssuedAt.Equal(kid.Time()) {
		return fmt.Errorf("id is not based on issued time")
	}

	return nil
}

// IsSignedByIssuer uses the chain data in Issuer and TrustChainSignature to determine if an issuer signed a token
func (c *StandardClaims) IsSignedByIssuer(pk ed25519.PublicKey) (bool, ed25519.PublicKey, error) {
	err := c.verifyIssuerRequiredClaims()
	if err != nil {
		return false, nil, err
	}

	switch {
	case strings.HasPrefix(c.Issuer, OrgIssuerPrefix):
		// This would be a token that is allowed to create clients in a chain.
		//
		// Its Issuer is set to I-issuerPubk
		// Its chain sig is signed by the issuer "<id>.<pubk>" of this token, obtained from OrgIssuerChainData()
		//
		// So we simply check if the signature in the TrustChainSignature match the data if signed by the
		// supplied issuer public key

		if c.Issuer != fmt.Sprintf("%s%s", OrgIssuerPrefix, hex.EncodeToString(pk)) {
			return false, nil, fmt.Errorf("public keys do not match")
		}

		sig, err := hex.DecodeString(c.TrustChainSignature)
		if err != nil {
			return false, nil, fmt.Errorf("invalid trust chain signature: %w", err)
		}

		dat, err := c.OrgIssuerChainData()
		if err != nil {
			return false, nil, err
		}

		valid, err := iu.Ed25519Verify(pk, dat, sig)
		if err != nil {
			return false, nil, err
		}

		return valid, pk, err

	case strings.HasPrefix(c.Issuer, ChainIssuerPrefix):
		// This is a token that was created by one in the chain - not the org issuer.
		//
		// Its Issuer is set to C-<creator id>.<creator pubk>
		// Its chain sig is set to <creator tcs>.hex(sign(tID,<creator tcs>))
		//
		// We know what the content of tcs unsigned is from the Issuer field
		// and we are given the issuer public key, so we can confirm the issuer
		// we are interested in signed the tcs.
		//
		// We know the holder of the creator private key made it because we have
		// it's public key and can confirm that, we know its the public key of
		// the creator since its in the tcs set there by our trusted issuer.
		//
		// We can confirm the tcs is valid and matches whats in the sig made by
		// the creator because we verify it using the requested issuer pubk
		if c.IssuerExpiresAt == nil || c.IssuerExpiresAt.IsZero() {
			return false, nil, fmt.Errorf("no issuer expires set")
		}

		_, hPubk, tcs, sig, err := c.ParseChainIssuerData()
		if err != nil {
			return false, nil, err
		}

		// this is the signature from the handler
		// now we check the signature is data + "." + sig(id+ "." + data)
		ok, err := iu.Ed25519Verify(hPubk, []byte(fmt.Sprintf("%s.%s", c.ID, tcs)), sig)
		if err != nil {
			return false, nil, fmt.Errorf("chain signature validation failed: %w", err)
		}
		if !ok {
			return false, nil, fmt.Errorf("invalid chain signature")
		}

		return true, hPubk, nil

	default:
		return false, nil, fmt.Errorf("unsupported issuer format")
	}
}
