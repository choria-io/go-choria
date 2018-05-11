package choria

import (
	"bytes"
	context "context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-protocol/protocol"
	"github.com/sirupsen/logrus"
)

// PuppetSecurity impliments SecurityProvider reusing AIO Puppet settings
// it supports enrollment the same way `puppet agent --waitforcert 10` does
type PuppetSecurity struct {
	fw   settingsProvider
	conf *Config
	log  *logrus.Entry

	fsec  *FileSecurity
	cache string
}

type settingsProvider interface {
	PuppetSetting(string) (string, error)
	Getuid() int
	QuerySrvRecords(records []string) ([]Server, error)
}

// NewPuppetSecurity creates a new instance of the Puppet Security provider
func NewPuppetSecurity(fw settingsProvider, conf *Config, log *logrus.Entry) (*PuppetSecurity, error) {
	p := &PuppetSecurity{
		fw:   fw,
		conf: conf,
		log:  log.WithFields(logrus.Fields{"ssl": "puppet"}),
	}

	return p, p.reinit()
}

func (s *PuppetSecurity) reinit() error {
	var err error

	s.conf.Choria.FileSecurityCA, err = s.caPath()
	if err != nil {
		return err
	}

	s.conf.Choria.FileSecurityCache, err = s.certCacheDir()
	if err != nil {
		return err
	}

	s.conf.Choria.FileSecurityCertificate, err = s.publicCertPath()
	if err != nil {
		return err
	}

	s.conf.Choria.FileSecurityKey, err = s.privateKeyPath()
	if err != nil {
		return err
	}

	s.fsec, err = NewFileSecurity(s.fw, s.conf, s.log)
	if err != nil {
		return err
	}

	return nil
}

// Enroll sends a CSR to the PuppetCA and wait for it to be signed
func (s *PuppetSecurity) Enroll(ctx context.Context, wait time.Duration, cb func(int)) error {
	err := s.createSSLDirectories()
	if err != nil {
		return fmt.Errorf("could not initialize ssl directories: %s", err)
	}

	if !(s.privateKeyExists() && s.csrExists() && s.publicCertExists()) {
		s.log.Debugf("Creating a new CSR and submitting it to the CA for %s", s.Identity())
		err = s.fetchCA()
		if err != nil {
			return fmt.Errorf("could not fetch CA: %s", err)
		}

		key, err := s.writePrivateKey()
		if err != nil {
			return fmt.Errorf("could not write a new private key: %s", err)
		}

		err = s.writeCSR(key, s.Identity(), "choria.io")
		if err != nil {
			return fmt.Errorf("could not write CSR: %s", err)
		}

		err = s.submitCSR()
		if err != nil {
			return fmt.Errorf("could not submit csr: %s", err)
		}
	}

	timeout := time.NewTimer(wait).C
	ticks := time.NewTicker(10 * time.Second).C

	complete := make(chan int, 2)

	attempt := 1

	fetcher := func() {
		cb(attempt)
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

	if _, err := s.sslDir(); err != nil {
		errors = append(errors, fmt.Sprintf("SSL Directory does not exist: %s", err))
		return errors, false
	}

	ferrs, _ := s.fsec.Validate()
	for _, err := range ferrs {
		errors = append(errors, err)
	}

	return errors, len(errors) == 0
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

func (s *PuppetSecurity) cachePath(identity string) (string, error) {
	var cache string
	var err error

	if s.cache == "" {
		cache, err = s.certCacheDir()
		if err != nil {
			return "", fmt.Errorf("cert cache dir does not exist: %s", err)
		}
	} else {
		cache = s.cache
	}

	certfile := filepath.Join(cache, fmt.Sprintf("%s.pem", identity))

	return certfile, nil
}

// VerifyCertificate verifies a certificate is signed with the configured CA and if
// name is not "" that it matches the name given
func (s *PuppetSecurity) VerifyCertificate(certpem []byte, name string) (error, bool) {
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

// Identity determines the choria certname
func (s *PuppetSecurity) Identity() string {
	if s.conf.OverrideCertname != "" {
		return s.conf.OverrideCertname
	}

	if certname, ok := os.LookupEnv("MCOLLECTIVE_CERTNAME"); ok {
		return certname
	}

	certname := s.conf.Identity

	if s.fw.Getuid() != 0 {
		if u, ok := os.LookupEnv("USER"); ok {
			certname = fmt.Sprintf("%s.mcollective", u)
		}
	}

	return certname
}

// TLSConfig creates a TLS configuration for use by NATS, HTTPS etc
func (s *PuppetSecurity) TLSConfig() (*tls.Config, error) {
	return s.fsec.TLSConfig()
}

// SSLContext creates a SSL context loaded with our certs and ca
func (s *PuppetSecurity) SSLContext() (*http.Transport, error) {
	return s.fsec.SSLContext()
}

func (s *PuppetSecurity) certCacheDir() (string, error) {
	ssldir, err := s.sslDir()
	if err != nil {
		return "", fmt.Errorf("could not determine Client Certificate Cache Directory: %s", err)
	}

	path := filepath.FromSlash(filepath.Join(ssldir, "choria_security", "public_certs"))

	return path, nil
}

func (s *PuppetSecurity) caPath() (string, error) {
	ssl, err := s.sslDir()
	if err != nil {
		return "", err
	}

	return filepath.FromSlash((filepath.Join(ssl, "certs", "ca.pem"))), nil
}

func (s *PuppetSecurity) privateKeyDir() (string, error) {
	ssl, err := s.sslDir()
	if err != nil {
		return "", err
	}

	return filepath.FromSlash((filepath.Join(ssl, "private_keys"))), nil
}

func (s *PuppetSecurity) privateKeyPath() (string, error) {
	dir, err := s.privateKeyDir()
	if err != nil {
		return "", err
	}

	return filepath.FromSlash(filepath.Join(dir, fmt.Sprintf("%s.pem", s.Identity()))), nil
}

func (s *PuppetSecurity) createSSLDirectories() error {
	ssl, err := s.sslDir()
	if err != nil {
		return err
	}

	err = os.MkdirAll(ssl, 0771)
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

func (s *PuppetSecurity) csrPath() (string, error) {
	ssl, err := s.sslDir()
	if err != nil {
		return "", err
	}

	return filepath.FromSlash((filepath.Join(ssl, "certificate_requests", fmt.Sprintf("%s.pem", s.Identity())))), nil
}

func (s *PuppetSecurity) publicCertPath() (string, error) {
	ssl, err := s.sslDir()
	if err != nil {
		return "", err
	}

	return filepath.FromSlash((filepath.Join(ssl, "certs", fmt.Sprintf("%s.pem", s.Identity())))), nil
}

func (s *PuppetSecurity) sslDir() (string, error) {
	if !protocol.IsSecure() {
		return filepath.FromSlash("/nonexisting"), nil
	}

	if s.conf.Choria.SSLDir != "" {
		return s.conf.Choria.SSLDir, nil
	}

	if s.fw.Getuid() == 0 {
		path, err := s.fw.PuppetSetting("ssldir")
		if err != nil {
			return "", err
		}

		// store it so future calls to this wil not call out to Puppet again
		s.conf.Choria.SSLDir = filepath.FromSlash(path)

		return s.conf.Choria.SSLDir, nil
	}

	homedir := os.Getenv("HOME")

	if runtime.GOOS == "windows" {
		if os.Getenv("HOMEDRIVE") == "" || os.Getenv("HOMEPATH") == "" {
			return "", errors.New("cannot determine home dir while looking for SSL Directory, no HOMEDRIVE or HOMEPATH environment is set.  Please set HOME or configure plugin.choria.ssldir")
		}

		homedir = filepath.Join(os.Getenv("HOMEDRIVE"), os.Getenv("HOMEPATH"))
	}

	return filepath.FromSlash(filepath.Join(homedir, ".puppetlabs", "etc", "puppet", "ssl")), nil
}

func (s *PuppetSecurity) writeCSR(key *rsa.PrivateKey, cn string, ou string) error {
	if s.csrExists() {
		return fmt.Errorf("a certificate request already exist for %s", s.Identity())
	}

	path, err := s.csrPath()
	if err != nil {
		return fmt.Errorf("could not determine csr path: %s", err)
	}

	subj := pkix.Name{
		CommonName:         cn,
		OrganizationalUnit: []string{ou},
	}

	asn1Subj, err := asn1.Marshal(subj.ToRDNSequence())
	if err != nil {
		return fmt.Errorf("could not create subject: %s", err)
	}

	template := x509.CertificateRequest{
		RawSubject:         asn1Subj,
		SignatureAlgorithm: x509.SHA256WithRSA,
	}

	csrBytes, err := x509.CreateCertificateRequest(rand.Reader, &template, key)
	if err != nil {
		return fmt.Errorf("could not create csr: %s", err)
	}

	csr, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0640)
	if err != nil {
		return fmt.Errorf("could not open csr %s for writing: %s", path, err)
	}
	defer csr.Close()

	err = pem.Encode(csr, &pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrBytes})
	if err != nil {
		return fmt.Errorf("could not encode csr into %s: %s", path, err)
	}

	return nil
}

func (s *PuppetSecurity) puppetCA() Server {
	server := Server{
		Host:   s.conf.Choria.PuppetCAHost,
		Port:   s.conf.Choria.PuppetCAPort,
		Scheme: "https",
	}

	// if either was specifically set and not using defaults then use whats there
	if s.conf.HasOption("plugin.choria.puppetca_host") || s.conf.HasOption("plugin.choria.puppetca_port") {
		return server
	}

	servers, err := s.fw.QuerySrvRecords([]string{"_x-puppet-ca._tcp", "_x-puppet._tcp"})
	if err != nil {
		s.log.Warnf("Could not resolve Puppet CA SRV records: %s", err)
	}

	if len(servers) == 0 {
		return server
	}

	if servers[0].Scheme == "" {
		servers[0].Scheme = "https"
	}

	return servers[0]
}

func (s *PuppetSecurity) fetchCert() error {
	if s.publicCertExists() {
		return nil
	}

	server := s.puppetCA()
	url := fmt.Sprintf("%s://%s:%d/puppet-ca/v1/certificate/%s?environment=production", server.Scheme, server.Host, server.Port, s.Identity())

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("could not create http request: %s", err)
	}

	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("User-Agent", fmt.Sprintf("Choria version %s http://choria.io", build.Version))

	client, err := s.HTTPClient(server.Scheme == "https")
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

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("could not read response body: %s", err)
	}

	path, err := s.publicCertPath()
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path, body, 0644)
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
	url := fmt.Sprintf("%s://%s:%d/puppet-ca/v1/certificate/ca?environment=production", server.Scheme, server.Host, server.Port)

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("could not read response body: %s", err)
	}

	if resp.StatusCode != 200 {
		return errors.New(string(body))
	}

	capath, err := s.caPath()
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(capath, body, 0644)
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

	url := fmt.Sprintf("%s://%s:%d/puppet-ca/v1/certificate_request/%s?environment=production", server.Scheme, server.Host, server.Port, s.Identity())

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(csr))
	if err != nil {
		return fmt.Errorf("could not create http request: %s", err)
	}

	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("User-Agent", fmt.Sprintf("Choria version %s http://choria.io", build.Version))

	req.Host = server.Host

	client, err := s.HTTPClient(server.Scheme == "https")
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

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("could not read response body: %s", err)
	}

	if len(body) > 0 {
		return fmt.Errorf("could not send CSR to %s://%s:%d: %s: %s", server.Scheme, server.Host, server.Port, resp.Status, string(body))
	}

	return fmt.Errorf("could not send CSR to %s://%s:%d: %s", server.Scheme, server.Host, server.Port, resp.Status)
}

// HTTPClient creates a standard HTTP client with optional security, it will
// be set to use the CA and client certs for auth. servername should match the
// remote hosts name for SNI
func (s *PuppetSecurity) HTTPClient(secure bool) (*http.Client, error) {
	return s.fsec.HTTPClient(secure)
}

func (s *PuppetSecurity) csrTXT() ([]byte, error) {
	path, err := s.csrPath()
	if err != nil {
		return nil, err
	}

	return ioutil.ReadFile(path)
}

func (s *PuppetSecurity) writePrivateKey() (*rsa.PrivateKey, error) {
	path, err := s.privateKeyPath()
	if err != nil {
		return nil, fmt.Errorf("could not determine private key path: %s", err)
	}

	if s.privateKeyExists() {
		return nil, fmt.Errorf("a private key already exist for %s", s.Identity())
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("could not generate rsa key: %s", err)
	}

	pkcs := x509.MarshalPKCS1PrivateKey(key)

	err = ioutil.WriteFile(path, pkcs, 0640)
	if err != nil {
		return nil, fmt.Errorf("could not write private key: %s", err)
	}

	return key, nil
}

func (s *PuppetSecurity) csrExists() bool {
	path, err := s.csrPath()
	if err != nil {
		return false
	}

	if _, err := os.Stat(path); err != nil {
		return false
	}

	return true
}

func (s *PuppetSecurity) privateKeyExists() bool {
	return s.fsec.privateKeyExists()
}

func (s *PuppetSecurity) publicCertExists() bool {
	return s.fsec.publicCertExists()
}

func (s *PuppetSecurity) caExists() bool {
	return s.fsec.caExists()
}
