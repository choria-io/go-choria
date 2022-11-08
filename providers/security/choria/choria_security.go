// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package choria

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/choria-io/go-choria/inter"
	iu "github.com/choria-io/go-choria/internal/util"
	"github.com/choria-io/go-choria/tlssetup"
	"github.com/choria-io/go-choria/tokens"
	"github.com/sirupsen/logrus"
)

var (
	callerFormat = "choria=%s"
	callerIDRe   = regexp.MustCompile(`^[a-z]+=([\w\.\-]+)`)
)

type ChoriaSecurity struct {
	conf *Config
	mu   *sync.Mutex
	log  *logrus.Entry
}

type Config struct {
	// Identity when not empty will force the identity to be used for validations etc
	Identity string

	// SeedFile is the file holding the ed25519 seed
	SeedFile string

	// TokenFile is the file holding the signed JWT file
	TokenFile string

	// TrustedTokenSigners are keys allowed to sign tokens
	TrustedTokenSigners []ed25519.PublicKey

	// Is a URL where a remote signer is running
	RemoteSignerURL string

	// RemoteSignerTokenFile is a file with a token for access to the remote signer
	RemoteSignerTokenFile string

	// TLSSetup is the shared TLS configuration state between security providers
	TLSConfig *tlssetup.Config

	// RemoteSigner is the signer used to sign requests using a remote like AAA Service
	RemoteSigner inter.RequestSigner

	// DisableTLSVerify disables TLS verify in HTTP clients etc
	DisableTLSVerify bool

	// Certificate is the path to the public certificate
	Certificate string

	// Key is the path to the private key
	Key string

	// CA is the path to the Certificate Authority
	CA string

	// SignedReplies indicates that servers replying should sign their messages
	SignedReplies bool

	// InitiatedByServer indicates this is a server, it would require trusted signers
	InitiatedByServer bool
}

func New(opts ...Option) (*ChoriaSecurity, error) {
	s := &ChoriaSecurity{
		conf: &Config{
			SignedReplies: true,
		},
		mu: &sync.Mutex{},
	}

	for _, opt := range opts {
		err := opt(s)
		if err != nil {
			return nil, err
		}
	}

	if s.log == nil {
		return nil, fmt.Errorf("logger is required")
	}

	s.log = s.log.WithFields(logrus.Fields{
		"mTLS":      s.conf.CA != "",
		"delegated": s.conf.RemoteSigner != nil,
		"token":     s.conf.TokenFile,
		"seed":      s.conf.SeedFile,
	})

	s.log.Infof("Security provider initializing")

	return s, nil
}

func (s *ChoriaSecurity) Provider() string {
	return "choria"
}

func (s *ChoriaSecurity) BackingTechnology() inter.SecurityTechnology {
	return inter.SecurityTechnologyED25519JWT
}

func (s *ChoriaSecurity) TokenBytes() ([]byte, error) {
	return os.ReadFile(s.conf.TokenFile)
}

func (s *ChoriaSecurity) Validate() ([]string, bool) {
	var errors []string

	if s.log == nil {
		errors = append(errors, "logger not given")
	}

	if s.conf == nil {
		errors = append(errors, "configuration not given")
	} else {
		if s.conf.Identity == "" {
			errors = append(errors, "identity could not be determine automatically via Choria or was not supplied")
		}

		if s.conf.TokenFile == "" {
			errors = append(errors, "the path to the JWT token is not configured")
		}

		if s.conf.SeedFile == "" {
			errors = append(errors, "the path to the ed25519 seed is not configured")
		}

		if s.conf.InitiatedByServer && len(s.conf.TrustedTokenSigners) == 0 {
			errors = append(errors, "no trusted token signers configured")
		}
	}

	return errors, len(errors) == 0
}

func (s *ChoriaSecurity) Identity() string {
	// TODO: should load the token and figure it out from there
	// ultimately in this case of identity probably just is a hint
	// only as really this is used to find certs by name in other
	// providers, so maybe this is fine
	return s.conf.Identity
}

func (s *ChoriaSecurity) CallerName() string {
	// TODO: since this calls identity the same concerns above apply
	return fmt.Sprintf(callerFormat, s.Identity())
}

func (s *ChoriaSecurity) CallerIdentity(caller string) (string, error) {
	match := callerIDRe.FindStringSubmatch(caller)

	if match == nil {
		return "", fmt.Errorf("could not find a valid caller identity name in %s", caller)
	}

	return match[1], nil
}

func (s *ChoriaSecurity) SignBytes(b []byte) (signature []byte, err error) {
	return iu.Ed25519SignWithSeedFile(s.conf.SeedFile, b)
}

func (s *ChoriaSecurity) VerifySignatureBytes(dat []byte, sig []byte, public ...[]byte) (should bool, signer string) {
	switch len(public) {
	case 0:
		s.log.Warnf("Received a signature verification request with no public parts")
		return false, ""
	case 1:
		// signature was made by the caller - first in the list of tokens - so it may not be one that requires delegated signatures
		return s.verifyByteSignatureByCaller(dat, sig, public[0])
	case 2:
		// signature was made by a delegation - it's the 2nd signature received. We try load it using all the trusted issuers
		// and, we make sure when it loads that it has delegator permission
		return s.verifyByteSignatureByDelegation(dat, sig, public[0], public[1])
	default:
		s.log.Warnf("Received a signature verification request with %d public parts", len(public))
		return false, ""
	}
}

func (s *ChoriaSecurity) verifyByteSignatureByDelegation(dat []byte, sig []byte, caller []byte, delegate []byte) (bool, string) {
	if len(delegate) == 0 {
		s.log.Warnf("Received an invalid token for signature verification")
		return false, ""
	}

	purpose := tokens.TokenPurpose(string(delegate))
	// delegate signers must be clients
	if purpose != tokens.ClientIDPurpose {
		s.log.Warnf("Cannot verify byte signatures using a %s token", purpose)
		return false, ""
	}

	var pk ed25519.PublicKey
	var pks string
	var name string

	for _, signer := range s.conf.TrustedTokenSigners {
		st, err := tokens.ParseClientIDToken(string(delegate), signer, true)
		if err != nil {
			continue
		}

		// it successfully parsed but now must be a delegator else it's not allowed to sign this data
		if st.Permissions == nil || !st.Permissions.AuthenticationDelegator {
			s.log.Warnf("Token attempted to sign a request as delegator without required delegator permission: %s", string(signer))
			return false, ""
		}

		// this ensures/assumes the caller is always signed by the same signer as the delegator, I am not yet sure if this is true
		ct, err := tokens.ParseClientIDToken(string(caller), signer, true)
		if err != nil {
			s.log.Warnf("Could not load caller token using the same signer as the delegator: %v", err)
			return false, ""
		}

		if ct.Permissions == nil || !(ct.Permissions.FleetManagement || ct.Permissions.SignedFleetManagement) {
			s.log.Warnf("Caller token can not be used without fleet management access: %s: %v", string(caller), err)
			return false, ""
		}

		if st.PublicKey != "" {
			pks = st.PublicKey
			name = st.CallerID
			break
		}
	}

	if pks == "" {
		s.log.Warnf("Signer token %s could not be loaded using %d authorized issuers", string(delegate), len(s.conf.TrustedTokenSigners))
		return false, ""
	}

	pk, err := hex.DecodeString(pks)
	if err != nil {
		s.log.Warnf("Could not extract public key from token")
		return false, ""
	}

	ok, err := iu.Ed24419Verify(pk, dat, sig)
	if err != nil {
		s.log.Warnf("Could not verify signature: %v", err)
		return false, ""
	}

	return ok, name
}

func (s *ChoriaSecurity) verifyByteSignatureByCaller(dat []byte, sig []byte, public []byte) (bool, string) {
	if len(public) == 0 {
		s.log.Warnf("Received an invalid token for signature verification")
		return false, ""
	}

	purpose := tokens.TokenPurpose(string(public))
	if purpose != tokens.ServerPurpose && purpose != tokens.ClientIDPurpose {
		s.log.Warnf("Cannot verify byte signatures using a %s token", purpose)
		return false, ""
	}

	var pk ed25519.PublicKey
	var pks string
	var name string

	for _, signer := range s.conf.TrustedTokenSigners {
		if purpose == tokens.ServerPurpose {
			t, err := tokens.ParseServerToken(string(public), signer)
			if err != nil {
				continue
			}

			if t.PublicKey != "" {
				pks = t.PublicKey
				name = t.ChoriaIdentity
				break
			}
		} else {
			t, err := tokens.ParseClientIDToken(string(public), signer, true)
			if err != nil {
				continue
			}

			// it successfully parsed but now must not require delegation
			if t.Permissions != nil && t.Permissions.SignedFleetManagement {
				s.log.Warnf("Could not verify signature by caller which requires authority delegation")
				return false, ""
			}

			if t.Permissions != nil && !t.Permissions.FleetManagement {
				s.log.Warnf("Could not verify signature by caller which does not have fleet management access")
				return false, ""
			}

			if t.PublicKey != "" {
				pks = t.PublicKey
				name = t.CallerID
				break
			}
		}
	}

	if pks == "" {
		s.log.Warnf("Signer token %s could not be loaded using %d authorized issuers", string(public), len(s.conf.TrustedTokenSigners))
		return false, ""
	}

	pk, err := hex.DecodeString(pks)
	if err != nil {
		s.log.Warnf("Could not extract public key from token")
		return false, ""
	}

	ok, err := iu.Ed24419Verify(pk, dat, sig)
	if err != nil {
		s.log.Warnf("Could not verify signature: %v", err)
		return false, ""
	}

	return ok, name
}

func (s *ChoriaSecurity) RemoteSignRequest(ctx context.Context, request []byte) (signed []byte, err error) {
	if s.conf.RemoteSigner == nil {
		return nil, fmt.Errorf("remote signing not configured")
	}

	s.log.Infof("Signing request using %s", s.conf.RemoteSigner.Kind())
	return s.conf.RemoteSigner.Sign(ctx, request, s)
}

func (s *ChoriaSecurity) RemoteSignerToken() ([]byte, error) {
	if s.conf.RemoteSignerTokenFile == "" {
		return nil, fmt.Errorf("no token file defined")
	}

	tb, err := os.ReadFile(s.conf.RemoteSignerTokenFile)
	if err != nil {
		return bytes.TrimSpace(tb), fmt.Errorf("could not read token file: %v", err)
	}

	return tb, err
}

func (s *ChoriaSecurity) RemoteSignerURL() (*url.URL, error) {
	if s.conf.RemoteSignerURL == "" {
		return nil, fmt.Errorf("no remote url configured")
	}

	return url.Parse(s.conf.RemoteSignerURL)
}

func (s *ChoriaSecurity) IsRemoteSigning() bool {
	return s.conf.RemoteSigner != nil
}

func (s *ChoriaSecurity) ChecksumBytes(data []byte) []byte {
	sum := sha256.Sum256(data)

	return sum[:]
}

func (s *ChoriaSecurity) TLSConfig() (*tls.Config, error) {
	tlsc := &tls.Config{
		MinVersion:       tls.VersionTLS12,
		CipherSuites:     s.conf.TLSConfig.CipherSuites,
		CurvePreferences: s.conf.TLSConfig.CurvePreferences,
	}

	if iu.FileExist(s.conf.Key) && iu.FileExist(s.conf.Certificate) {
		cert, err := tls.LoadX509KeyPair(s.conf.Certificate, s.conf.Key)
		if err != nil {
			err = fmt.Errorf("could not load certificate %s and key %s: %s", s.conf.Certificate, s.conf.Key, err)
			return nil, err
		}

		cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
		if err != nil {
			err = fmt.Errorf("error parsing certificate: %v", err)
			return nil, err
		}

		tlsc.Certificates = []tls.Certificate{cert}
	}

	if iu.FileExist(s.conf.CA) {
		caCert, err := os.ReadFile(s.conf.CA)
		if err != nil {
			return nil, err
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		tlsc.ClientCAs = caCertPool
		tlsc.RootCAs = caCertPool
	} else {
		// in this security system we are specifically building a system
		// where mTLS is optional, so when we do not have a CA we disable
		// mutual verification
		tlsc.InsecureSkipVerify = true
	}

	if s.conf.DisableTLSVerify {
		tlsc.InsecureSkipVerify = true
	}

	return tlsc, nil
}

func (s *ChoriaSecurity) ClientTLSConfig() (*tls.Config, error) {
	return s.TLSConfig()
}

func (s *ChoriaSecurity) SSLContext() (*http.Transport, error) {
	tlsConfig, err := s.ClientTLSConfig()
	if err != nil {
		return nil, err
	}

	transport := &http.Transport{TLSClientConfig: tlsConfig}

	return transport, nil
}

func (s *ChoriaSecurity) HTTPClient(secure bool) (*http.Client, error) {
	client := &http.Client{}

	if secure {
		tlsc, err := s.TLSConfig()
		if err != nil {
			return nil, fmt.Errorf("could not set up HTTP connection: %s", err)
		}

		client.Transport = &http.Transport{TLSClientConfig: tlsc}
	}

	return client, nil
}

func (s *ChoriaSecurity) PublicCert() (*x509.Certificate, error) {
	if s.conf.Key == "" || s.conf.Certificate == "" {
		return nil, fmt.Errorf("no certificates configured")
	}

	cert, err := tls.LoadX509KeyPair(s.conf.Certificate, s.conf.Key)
	if err != nil {
		err = fmt.Errorf("could not load certificate %s and key %s: %s", s.conf.Certificate, s.conf.Key, err)
		return nil, err
	}

	cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		err = fmt.Errorf("error parsing certificate: %v", err)
		return nil, err
	}

	return cert.Leaf, nil
}

func (s *ChoriaSecurity) PublicCertBytes() ([]byte, error) {
	if s.conf.Key == "" || s.conf.Certificate == "" {
		return nil, fmt.Errorf("no certificates configured")
	}

	return os.ReadFile(s.conf.Certificate)
}

func (s *ChoriaSecurity) ShouldAllowCaller(name string, callers ...[]byte) (privileged bool, err error) {
	switch len(callers) {
	case 1:
		return s.shouldAllowCallerUnsigned(name, callers[0])
	case 2:
		return s.shouldAllowSignedCaller(name, callers...)
	default:
		return false, fmt.Errorf("invalid caller data provided")
	}
}

func (s *ChoriaSecurity) shouldAllowSignedCaller(name string, callers ...[]byte) (privileged bool, err error) {
	if len(callers) != 2 {
		return false, fmt.Errorf("invalid caller data")
	}

	signerT, err := tokens.ParseClientIDTokenUnverified(string(callers[1]))
	if err != nil {
		return false, fmt.Errorf("invalid signer token: %v", err)
	}

	if signerT.Permissions == nil || !signerT.Permissions.AuthenticationDelegator {
		return false, fmt.Errorf("signer token does not have delegator permission")
	}

	callerT, err := tokens.ParseClientIDTokenUnverified(string(callers[0]))
	if err != nil {
		return false, fmt.Errorf("invalid caller token: %v", err)
	}

	if callerT.Permissions == nil || !(callerT.Permissions.SignedFleetManagement || callerT.Permissions.FleetManagement) {
		return false, fmt.Errorf("caller does not have fleet management access")
	}

	// we do not check the name, delegators can override, but we log the delegation
	s.log.Warnf("Allowing delegator %s to authorize caller %s who holds token %s", signerT.CallerID, name, callerT.CallerID)

	return true, nil
}

func (s *ChoriaSecurity) shouldAllowCallerUnsigned(name string, caller []byte) (privileged bool, err error) {
	// will fail for non client tokens
	// we do not verify since was all verified already in sig check
	// TODO: we should think about servers making requests out to choria services or publishing registration data (1740)
	token, err := tokens.ParseClientIDTokenUnverified(string(caller))
	if err != nil {
		return false, err
	}

	// technically already done in sig verify but cant harm
	if token.Permissions == nil || !(token.Permissions.SignedFleetManagement || token.Permissions.FleetManagement) {
		if token.Permissions.SignedFleetManagement {
			return false, fmt.Errorf("requires signed fleet management access")
		}
		return false, fmt.Errorf("does not have fleet management access")
	}

	if token.CallerID != name {
		return false, fmt.Errorf("caller name does not match token")
	}

	return false, nil
}

func (s *ChoriaSecurity) Enroll(ctx context.Context, wait time.Duration, cb func(digest string, try int)) error {
	return errors.New("the choria security provider does not support enrollment")
}

func (s *ChoriaSecurity) ShouldSignReplies() bool { return s.conf.SignedReplies }
