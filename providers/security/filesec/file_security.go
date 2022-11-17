// Copyright (c) 2020-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

// Package filesec provides a manually configurable security Provider
// it allows you set every parameter like key paths etc manually without
// making any assumptions about your system
//
// It does not support any enrollment
package filesec

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/internal/util"
	"github.com/choria-io/go-choria/tlssetup"

	"github.com/sirupsen/logrus"
)

// used by tests to stub out uids etc, should probably be a class and use dependency injection, meh
var (
	useFakeUID = false
	fakeUID    = 0
	useFakeOS  = false
	fakeOS     = "fake"
	callerIDRe = regexp.MustCompile(`^[a-z]+=([\w\.\-]+)`)
)

// FileSecurity implements SecurityProvider using files on disk
type FileSecurity struct {
	conf *Config
	log  *logrus.Entry

	mu *sync.Mutex
}

// Config is the configuration for FileSecurity
type Config struct {
	// Identity when not empty will force the identity to be used for validations etc
	Identity string

	// Certificate is the path to the public certificate
	Certificate string

	// Key is the path to the private key
	Key string

	// CA is the path to the Certificate Authority
	CA string

	// PrivilegedUsers is a list of regular expressions that identity privileged users
	PrivilegedUsers []string

	// AllowList is a list of regular expressions that identity valid users to allow in
	AllowList []string

	// DisableTLSVerify disables TLS verify in HTTP clients etc
	DisableTLSVerify bool

	// Is a URL where a remote signer is running
	RemoteSignerURL string

	// RemoteSignerTokenFile is a file with a token for access to the remote signer
	RemoteSignerTokenFile string

	// RemoteSignerSeedFile is a file with a seed related to RemoteSignerTokenFile
	RemoteSignerSeedFile string

	// TLSSetup is the shared TLS configuration state between security providers
	TLSConfig *tlssetup.Config

	// BackwardCompatVerification enables custom verification that allows legacy certificates without SANs
	BackwardCompatVerification bool

	// IdentitySuffix is the suffix to append to usernames when creating certnames and identities
	IdentitySuffix string

	// RemoteSigner is the signer used to sign requests using a remote like AAA Service
	RemoteSigner inter.RequestSigner
}

// New creates a new instance of the File Security provider
func New(opts ...Option) (*FileSecurity, error) {
	f := &FileSecurity{
		mu: &sync.Mutex{},
	}

	for _, opt := range opts {
		err := opt(f)
		if err != nil {
			return nil, err
		}
	}

	if f.conf == nil {
		return nil, errors.New("configuration not given")
	}

	if f.log == nil {
		return nil, errors.New("logger not given")
	}

	if f.conf.Identity == "" {
		return nil, errors.New("identity could not be determine automatically via Choria or was not supplied")
	}

	if f.conf.BackwardCompatVerification {
		f.log.Infof("Enabling support for legacy SAN free certificates")
	}

	return f, nil
}

func (s *FileSecurity) BackingTechnology() inter.SecurityTechnology {
	return inter.SecurityTechnologyX509
}

// Provider reports the name of the security provider
func (s *FileSecurity) Provider() string {
	return "file"
}

func (s *FileSecurity) TokenBytes() ([]byte, error) {
	return nil, fmt.Errorf("tokens not available for file security provider")
}

func (s *FileSecurity) RemoteSignerSeedFile() (string, error) {
	if s.conf.RemoteSignerTokenFile != "" && s.conf.RemoteSignerSeedFile == "" {
		// copies the behavior from framework SignerSeedFile()
		s.conf.RemoteSignerSeedFile = fmt.Sprintf("%s.key", strings.TrimSuffix(s.conf.RemoteSignerTokenFile, filepath.Ext(s.conf.RemoteSignerTokenFile)))
	}

	if s.conf.RemoteSignerSeedFile == "" {
		return "", fmt.Errorf("no seed file defined")
	}

	return s.conf.RemoteSignerSeedFile, nil
}

func (s *FileSecurity) RemoteSignerToken() ([]byte, error) {
	if s.conf.RemoteSignerTokenFile == "" {
		return nil, fmt.Errorf("no token file defined")
	}

	tb, err := os.ReadFile(s.conf.RemoteSignerTokenFile)
	if err != nil {
		return bytes.TrimSpace(tb), fmt.Errorf("could not read token file: %v", err)
	}

	return tb, err
}

func (s *FileSecurity) RemoteSignerURL() (*url.URL, error) {
	if s.conf.RemoteSignerURL == "" {
		return nil, fmt.Errorf("no remote url configured")
	}

	return url.Parse(s.conf.RemoteSignerURL)
}

// RemoteSignRequest signs a choria request using a remote signer and returns a secure request
func (s *FileSecurity) RemoteSignRequest(ctx context.Context, request []byte) (signed []byte, err error) {
	if s.conf.RemoteSigner == nil {
		return nil, fmt.Errorf("remote signing not configured")
	}

	s.log.Infof("Signing request using %s", s.conf.RemoteSigner.Kind())
	return s.conf.RemoteSigner.Sign(ctx, request, s)
}

// Validate determines if the node represents a valid SSL configuration
func (s *FileSecurity) Validate() ([]string, bool) {
	var errors []string

	if s.publicCertPath() != "" {
		if !s.publicCertExists() {
			errors = append(errors, fmt.Sprintf("public certificate %s does not exist", s.publicCertPath()))
		}
	} else {
		errors = append(errors, "the public certificate path is not configured")
	}

	if s.privateKeyPath() != "" {
		if !s.privateKeyExists() {
			errors = append(errors, fmt.Sprintf("private key %s does not exist", s.privateKeyPath()))
		}
	} else {
		errors = append(errors, "the private key path is not configured")
	}

	if s.caPath() != "" {
		if !s.caExists() {
			errors = append(errors, fmt.Sprintf("CA %s does not exist", s.caPath()))
		}
	} else {
		errors = append(errors, "the CA path is not configured")
	}

	return errors, len(errors) == 0
}

// ChecksumBytes calculates a sha256 checksum for data
func (s *FileSecurity) ChecksumBytes(data []byte) []byte {
	sum := sha256.Sum256(data)

	return sum[:]
}

// SignBytes signs a message using a SHA256 PKCS1v15 protocol
func (s *FileSecurity) SignBytes(str []byte) ([]byte, error) {
	sig := []byte{}

	pkpem, err := s.privateKeyPEM()
	if err != nil {
		return sig, err
	}
	var parsedKey any

	parsedKey, err = x509.ParsePKCS1PrivateKey(pkpem.Bytes)
	if err != nil {
		parsedKey, err = x509.ParsePKCS8PrivateKey(pkpem.Bytes)
		if err != nil {
			err = fmt.Errorf("could not parse private key PEM data: %s", err)
			return sig, err
		}
	}

	rng := rand.Reader
	hashed := s.ChecksumBytes(str)

	switch t := parsedKey.(type) {
	case *rsa.PrivateKey:
		sig, err = rsa.SignPKCS1v15(rng, t, crypto.SHA256, hashed[:])
	default:
		return sig, fmt.Errorf("unhandled key type %T", t)
	}

	if err != nil {
		err = fmt.Errorf("could not sign message: %s", err)
	}

	return sig, err
}

// VerifyByteSignature verify that dat matches signature sig made by the key, if pub cert is empty the active public key will be used
func (s *FileSecurity) VerifySignatureBytes(dat []byte, sig []byte, public ...[]byte) (should bool, signer string) {
	if len(public) != 1 {
		s.log.Errorf("Could not process public data: only single signer public data is supported")
		return false, ""
	}

	pubcert := public[0]

	var err error

	if len(pubcert) == 0 {
		pubcert, err = s.PublicCertBytes()
		if err != nil {
			s.log.Errorf("Could not load public cert: %v", err)
			return false, ""
		}
	}

	pkpem, _ := pem.Decode(pubcert)
	if pkpem == nil {
		s.log.Errorf("Could not decode PEM data in public key: invalid pem data")
		return false, ""
	}

	cert, err := x509.ParseCertificate(pkpem.Bytes)
	if err != nil {
		s.log.Errorf("Could not parse decoded PEM data for public certificate: %s", err)
		return false, ""
	}

	rsaPublicKey := cert.PublicKey.(*rsa.PublicKey)
	hashed := s.ChecksumBytes(dat)

	err = rsa.VerifyPKCS1v15(rsaPublicKey, crypto.SHA256, hashed[:], sig)
	if err != nil {
		s.log.Errorf("Signature verification failed: %s", err)
		return false, ""
	}

	names := []string{cert.Subject.CommonName}
	names = append(names, cert.DNSNames...)

	if len(names) == 0 {
		s.log.Errorf("Signature verification failed: no names found in signer certificate")
		return false, ""
	}

	s.log.Debugf("Verified signature from %s", strings.Join(names, ", "))

	return true, names[0]
}

// CallerName creates a choria like caller name in the form of choria=identity
func (s *FileSecurity) CallerName() string {
	return fmt.Sprintf("choria=%s", s.Identity())
}

// CallerIdentity extracts the identity from a choria like caller name in the form of choria=identity
func (s *FileSecurity) CallerIdentity(caller string) (string, error) {
	match := callerIDRe.FindStringSubmatch(caller)

	if match == nil {
		return "", fmt.Errorf("could not find a valid caller identity name in %s", caller)
	}

	return match[1], nil
}

// IsRemoteSigning determines if remote signer is set
func (s *FileSecurity) IsRemoteSigning() bool {
	return s.conf.RemoteSigner != nil
}

// Identity determines the choria certname
func (s *FileSecurity) Identity() string {
	return s.conf.Identity
}

// VerifyCertificate verifies a certificate is signed with the configured CA and if
// name is not "" that it matches the name given
func (s *FileSecurity) VerifyCertificate(certpem []byte, name string) error {
	ca := s.caPath()
	capem, err := os.ReadFile(ca)
	if err != nil {
		s.log.Errorf("Could not read CA '%s': %s", ca, err)
		return err
	}

	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(capem) {
		s.log.Warnf("Could not use CA '%s' as PEM data: %s", ca, err)
		return err
	}

	block, _ := pem.Decode(certpem)
	if block == nil {
		s.log.Warnf("Could not decode certificate '%s' PEM data: %s", name, err)
		return err
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		s.log.Warnf("Could not parse certificate '%s': %s", name, err)
		return err
	}

	intermediates := x509.NewCertPool()
	if !intermediates.AppendCertsFromPEM(certpem) {
		s.log.Warnf("Could not add intermediates: %s", err)
		return err
	}

	opts := x509.VerifyOptions{
		Roots:         roots,
		Intermediates: intermediates,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	_, err = cert.Verify(opts)
	if err != nil {
		s.log.Warnf("Certificate does not pass verification as '%s': %s", name, err)
		return err
	}

	if len(cert.EmailAddresses) > 0 && strings.HasPrefix(name, "email:") {
		s.log.Debug("Email addresses found in certificate, attempting verification")
		for _, email := range cert.EmailAddresses {
			if strings.TrimPrefix(name, "email:") == email {
				return nil
			}
		}

		return fmt.Errorf("email address not found in SAN: %s, %v", name, cert.EmailAddresses)
	}

	// ShouldAllowCaller passes in an empty name, we just want it to verify validity of the CA chain at this point
	if name == "" {
		return nil
	}

	if !findName(cert.DNSNames, name) {
		if cert.Subject.CommonName != name {
			return fmt.Errorf("x509: certificate is valid for %s, not %s", cert.Subject.CommonName, name)
		}
	}

	return nil
}

func findName(names []string, name string) bool {
	for _, n := range names {
		if n == name {
			return true
		}
	}
	return false
}

// HTTPClient creates a standard HTTP client with optional security, it will
// be set to use the CA and client certs for auth. servername should match the
// remote hosts name for SNI
func (s *FileSecurity) HTTPClient(secure bool) (*http.Client, error) {
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

func (s *FileSecurity) ClientTLSConfig() (*tls.Config, error) {
	tlsc, err := s.TLSConfig()
	if err != nil {
		return nil, err
	}

	if s.conf.BackwardCompatVerification {
		tlsc.InsecureSkipVerify = true
		tlsc.VerifyConnection = s.constructCustomVerifier(tlsc.RootCAs)
	}

	return tlsc, nil
}

// TLSConfig creates a TLS configuration for use by NATS, HTTPS etc
func (s *FileSecurity) TLSConfig() (*tls.Config, error) {
	pub := s.publicCertPath()
	pri := s.privateKeyPath()
	ca := s.caPath()

	tlsc := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		PreferServerCipherSuites: true,
		CipherSuites:             s.conf.TLSConfig.CipherSuites,
		CurvePreferences:         s.conf.TLSConfig.CurvePreferences,
	}

	if s.privateKeyExists() && s.publicCertExists() {
		cert, err := tls.LoadX509KeyPair(pub, pri)
		if err != nil {
			err = fmt.Errorf("could not load certificate %s and key %s: %s", pub, pri, err)
			return nil, err
		}

		cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
		if err != nil {
			err = fmt.Errorf("error parsing certificate: %v", err)
			return nil, err
		}

		tlsc.Certificates = []tls.Certificate{cert}
	}

	if s.caExists() {
		caCert, err := os.ReadFile(ca)
		if err != nil {
			return nil, err
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		tlsc.ClientCAs = caCertPool
		tlsc.RootCAs = caCertPool
	}

	if s.conf.DisableTLSVerify {
		tlsc.InsecureSkipVerify = true
	}

	return tlsc, nil
}

func (s *FileSecurity) constructCustomVerifier(pool *x509.CertPool) func(cs tls.ConnectionState) error {
	return func(cs tls.ConnectionState) error {
		s.log.Debug("Verifying connection using legacy SAN free certificate support")
		opts := x509.VerifyOptions{
			Roots:         pool,
			Intermediates: x509.NewCertPool(),
		}
		// If there is no SAN, then fallback to using the CommonName
		hasSanExtension := func(cert *x509.Certificate) bool {
			// oid taken from crypt/x509/x509.go
			var oidExtensionSubjectAltName = []int{2, 5, 29, 17}
			for _, e := range cert.Extensions {
				if e.Id.Equal(oidExtensionSubjectAltName) {
					return true
				}
			}
			return false
		}
		if !hasSanExtension(cs.PeerCertificates[0]) {
			if !strings.EqualFold(cs.ServerName, cs.PeerCertificates[0].Subject.CommonName) {
				return x509.HostnameError{Certificate: cs.PeerCertificates[0], Host: cs.ServerName}
			}
		} else {
			opts.DNSName = cs.ServerName
		}
		for _, cert := range cs.PeerCertificates[1:] {
			opts.Intermediates.AddCert(cert)
		}
		_, err := cs.PeerCertificates[0].Verify(opts)
		return err
	}
}

// publicCertPem retrieves the public certificate for this instance
func (s *FileSecurity) publicCertPem() (*pem.Block, error) {
	path := s.publicCertPath()

	return s.decodePEM(path)
}

// PublicCertBytes retrieves pem data in textual form for the public certificate of the current identity
func (s *FileSecurity) PublicCertBytes() ([]byte, error) {
	path := s.publicCertPath()

	return os.ReadFile(path)
}

// PublicCert is the parsed public certificate
func (s *FileSecurity) PublicCert() (*x509.Certificate, error) {
	block, err := s.publicCertPem()
	if err != nil {
		return nil, err
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}

	return cert, nil
}

// SSLContext creates a SSL context loaded with our certs and ca
func (s *FileSecurity) SSLContext() (*http.Transport, error) {
	tlsConfig, err := s.ClientTLSConfig()
	if err != nil {
		return nil, err
	}

	transport := &http.Transport{TLSClientConfig: tlsConfig}

	return transport, nil
}

// Enroll is not supported
func (s *FileSecurity) Enroll(ctx context.Context, wait time.Duration, cb func(digest string, try int)) error {
	return errors.New("the file security provider does not support enrollment")
}

func (s *FileSecurity) decodePEM(certpath string) (*pem.Block, error) {
	var err error

	if certpath == "" {
		return nil, errors.New("invalid certpath '' provided")
	}

	keydat, err := os.ReadFile(certpath)
	if err != nil {
		return nil, fmt.Errorf("could not read PEM data from %s: %s", certpath, err)
	}

	pb, _ := pem.Decode(keydat)
	if pb == nil {
		return nil, fmt.Errorf("failed to parse PEM data from key %s", certpath)
	}

	return pb, nil
}

func (s *FileSecurity) privateKeyPath() string {
	return filepath.FromSlash(s.conf.Key)
}

func (s *FileSecurity) publicCertPath() string {
	return filepath.FromSlash(s.conf.Certificate)
}

func (s *FileSecurity) caPath() string {
	return filepath.FromSlash(s.conf.CA)
}

func (s *FileSecurity) privateKeyExists() bool {
	return util.FileExist(s.privateKeyPath())
}

func (s *FileSecurity) publicCertExists() bool {
	return util.FileExist(s.publicCertPath())
}

func (s *FileSecurity) caExists() bool {
	return util.FileExist(s.caPath())
}

func (s *FileSecurity) privateKeyPEM() (pb *pem.Block, err error) {
	key := s.privateKeyPath()

	keydat, err := os.ReadFile(key)
	if err != nil {
		return pb, fmt.Errorf("could not read Private Key %s: %s", key, err)
	}

	pb, _ = pem.Decode(keydat)
	if pb == nil {
		return pb, fmt.Errorf("failed to parse PEM data from key %s", key)
	}

	return
}

func (s *FileSecurity) ShouldAllowCaller(name string, callers ...[]byte) (privileged bool, err error) {
	if len(callers) != 1 {
		s.log.Warnf("Received multiple items of caller identity data in x509 security provider")
		return false, fmt.Errorf("invalid public data")
	}

	data := callers[0]

	privNames, err := s.certDNSNames(data)
	if err != nil {
		s.log.Warnf("Could not extract DNS Names from certificate")
		return false, err
	}

	for _, privName := range privNames {
		if MatchAnyRegex(privName, s.conf.PrivilegedUsers) {
			privileged = true
			break
		}
	}

	if privileged {
		// Checks if it was signed by a CA issued cert but without any name validation since privileged name wouldnt match
		err = s.VerifyCertificate(data, "")
		if err != nil {
			s.log.Warnf("Received certificate '%s' certificate did not pass verification: %s", name, err)
			return false, err
		}

		return true, nil
	} else {
		// Checks if it was signed by a CA issued cert that matches name since it must match when not privileged
		err = s.VerifyCertificate(data, name)
		if err != nil {
			s.log.Warnf("Received certificate '%s' did not pass verification: %s", name, err)
			return false, err
		}
	}

	// Finally if its on the allow list
	if MatchAnyRegex(name, s.conf.AllowList) {
		return false, nil
	}

	s.log.Warnf("Received certificate '%s' does not match the allowed list '%s'", name, s.conf.AllowList)

	return false, fmt.Errorf("not on allow list")
}

func (s *FileSecurity) certDNSNames(certpem []byte) (names []string, err error) {
	block, _ := pem.Decode(certpem)
	if block == nil {
		s.log.Warnf("Could not decode certificate PEM data: %s", err)
		return names, err
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		s.log.Warnf("Could not parse certificate: %s", err)
		return names, err
	}

	names = append(names, cert.Subject.CommonName)
	names = append(names, cert.DNSNames...)

	return names, nil
}

func (s *FileSecurity) ShouldSignReplies() bool { return false }
