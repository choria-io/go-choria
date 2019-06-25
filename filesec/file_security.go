// Package filesec provides a manually configurable security Provider
// it allows you set every paramter like key paths etc manually without
// making any assumptions about your system
//
// It does not support any enrollment
package filesec

import (
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
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// used by tests to stub out uids etc, should probably be a class and use dependency injection, meh
var useFakeUID = false
var fakeUID = 0
var useFakeOS = false
var fakeOS = "fake"

// FileSecurity impliments SecurityProvider using files on disk
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

	// Cache is where known client certificates will be stored
	Cache string

	// PrivilegedUsers is a list of regular expressions that identity privilged users
	PrivilegedUsers []string

	// AllowList is a list of regular expressions that identity valid users to allow in
	AllowList []string

	// DisableTLSVerify disables TLS verify in HTTP clients etc
	DisableTLSVerify bool

	// AlwaysOverwriteCache supports always overwriting the local filesystem cache
	AlwaysOverwriteCache bool
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

	return f, nil
}

// Provider reports the name of the security provider
func (s *FileSecurity) Provider() string {
	return "file"
}

// Validate determines if the node represents a valid SSL configuration
func (s *FileSecurity) Validate() ([]string, bool) {
	errors := []string{}

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

// ChecksumString calculates a sha256 checksum for data
func (s *FileSecurity) ChecksumString(data string) []byte {
	return s.ChecksumBytes([]byte(data))
}

// SignBytes signs a message using a SHA256 PKCS1v15 protocol
func (s *FileSecurity) SignBytes(str []byte) ([]byte, error) {
	sig := []byte{}

	pkpem, err := s.privateKeyPEM()
	if err != nil {
		return sig, err
	}

	pk, err := x509.ParsePKCS1PrivateKey(pkpem.Bytes)
	if err != nil {
		err = fmt.Errorf("could not parse private key PEM data: %s", err)
		return sig, err
	}

	rng := rand.Reader
	hashed := s.ChecksumBytes(str)
	sig, err = rsa.SignPKCS1v15(rng, pk, crypto.SHA256, hashed[:])
	if err != nil {
		err = fmt.Errorf("could not sign message: %s", err)
	}

	return sig, err
}

// VerifyByteSignature verify that dat matches signature sig made by the key of identity
// if identity is "" the active public key will be used
func (s *FileSecurity) VerifyByteSignature(dat []byte, sig []byte, identity string) bool {
	pubkeyPath := ""
	var err error

	pubkeyPath = s.publicCertPath()

	if identity != "" {
		pubkeyPath, err = s.cachePath(identity)
	}

	s.log.Debugf("Attempting to verify signature for %s using %s", identity, pubkeyPath)

	pkpem, err := s.decodePEM(pubkeyPath)
	if err != nil {
		s.log.Errorf("Could not decode PEM data in public key %s: %s", pubkeyPath, err)
		return false
	}

	cert, err := x509.ParseCertificate(pkpem.Bytes)
	if err != nil {
		s.log.Errorf("Could not parse decoded PEM data for public key %s: %s", pubkeyPath, err)
		return false
	}

	rsaPublicKey := cert.PublicKey.(*rsa.PublicKey)
	hashed := s.ChecksumBytes(dat)

	err = rsa.VerifyPKCS1v15(rsaPublicKey, crypto.SHA256, hashed[:], sig)
	if err != nil {
		s.log.Errorf("Signature verification using %s failed: %s", pubkeyPath, err)
		return false
	}

	s.log.Debugf("Verified signature from %s using %s", identity, pubkeyPath)
	return true
}

// VerifyStringSignature verify that str matches signature sig made by the key of identity
func (s *FileSecurity) VerifyStringSignature(str string, sig []byte, identity string) bool {
	return s.VerifyByteSignature([]byte(str), sig, identity)
}

// PrivilegedVerifyByteSignature verifies if the signature received is from any of the privileged certs or the given identity
func (s *FileSecurity) PrivilegedVerifyByteSignature(dat []byte, sig []byte, identity string) bool {
	var candidates []string

	if identity != "" && s.cachedCertExists(identity) {
		candidates = append(candidates, identity)
	}

	for _, candidate := range s.privilegedCerts() {
		candidates = append(candidates, candidate)
	}

	for _, candidate := range candidates {
		if s.VerifyByteSignature(dat, sig, candidate) {
			s.log.Debugf("Allowing certificate %s to act as %s", candidate, identity)
			return true
		}
	}

	return false
}

// PrivilegedVerifyStringSignature verifies if the signature received is from any of the privileged certs or the given identity
func (s *FileSecurity) PrivilegedVerifyStringSignature(dat string, sig []byte, identity string) bool {
	return s.PrivilegedVerifyByteSignature([]byte(dat), sig, identity)
}

// SignString signs a message using a SHA256 PKCS1v15 protocol
func (s *FileSecurity) SignString(str string) ([]byte, error) {
	return s.SignBytes([]byte(str))
}

// CallerName creates a choria like caller name in the form of choria=identity
func (s *FileSecurity) CallerName() string {
	return fmt.Sprintf("choria=%s", s.Identity())
}

// CallerIdentity extracts the identity from a choria like caller name in the form of choria=identity
func (s *FileSecurity) CallerIdentity(caller string) (string, error) {
	re := regexp.MustCompile("^[a-z]+=([\\w\\.\\-]+)")
	match := re.FindStringSubmatch(caller)

	if match == nil {
		return "", fmt.Errorf("could not find a valid caller identity name in %s", caller)
	}

	return match[1], nil
}

// CachePublicData caches the public key for a identity
func (s *FileSecurity) CachePublicData(data []byte, identity string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	should, privileged, identity := s.shouldCacheClientCert(data, identity)
	if !should {
		return fmt.Errorf("certificate '%s' did not pass validation", identity)
	}

	err := os.MkdirAll(s.certCacheDir(), os.FileMode(int(0755)))
	if err != nil {
		return fmt.Errorf("could not create Client Certificate Cache Directory: %s", err)
	}

	certfile, err := s.cachePath(identity)
	if err != nil {
		return err
	}

	_, err = os.Stat(certfile)
	if err == nil {
		if !s.conf.AlwaysOverwriteCache {
			s.log.Debugf("Already have a certificate in %s, refusing to overwrite with a new one", certfile)
			return nil
		}

		// it exists, lets check if its required to update it, quicker to just update it but that
		// risks failing when disks are full etc this attempts that risky step only when needed
		rsum := sha256.Sum256([]byte(data))
		fsum, err := fsha256(certfile)
		if err != nil {
			return fmt.Errorf("could not determine sha256 of current certificate in %s: %s", certfile, err)
		}

		if fmt.Sprintf("%x", fsum) == fmt.Sprintf("%x", rsum) {
			s.log.Debugf("Received certificate is the same as cached certificate %s, not updating cache", certfile)
			return nil
		}
	}

	err = ioutil.WriteFile(certfile, []byte(data), os.FileMode(int(0644)))
	if err != nil {
		return fmt.Errorf("could not cache client public certificate: %s", err.Error())
	}

	if privileged {
		s.log.Warnf("Cached privileged certificate %s for %s", certfile, identity)
	} else {
		s.log.Infof("Cached certificate %s for %s", certfile, identity)
	}

	return nil
}

// CachedPublicData retrieves the previously cached public data for a given identity
func (s *FileSecurity) CachedPublicData(identity string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	certfile, err := s.cachePath(identity)
	if err != nil {
		return []byte{}, fmt.Errorf("could not cache public data: %s", err)
	}

	if _, err := os.Stat(certfile); os.IsNotExist(err) {
		return []byte{}, fmt.Errorf("unknown public data: %s", identity)
	}

	return ioutil.ReadFile(certfile)
}

// Identity determines the choria certname
func (s *FileSecurity) Identity() string {
	return s.conf.Identity
}

func (s *FileSecurity) privilegedCerts() []string {
	certs := []string{}

	filepath.Walk(s.certCacheDir(), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			cert := []byte(strings.TrimSuffix(filepath.Base(path), ".pem"))

			if s.isPrivilegedCert(cert) {
				certs = append(certs, string(cert))
			}
		}

		return nil
	})

	sort.Strings(certs)

	return certs
}

func (s *FileSecurity) isPrivilegedCert(cert []byte) bool {
	return MatchAnyRegex(cert, s.conf.PrivilegedUsers)
}

// VerifyCertificate verifies a certificate is signed with the configured CA and if
// name is not "" that it matches the name given
func (s *FileSecurity) VerifyCertificate(certpem []byte, name string) error {
	ca := s.caPath()
	capem, err := ioutil.ReadFile(ca)
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

	// If there is an email address in the name passed, we should not search by DNSName
	// in the CN or SAN
	if name != "" && !strings.HasPrefix(name, "email:") {
		opts.DNSName = name
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

	return nil
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

// TLSConfig creates a TLS configuration for use by NATS, HTTPS etc
func (s *FileSecurity) TLSConfig() (*tls.Config, error) {
	pub := s.publicCertPath()
	pri := s.privateKeyPath()
	ca := s.caPath()

	tlsc := &tls.Config{
		MinVersion: tls.VersionTLS12,
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
		caCert, err := ioutil.ReadFile(ca)
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

	tlsc.BuildNameToCertificate()

	return tlsc, nil
}

// PublicCertPem retrieves the public certificate for this instance
func (s *FileSecurity) PublicCertPem() (*pem.Block, error) {
	path := s.publicCertPath()

	return s.decodePEM(path)
}

// PublicCertTXT retrieves pem data in textual form for the public certificate of the current identity
func (s *FileSecurity) PublicCertTXT() ([]byte, error) {
	path := s.publicCertPath()

	return ioutil.ReadFile(path)
}

// SSLContext creates a SSL context loaded with our certs and ca
func (s *FileSecurity) SSLContext() (*http.Transport, error) {
	tlsConfig, err := s.TLSConfig()
	if err != nil {
		return nil, err
	}

	transport := &http.Transport{TLSClientConfig: tlsConfig}

	return transport, nil
}

// Enroll is not supported
func (s *FileSecurity) Enroll(ctx context.Context, wait time.Duration, cb func(int)) error {
	return errors.New("The file security provider does not support enrollement")
}

func (s *FileSecurity) cachePath(identity string) (string, error) {
	certfile := filepath.Join(s.certCacheDir(), fmt.Sprintf("%s.pem", identity))

	return certfile, nil
}

func (s *FileSecurity) decodePEM(certpath string) (*pem.Block, error) {
	var err error

	if certpath == "" {
		return nil, errors.New("invalid certpath '' provided")
	}

	keydat, err := ioutil.ReadFile(certpath)
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
	_, err := os.Stat(s.privateKeyPath())

	return !os.IsNotExist(err)
}

func (s *FileSecurity) publicCertExists() bool {
	_, err := os.Stat(s.publicCertPath())

	return !os.IsNotExist(err)
}

func (s *FileSecurity) caExists() bool {
	_, err := os.Stat(s.caPath())

	return !os.IsNotExist(err)
}

func (s *FileSecurity) cachedCertExists(identity string) bool {
	f, err := s.cachePath(identity)
	if err != nil {
		return false
	}

	_, err = os.Stat(f)

	return !os.IsNotExist(err)
}

func (s *FileSecurity) privateKeyPEM() (pb *pem.Block, err error) {
	key := s.privateKeyPath()

	keydat, err := ioutil.ReadFile(key)
	if err != nil {
		return pb, fmt.Errorf("Could not read Private Key %s: %s", key, err)
	}

	pb, _ = pem.Decode(keydat)
	if pb == nil {
		return pb, fmt.Errorf("Failed to parse PEM data from key %s", key)
	}

	return
}

func (s *FileSecurity) certCacheDir() string {
	return filepath.FromSlash(s.conf.Cache)
}

// shouldCacheClientCert figure out if we should cache this cert and if we do by what name, we do
// not want certificate for caller bob which is in fact signed by a privilged cert to end up cached
// as bob, we so we determine the right name to use and pass that along back to the caller who then
// use that to determine the cache path
func (s *FileSecurity) shouldCacheClientCert(data []byte, name string) (should bool, privileged bool, savename string) {
	// Checks if it was signed by the CA but without any name validation
	if err := s.VerifyCertificate(data, ""); err != nil {
		s.log.Warnf("Received certificate '%s' certificate did not pass verification: %s", name, err)
		return false, false, name
	}

	// Check if the certificate that would be validated is a privileged one, so we don't name validate that
	// we already know its signed by the right CA so we accept the privileged ones.
	//
	// At this point name is from the caller id but we need what is in the presented certificate
	// in order to validate since the priv'd cert can overide name to something else, so we extract
	// the common name and all the dnsnames and check each one, if any of them are a privileged user
	// we can go ahead with that one
	privNames, err := s.certDNSNames(data)
	if err != nil {
		s.log.Warnf("Could not extract DNS Names from certificate")
		return false, false, name
	}

	for _, privName := range privNames {
		if MatchAnyRegex([]byte(privName), s.conf.PrivilegedUsers) {
			return true, true, privName
		}
	}

	// At this point we know ifs not privileged so we verify again but this time also check the name matches what
	// is in the cert since at this point it must match the caller id name
	if err := s.VerifyCertificate(data, name); err != nil {
		s.log.Warnf("Received certificate '%s' did not pass verification: %s", name, err)
		return false, false, name
	}

	// Finally if its on the allow list
	if MatchAnyRegex([]byte(name), s.conf.AllowList) {
		return true, false, name
	}

	s.log.Warnf("Received certificate '%s' does not match the allowed list '%s'", name, s.conf.AllowList)
	return false, false, name
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

	for _, name := range cert.DNSNames {
		names = append(names, name)
	}

	return names, nil
}
