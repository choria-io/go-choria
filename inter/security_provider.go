// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package inter

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"time"
)

// SecurityProvider provides a security plugin for the choria subsystem
type SecurityProvider interface {
	// Provider reports the name of the current security provider
	Provider() string

	// Validate that the security provider is functional
	Validate() ([]string, bool)

	// Identity from the active certificates
	Identity() string

	// CallerName is a valid choria like foo=bar style caller name from the identity
	CallerName() string

	// CallerIdentity extracts the Identity from a caller name
	CallerIdentity(caller string) (string, error)

	// SignBytes signs bytes using the current active certificate
	SignBytes(b []byte) (signature []byte, err error)

	// VerifyByteSignature verifies that dat signature was made using pubcert
	VerifyByteSignature(dat []byte, sig []byte, pubcert []byte) (should bool, signer string)

	// SignString signs a string using the current active certificate
	SignString(s string) (signature []byte, err error)

	// RemoteSignRequest signs a choria request using a remote signer and returns a secure request
	RemoteSignRequest(ctx context.Context, str []byte) (signed []byte, err error)

	// IsRemoteSigning reports if the security provider is signing using a remote
	IsRemoteSigning() bool

	// ChecksumBytes produce a crypto checksum for data
	ChecksumBytes(data []byte) []byte

	// ChecksumString produce a crypto checksum for data
	ChecksumString(data string) []byte

	// TLSConfig produce a tls.Config for the current identity using it's certificates etc
	TLSConfig() (*tls.Config, error)

	// ClientTLSConfig produces a tls.Config specifically for clients
	ClientTLSConfig() (*tls.Config, error)

	// SSLContext produce a http.Transport for the current identity using it's certificates etc
	SSLContext() (*http.Transport, error)

	// HTTPClient creates a standard HTTP client with optional security, it will
	// be set to use the CA and client certs for auth.
	HTTPClient(secure bool) (*http.Client, error)

	// VerifyCertificate validates that a certificate is signed by a known CA
	VerifyCertificate(certpem []byte, identity string) error

	// PublicCert is the parsed public certificate
	PublicCert() (*x509.Certificate, error)

	// PublicCertPem retrieves pem data for the public certificate of the current identity
	PublicCertPem() (*pem.Block, error)

	// PublicCertBytes retrieves pem data in textual form for the public certificate of the current identity
	PublicCertBytes() ([]byte, error)

	// ShouldAllowCaller validates the identity, the public data like certificate or JWT and checks
	// against allowed lists and is privileged user aware
	ShouldAllowCaller(data []byte, name string) (privileged bool, err error)

	// Enroll creates a new cert with the active identity and attempt to enroll it with the security system
	// if there's a process of waiting for the certificate to be signed for example this should wait
	// no more than wait.  cb gets called on every attempt to download a cert with the attempt number
	// as argument
	Enroll(ctx context.Context, wait time.Duration, cb func(digest string, try int)) error
}
