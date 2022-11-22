// Copyright (c) 2021-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package tokens

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/choria-io/go-choria/build"
	iu "github.com/choria-io/go-choria/internal/util"
	"github.com/golang-jwt/jwt/v4"
	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

const (
	algRS256          = "RS256"
	algRS384          = "RS384"
	algRS512          = "RS512"
	algEdDSA          = "EdDSA"
	rsaKeyHeader      = "-----BEGIN RSA PRIVATE KEY"
	certHeader        = "-----BEGIN CERTIFICATE"
	pkHeader          = "-----BEGIN PUBLIC KEY"
	keyHeader         = "-----BEGIN PRIVATE KEY"
	defaultOrg        = "choria"
	OrgIssuerPrefix   = "I-"
	ChainIssuerPrefix = "C-"
)

var defaultIssuer = fmt.Sprintf("Choria Tokens Package v%s", build.Version)

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

// ParseToken parses token into claims and verify the token is valid using the pk,
// if the token is signed by a chain issuer then pk must be the org issuer pk and
// the chain will be verified
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

		case algEdDSA:
			pk, ok := pk.(ed25519.PublicKey)
			if !ok {
				return nil, fmt.Errorf("ed25519 public key required")
			}

			var sc *StandardClaims

			// if its a client and from a chain we will verify it using the chain issuer pubk
			client, ok := claims.(*ClientIDClaims)
			if ok && strings.HasPrefix(client.Issuer, ChainIssuerPrefix) {
				sc = &client.StandardClaims
			}

			// if its a server and from a chain we will verify it using the chain issuer pubk
			server, ok := claims.(*ServerClaims)
			if ok && strings.HasPrefix(server.Issuer, ChainIssuerPrefix) {
				sc = &server.StandardClaims
			}

			if sc != nil {
				valid, signerPk, err := sc.IsSignedByIssuer(pk)
				if err != nil {
					return nil, fmt.Errorf("not signed by issuer: %w", err)
				}
				if !valid {
					return nil, fmt.Errorf("token not signed by issuer")
				}
				pk = signerPk
			}

			return pk, nil

		default:
			return nil, fmt.Errorf("unsupported signing method %v in token", t.Method)
		}
	})
	if err != nil {
		return err
	}

	return nil
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

// SaveAndSignTokenWithVault signs a token using the named key in a Vault Transit engine.  Requires VAULT_TOKEN and VAULT_ADDR to be set.
func SaveAndSignTokenWithVault(ctx context.Context, claims jwt.Claims, key string, outFile string, perm os.FileMode, tlsc *tls.Config, log *logrus.Entry) error {
	vt := os.Getenv("VAULT_TOKEN")
	va := os.Getenv("VAULT_ADDR")

	if vt == "" || va == "" {
		return fmt.Errorf("requires VAULT_TOKEN and VAULT_ADDR environment variables")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	ss, err := token.SigningString()
	if err != nil {
		return err
	}

	uri, err := url.Parse(va)
	if err != nil {
		return err
	}
	uri.Path = fmt.Sprintf("/v1/transit/sign/%s", key)

	dat := map[string]any{
		"signature_algorithm": "ed25519",
		"input":               base64.StdEncoding.EncodeToString([]byte(ss)),
	}
	jdat, err := json.Marshal(dat)
	if err != nil {
		return err
	}
	log.Debugf("JSON Request: %s", string(jdat))

	client := &http.Client{}
	if tlsc != nil {
		client.Transport = &http.Transport{TLSClientConfig: tlsc}
	}

	req, err := http.NewRequestWithContext(ctx, "POST", uri.String(), bytes.NewBuffer(jdat))
	if err != nil {
		return err
	}
	req.Header.Add("X-Vault-Token", vt)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("request failed: code: %d: %s", resp.StatusCode, string(body))
	}

	sig := gjson.GetBytes(body, "data.signature")
	if !sig.Exists() {
		return fmt.Errorf("no signature in response: %s", string(body))
	}

	sigs := sig.String()
	const vaultSigPrefix = "vault:v1:"

	if !strings.HasPrefix(sigs, vaultSigPrefix) {
		return fmt.Errorf("invalid signature, no vault:v1 prefix")
	}

	signed := fmt.Sprintf("%s.%s", ss, strings.TrimPrefix(sigs, vaultSigPrefix))

	return os.WriteFile(outFile, []byte(signed), perm)
}

func newStandardClaims(issuer string, purpose Purpose, validity time.Duration, setSubject bool) (*StandardClaims, error) {
	if issuer == "" {
		issuer = defaultIssuer
	}

	now := jwt.NewNumericDate(time.Now().UTC())
	claims := &StandardClaims{
		Purpose: purpose,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        strings.ReplaceAll(iu.UniqueID(), "-", ""),
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
