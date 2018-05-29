package security

import (
	context "context"
	"crypto/tls"
	"encoding/pem"
	"net/http"
	"time"
)

// Provider provides a security plugin for the choria subsystem
type Provider interface {
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

	// VerifyByteSignature verifies that str when signed by identity would match signature.
	// The certificate for identity should previously have been saved into the cache
	VerifyByteSignature(str []byte, signature []byte, identity string) bool

	// SignString signs a string using the current active certificate
	SignString(s string) (signature []byte, err error)

	// VerifyStringSignature verifies that str when signed by identity would match signature.
	// The certificate for identity should previously have been saved into the cache
	VerifyStringSignature(str string, signature []byte, identity string) bool

	// PrivilegedVerifyByteSignature verifies that dat is a valid signature for identity
	// or any of the privileged certificates
	PrivilegedVerifyByteSignature(dat []byte, sig []byte, identity string) bool

	// PrivilegedVerifyStringSignature verifies that dat is a valid signature for identity
	// or any of the privileged certificates
	PrivilegedVerifyStringSignature(dat string, sig []byte, identity string) bool

	// ChecksumBytes produce a crypto checksum for data
	ChecksumBytes(data []byte) []byte

	// ChecksumString produce a crypto checksum for data
	ChecksumString(data string) []byte

	// TLSConfig produce a tls.Config for the current identity using it's certificates etc
	TLSConfig() (*tls.Config, error)

	// SSLContext produce a http.Transport for the current identity using it's certificates etc
	SSLContext() (*http.Transport, error)

	// HTTPClient creates a standard HTTP client with optional security, it will
	// be set to use the CA and client certs for auth.
	HTTPClient(secure bool) (*http.Client, error)

	// VerifyCertificate validates that a certificate is signed by a known CA
	VerifyCertificate(certpem []byte, identity string) error

	// PublicCertPem retrieves pem data for the public certificate of the current identity
	PublicCertPem() (*pem.Block, error)

	// PublicCertTXT retrieves pem data in textual form for the public certificate of the current identity
	PublicCertTXT() ([]byte, error)

	// CachePublicData when given a pem encoded certificate and expected identity should validate
	// the cert and then check against things like the certificate allow lists, privilege lists
	// etc and only cache certificates that is completely acceptable by us
	CachePublicData(data []byte, identity string) error

	// CachedPublicData retrieves a previously cached certificate
	CachedPublicData(identity string) ([]byte, error)

	// Enroll creates a new cert with the active identity and attempt to enroll it with the security system
	// if there's a process of waiting for the certificate to be signed for example this should wait
	// no more than wait.  cb gets called on every attempt to download a cert with the attempt number
	// as argument
	Enroll(ctx context.Context, wait time.Duration, cb func(int)) error
}
