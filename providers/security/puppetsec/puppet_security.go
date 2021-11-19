// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

// Package puppetsec provides a Puppet compatable Security Provider
//
// The provider supports enrolling into a Puppet CA by creating a
// key and csr, sending it to the PuppetCA and waiting for it to
// be signed and later it will download the certificate once signed
package puppetsec

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/internal/util"
	"github.com/choria-io/go-choria/tlssetup"

	"github.com/choria-io/go-choria/providers/security/filesec"
	"github.com/choria-io/go-choria/srvcache"
	"github.com/sirupsen/logrus"
)

// Resolver provides DNS lookup facilities
type Resolver interface {
	QuerySrvRecords(records []string) (srvcache.Servers, error)
}

// BuildInfoProvider provides info about the build
type BuildInfoProvider interface {
	ClientIdentitySuffix() string
}

// PuppetSecurity implements SecurityProvider reusing AIO Puppet settings
// it supports enrollment the same way `puppet agent --waitforcert 10` does
type PuppetSecurity struct {
	res  Resolver
	conf *Config
	log  *logrus.Entry

	fsec  *filesec.FileSecurity
	cache string
}

// Config is the configuration for PuppetSecurity
type Config struct {
	// Identity when not empty will force the identity to be used for validations etc
	Identity string

	// SSLDir is the directory where Puppet stores it's SSL
	SSLDir string

	// PrivilegedUsers is a list of regular expressions that identity privilged users
	PrivilegedUsers []string

	// AllowList is a list of regular expressions that identity valid users to allow in
	AllowList []string

	// DisableTLSVerify disables TLS verify in HTTP clients etc
	DisableTLSVerify bool

	// PuppetCAHost is the hostname of the PuppetCA
	PuppetCAHost string

	// PuppetCAPort is the port of the PuppetCA
	PuppetCAPort int

	// DisableSRV prevents SRV lookups
	DisableSRV bool

	// Is a URL where a remote signer is running
	RemoteSignerURL string

	// RemoteSignerTokenFile is a file with a token for access to the remote signer
	RemoteSignerTokenFile string

	// RemoteSignerTokenEnvironment is an environment variable that will hold the signer token
	RemoteSignerTokenEnvironment string

	useFakeUID bool
	fakeUID    int

	// AlwaysOverwriteCache supports always overwriting the local filesystem cache
	AlwaysOverwriteCache bool

	// TLSConfig is the shared TLS configuration
	TLSConfig *tlssetup.Config

	// AltNames are additional names to add to the CSR
	AltNames []string

	// IdentitySuffix is the suffix to append to user names when creating certnames and identities
	IdentitySuffix string

	// RemoteSigner is the signer used to sign requests using a remote like AAA Service
	RemoteSigner inter.RequestSigner
}

// New creates a new instance of the Puppet Security Provider
func New(opts ...Option) (*PuppetSecurity, error) {
	p := &PuppetSecurity{}

	for _, opt := range opts {
		err := opt(p)
		if err != nil {
			return nil, err
		}
	}

	if p.conf == nil {
		return nil, errors.New("configuration not given")
	}

	if p.log == nil {
		return nil, errors.New("logger not given")
	}

	if p.conf.Identity == "" {
		return nil, errors.New("identity could not be determine automatically via Choria or was not supplied")
	}

	return p, p.reinit()
}

func (s *PuppetSecurity) reinit() error {
	var err error

	fc := filesec.Config{
		AllowList:                  s.conf.AllowList,
		DisableTLSVerify:           s.conf.DisableTLSVerify,
		PrivilegedUsers:            s.conf.PrivilegedUsers,
		CA:                         s.caPath(),
		Cache:                      s.certCacheDir(),
		Certificate:                s.publicCertPath(),
		Key:                        s.privateKeyPath(),
		Identity:                   s.conf.Identity,
		AlwaysOverwriteCache:       s.conf.AlwaysOverwriteCache,
		RemoteSignerURL:            s.conf.RemoteSignerURL,
		RemoteSignerTokenFile:      s.conf.RemoteSignerTokenFile,
		TLSConfig:                  s.conf.TLSConfig,
		IdentitySuffix:             s.conf.IdentitySuffix,
		BackwardCompatVerification: true,
		RemoteSigner:               s.conf.RemoteSigner,
	}

	s.fsec, err = filesec.New(filesec.WithConfig(&fc), filesec.WithLog(s.log))
	if err != nil {
		return err
	}

	if fc.BackwardCompatVerification {
		s.log.Debugf("Puppet security system requesting legacy TLS support")
	} else {
		s.log.Debugf("Puppet security system supporting only new certificates")
	}

	return nil
}

// Provider reports the name of the security provider
func (s *PuppetSecurity) Provider() string {
	return "puppet"
}

// Enroll sends a CSR to the PuppetCA and wait for it to be signed
func (s *PuppetSecurity) Enroll(ctx context.Context, wait time.Duration, cb func(digest string, try int)) error {
	if s.privateKeyExists() && s.caExists() && s.publicCertExists() {
		return errors.New("already have all files needed for SSL operations")
	}

	err := s.createSSLDirectories()
	if err != nil {
		return fmt.Errorf("could not initialize ssl directories: %s", err)
	}

	var key *rsa.PrivateKey

	if s.privateKeyExists() {
		s.log.Debugf("Loading existing private key for %s", s.Identity())
		key, err = s.readPrivateKey()
		if err != nil {
			return fmt.Errorf("could not read private key for %s: %s", s.Identity(), err)
		}
	} else {
		s.log.Debugf("Creating a new Private Key %s", s.Identity())

		key, err = s.writePrivateKey()
		if err != nil {
			return fmt.Errorf("could not write a new private key: %s", err)
		}
	}

	if !s.caExists() {
		s.log.Debug("Fetching CA")

		err = s.fetchCA()
		if err != nil {
			return fmt.Errorf("could not fetch CA: %s", err)
		}
	}

	previousCSR := s.csrExists()
	var digest string

	if !previousCSR {
		s.log.Debugf("Creating a new CSR for %s", s.Identity())

		digest, err = s.writeCSR(key, s.Identity(), "choria.io")
		if err != nil {
			return fmt.Errorf("could not write CSR: %s", err)
		}
	}

	if !s.publicCertExists() {
		s.log.Debug("Submitting CSR to the PuppetCA")

		err = s.submitCSR()
		if err != nil {
			if previousCSR {
				s.log.Warnf("Submitting CSR failed, ignoring failure as this might be a continuation of a previous attempts: %s", err)
			} else {
				return fmt.Errorf("could not submit csr: %s", err)
			}
		}
	}

	timeout := time.NewTimer(wait).C
	ticks := time.NewTicker(10 * time.Second).C

	complete := make(chan int, 2)

	attempt := 1

	fetcher := func() {
		cb(digest, attempt)
		attempt++

		err := s.fetchCert()
		if err != nil {
			s.log.Debugf("Error while fetching cert on attempt %d: %s", attempt-1, err)
			return
		}

		complete <- 1
	}

	fetcher()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("interrupted")
		case <-timeout:
			return fmt.Errorf("timed out waiting for a certificate")
		case <-complete:
			return nil
		case <-ticks:
			fetcher()
		}
	}
}

// Validate determines if the node represents a valid SSL configuration
func (s *PuppetSecurity) Validate() ([]string, bool) {
	errors := []string{}

	ferrs, _ := s.fsec.Validate()
	errors = append(errors, ferrs...)

	return errors, len(errors) == 0
}

// RemoteSignRequest signs a choria request using a remote signer and returns a secure request
func (s *PuppetSecurity) RemoteSignRequest(ctx context.Context, str []byte) (signed []byte, err error) {
	return s.fsec.RemoteSignRequest(ctx, str)
}

func (s *PuppetSecurity) IsRemoteSigning() bool {
	return s.fsec.IsRemoteSigning()
}

// ChecksumBytes calculates a sha256 checksum for data
func (s *PuppetSecurity) ChecksumBytes(data []byte) []byte {
	return s.fsec.ChecksumBytes(data)
}

// ChecksumString calculates a sha256 checksum for data
func (s *PuppetSecurity) ChecksumString(data string) []byte {
	return s.fsec.ChecksumBytes([]byte(data))
}

// SignBytes signs a message using a SHA256 PKCS1v15 protocol
func (s *PuppetSecurity) SignBytes(str []byte) ([]byte, error) {
	return s.fsec.SignBytes(str)
}

// VerifyByteSignature verify that dat matches signature sig made by the key of identity
// if identity is "" the active public key will be used
func (s *PuppetSecurity) VerifyByteSignature(dat []byte, sig []byte, identity string) bool {
	return s.fsec.VerifyByteSignature(dat, sig, identity)
}

// VerifyStringSignature verify that str matches signature sig made by the key of identity
func (s *PuppetSecurity) VerifyStringSignature(str string, sig []byte, identity string) bool {
	return s.VerifyByteSignature([]byte(str), sig, identity)
}

// PrivilegedVerifyByteSignature verifies if the signature received is from any of the privileged certs or the given identity
func (s *PuppetSecurity) PrivilegedVerifyByteSignature(dat []byte, sig []byte, identity string) bool {
	return s.fsec.PrivilegedVerifyByteSignature(dat, sig, identity)
}

// PrivilegedVerifyStringSignature verifies if the signature received is from any of the privilged certs or the given identity
func (s *PuppetSecurity) PrivilegedVerifyStringSignature(dat string, sig []byte, identity string) bool {
	return s.fsec.PrivilegedVerifyStringSignature(dat, sig, identity)
}

// SignString signs a message using a SHA256 PKCS1v15 protocol
func (s *PuppetSecurity) SignString(str string) ([]byte, error) {
	return s.fsec.SignString(str)
}

// CallerName creates a choria like caller name in the form of choria=identity
func (s *PuppetSecurity) CallerName() string {
	return s.fsec.CallerName()
}

// CallerIdentity extracts the identity from a choria like caller name in the form of choria=identity
func (s *PuppetSecurity) CallerIdentity(caller string) (string, error) {
	return s.fsec.CallerIdentity(caller)
}

// CachePublicData caches the public key for a identity
func (s *PuppetSecurity) CachePublicData(data []byte, identity string) error {
	return s.fsec.CachePublicData(data, identity)
}

// CachedPublicData retrieves the previously cached public data for a given identity
func (s *PuppetSecurity) CachedPublicData(identity string) ([]byte, error) {
	return s.fsec.CachedPublicData(identity)
}

func (s *PuppetSecurity) cachePath(identity string) string {
	var cache string

	cache = s.cache

	if cache == "" {
		cache = s.certCacheDir()
	}

	certfile := filepath.Join(cache, fmt.Sprintf("%s.pem", identity))

	return certfile
}

// VerifyCertificate verifies a certificate is signed with the configured CA and if
// name is not "" that it matches the name given
func (s *PuppetSecurity) VerifyCertificate(certpem []byte, name string) error {
	return s.fsec.VerifyCertificate(certpem, name)
}

// PublicCertPem retrieves the public certificate for this instance
func (s *PuppetSecurity) PublicCertPem() (*pem.Block, error) {
	return s.fsec.PublicCertPem()
}

// PublicCertTXT retrieves pem data in textual form for the public certificate of the current identity
func (s *PuppetSecurity) PublicCertTXT() ([]byte, error) {
	return s.fsec.PublicCertTXT()
}

// PublicCert is the parsed public certificate
func (s *PuppetSecurity) PublicCert() (*x509.Certificate, error) {
	return s.fsec.PublicCert()
}

// Identity determines the choria certname
func (s *PuppetSecurity) Identity() string {
	return s.conf.Identity
}

// TLSConfig creates a TLS configuration for use by NATS, HTTPS etc
func (s *PuppetSecurity) TLSConfig() (*tls.Config, error) {
	return s.fsec.TLSConfig()
}

// ClientTLSConfig creates a TLS configuration for use by NATS, HTTPS etc
func (s *PuppetSecurity) ClientTLSConfig() (*tls.Config, error) {
	return s.fsec.ClientTLSConfig()
}

// SSLContext creates a SSL context loaded with our certs and ca
func (s *PuppetSecurity) SSLContext() (*http.Transport, error) {
	return s.fsec.SSLContext()
}

func (s *PuppetSecurity) certCacheDir() string {
	return filepath.FromSlash(filepath.Join(s.sslDir(), "choria_security", "public_certs"))
}

func (s *PuppetSecurity) caPath() string {
	return filepath.FromSlash((filepath.Join(s.sslDir(), "certs", "ca.pem")))
}

func (s *PuppetSecurity) privateKeyDir() string {
	return filepath.FromSlash((filepath.Join(s.sslDir(), "private_keys")))
}

func (s *PuppetSecurity) privateKeyPath() string {
	return filepath.FromSlash(filepath.Join(s.privateKeyDir(), fmt.Sprintf("%s.pem", s.Identity())))
}

func (s *PuppetSecurity) createSSLDirectories() error {
	ssl := s.sslDir()

	err := os.MkdirAll(ssl, 0771)
	if err != nil {
		return err
	}

	for _, dir := range []string{"certificate_requests", "certs", "public_keys"} {
		path := filepath.FromSlash(filepath.Join(ssl, dir))
		err = os.MkdirAll(path, 0755)
		if err != nil {
			return err
		}
	}

	for _, dir := range []string{"private_keys", "private"} {
		path := filepath.FromSlash(filepath.Join(ssl, dir))
		err = os.MkdirAll(path, 0750)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *PuppetSecurity) csrPath() string {
	return filepath.FromSlash((filepath.Join(s.sslDir(), "certificate_requests", fmt.Sprintf("%s.pem", s.Identity()))))
}

func (s *PuppetSecurity) publicCertPath() string {
	return filepath.FromSlash((filepath.Join(s.sslDir(), "certs", fmt.Sprintf("%s.pem", s.Identity()))))
}

func (s *PuppetSecurity) sslDir() string {
	return s.conf.SSLDir
}

func (s *PuppetSecurity) writeCSR(key *rsa.PrivateKey, cn string, ou string) (string, error) {
	if s.csrExists() {
		return "", fmt.Errorf("a certificate request already exist for %s", s.Identity())
	}

	path := s.csrPath()

	subj := pkix.Name{
		CommonName:         cn,
		OrganizationalUnit: []string{ou},
	}

	asn1Subj, err := asn1.Marshal(subj.ToRDNSequence())
	if err != nil {
		return "", fmt.Errorf("could not create subject: %s", err)
	}

	template := x509.CertificateRequest{
		RawSubject:         asn1Subj,
		SignatureAlgorithm: x509.SHA256WithRSA,
	}

	template.DNSNames = append(template.DNSNames, s.conf.AltNames...)

	csrBytes, err := x509.CreateCertificateRequest(rand.Reader, &template, key)
	if err != nil {
		return "", fmt.Errorf("could not create csr: %s", err)
	}

	csr, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0640)
	if err != nil {
		return "", fmt.Errorf("could not open csr %s for writing: %s", path, err)
	}
	defer csr.Close()

	err = pem.Encode(csr, &pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrBytes})
	if err != nil {
		return "", fmt.Errorf("could not encode csr into %s: %s", path, err)
	}

	return fmt.Sprintf("%x", sha256.Sum256(csrBytes)), nil
}

func (s *PuppetSecurity) puppetCA() srvcache.Server {
	found := srvcache.NewServer(s.conf.PuppetCAHost, s.conf.PuppetCAPort, "https")

	if s.conf.DisableSRV || s.res == nil {
		return found
	}

	servers, err := s.res.QuerySrvRecords([]string{"_x-puppet-ca._tcp", "_x-puppet._tcp"})
	if err != nil {
		s.log.Warnf("Could not resolve Puppet CA SRV records: %s", err)
		return found
	}

	if servers.Count() == 0 {
		return found
	}

	found = servers.Servers()[0]

	if found.Scheme() == "" {
		found.SetScheme("https")
	}

	return found
}

func (s *PuppetSecurity) fetchCert() error {
	if s.publicCertExists() {
		return nil
	}

	server := s.puppetCA()
	url := fmt.Sprintf("%s://%s:%d/puppet-ca/v1/certificate/%s?environment=production", server.Scheme(), server.Host(), server.Port(), s.Identity())

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("could not create http request: %s", err)
	}

	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("User-Agent", "Choria Orchestrator - http://choria.io")

	client, err := s.HTTPClient(server.Scheme() == "https")
	if err != nil {
		return fmt.Errorf("could not set up HTTP connection: %s", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("could not fetch certificate: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("could not fetch certificate: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("could not read response body: %s", err)
	}

	err = os.WriteFile(s.publicCertPath(), body, 0644)
	if err != nil {
		return err
	}

	return nil
}

func (s *PuppetSecurity) fetchCA() error {
	if s.caExists() {
		return nil
	}

	server := s.puppetCA()
	url := fmt.Sprintf("%s://%s:%d/puppet-ca/v1/certificate/ca?environment=production", server.Scheme(), server.Host(), server.Port())

	// specifically disabling verification as at this point we do not have
	// the CA needed to do verification, there's no choice in the matter
	// really and this is just how its designed to work
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}

	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("could not read response body: %s", err)
	}

	if resp.StatusCode != 200 {
		return errors.New(string(body))
	}

	err = os.WriteFile(s.caPath(), body, 0644)
	if err != nil {
		return err
	}

	return nil
}

func (s *PuppetSecurity) submitCSR() error {
	csr, err := s.csrTXT()
	if err != nil {
		return fmt.Errorf("could not read CSR: %s", err)
	}

	server := s.puppetCA()

	url := fmt.Sprintf("%s://%s:%d/puppet-ca/v1/certificate_request/%s?environment=production", server.Scheme(), server.Host(), server.Port(), s.Identity())

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(csr))
	if err != nil {
		return fmt.Errorf("could not create http request: %s", err)
	}

	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("User-Agent", "Choria Orchestrator - http://choria.io")

	req.Host = server.Host()

	client, err := s.HTTPClient(server.Scheme() == "https")
	if err != nil {
		return fmt.Errorf("could not set up HTTP connection: %s", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("could not send CSR: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("could not read response body: %s", err)
	}

	if len(body) > 0 {
		return fmt.Errorf("could not send CSR to %s://%s:%d: %s: %s", server.Scheme(), server.Host(), server.Port(), resp.Status, string(body))
	}

	return fmt.Errorf("could not send CSR to %s://%s:%d: %s", server.Scheme(), server.Host(), server.Port(), resp.Status)
}

// HTTPClient creates a standard HTTP client with optional security, it will
// be set to use the CA and client certs for auth. servername should match the
// remote hosts name for SNI
func (s *PuppetSecurity) HTTPClient(secure bool) (*http.Client, error) {
	return s.fsec.HTTPClient(secure)
}

func (s *PuppetSecurity) csrTXT() ([]byte, error) {
	return os.ReadFile(s.csrPath())
}

func (s *PuppetSecurity) readPrivateKey() (*rsa.PrivateKey, error) {
	if !s.privateKeyExists() {
		return nil, fmt.Errorf("key not found in %s", s.privateKeyPath())
	}

	pd, err := os.ReadFile(s.privateKeyPath())
	if err != nil {
		return nil, err
	}

	privPem, _ := pem.Decode(pd)
	if privPem.Type != "RSA PRIVATE KEY" {
		return nil, fmt.Errorf("key file %s did not contain a private key", s.privateKeyPath())
	}

	parsedKey, err := x509.ParsePKCS1PrivateKey(privPem.Bytes)
	if err != nil {
		return nil, err
	}

	return parsedKey, nil
}

func (s *PuppetSecurity) writePrivateKey() (*rsa.PrivateKey, error) {
	if s.privateKeyExists() {
		return nil, fmt.Errorf("a private key already exist for %s", s.Identity())
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("could not generate rsa key: %s", err)
	}

	pemdata := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(key),
		},
	)

	err = os.WriteFile(s.privateKeyPath(), pemdata, 0640)
	if err != nil {
		return nil, fmt.Errorf("could not write private key: %s", err)
	}

	return key, nil
}

func (s *PuppetSecurity) csrExists() bool {
	return util.FileExist(s.csrPath())
}

func (s *PuppetSecurity) privateKeyExists() bool {
	return util.FileExist(s.privateKeyPath())
}

func (s *PuppetSecurity) publicCertExists() bool {
	return util.FileExist(s.publicCertPath())
}

func (s *PuppetSecurity) caExists() bool {
	return util.FileExist(s.caPath())
}
