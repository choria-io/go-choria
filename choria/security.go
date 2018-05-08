package choria

import (
	"crypto/tls"
	"encoding/pem"
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"
)

// SecurityProvider provides a security plugin for the choria subsystem
type SecurityProvider interface {
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

	// VerifyCertificate validates that a certificate is signed by a known CA
	VerifyCertificate(certpem []byte, identity string) (error, bool)

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
}

// NewSecurityProvider creates a new instance of a security provider
func NewSecurityProvider(provider string, fw *Framework, log *logrus.Entry) (SecurityProvider, error) {
	switch provider {
	case "puppet":
		return NewPuppetSecurity(fw, fw.Config, log)
	}

	return nil, fmt.Errorf("unknown security provider: %s", provider)
}
