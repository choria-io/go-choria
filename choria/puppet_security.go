package choria

import (
	"bytes"
	context "context"
	"crypto"
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
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-protocol/protocol"
	"github.com/sirupsen/logrus"
)

// PuppetSecurity impliments SecurityProvider reusing AIO Puppet settings
type PuppetSecurity struct {
	fw    settingsProvider
	conf  *Config
	log   *logrus.Entry
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

	return p, nil
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
	ok := false

	if _, err := s.sslDir(); err != nil {
		errors = append(errors, fmt.Sprintf("SSL Directory does not exist: %s", err))
		return errors, false
	}

	if c, err := s.publicCertPath(); err == nil {
		if !s.publicCertExists() {
			errors = append(errors, fmt.Sprintf("public certificate %s does not exist", c))
		}
	} else {
		errors = append(errors, fmt.Sprintf("could not determine public certificate path: %s", err))
	}

	if c, err := s.privateKeyPath(); err == nil {
		if !s.privateKeyExists() {
			errors = append(errors, fmt.Sprintf("private key %s does not exist", c))
		}
	} else {
		errors = append(errors, fmt.Sprintf("could not determine private certificate path: %s", err))
	}

	if c, err := s.caPath(); err == nil {
		if !s.caExists() {
			errors = append(errors, fmt.Sprintf("CA %s does not exist", c))
		}
	} else {
		errors = append(errors, fmt.Sprintf("could not determine CA path: %s", err))
	}

	ok = len(errors) == 0

	return errors, ok
}

// ChecksumBytes calculates a sha256 checksum for data
func (s *PuppetSecurity) ChecksumBytes(data []byte) []byte {
	sum := sha256.Sum256(data)

	return sum[:]
}

// ChecksumString calculates a sha256 checksum for data
func (s *PuppetSecurity) ChecksumString(data string) []byte {
	return s.ChecksumBytes([]byte(data))
}

// SignBytes signs a message using a SHA256 PKCS1v15 protocol
func (s *PuppetSecurity) SignBytes(str []byte) ([]byte, error) {
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
func (s *PuppetSecurity) VerifyByteSignature(dat []byte, sig []byte, identity string) bool {
	pubkeyPath := ""
	var err error

	pubkeyPath, err = s.publicCertPath()

	if identity != "" {
		pubkeyPath, err = s.cachePath(identity)
	}

	if err != nil {
		s.log.Errorf("Could not verify signature: %s", err)
		return false
	}

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

	return true
}

// VerifyStringSignature verify that str matches signature sig made by the key of identity
func (s *PuppetSecurity) VerifyStringSignature(str string, sig []byte, identity string) bool {
	return s.VerifyByteSignature([]byte(str), sig, identity)
}

// PrivilegedVerifyByteSignature verifies if the signature received is from any of the privileged certs or the given identity
func (s *PuppetSecurity) PrivilegedVerifyByteSignature(dat []byte, sig []byte, identity string) bool {
	var candidates []string

	if identity != "" {
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

// PrivilegedVerifyStringSignature verifies if the signature received is from any of the privilged certs or the given identity
func (s *PuppetSecurity) PrivilegedVerifyStringSignature(dat string, sig []byte, identity string) bool {
	return s.PrivilegedVerifyByteSignature([]byte(dat), sig, identity)
}

// SignString signs a message using a SHA256 PKCS1v15 protocol
func (s *PuppetSecurity) SignString(str string) ([]byte, error) {
	return s.SignBytes([]byte(str))
}

// CallerName creates a choria like caller name in the form of choria=identity
func (s *PuppetSecurity) CallerName() string {
	return fmt.Sprintf("choria=%s", s.Identity())
}

// CallerIdentity extracts the identity from a choria like caller name in the form of choria=identity
func (s *PuppetSecurity) CallerIdentity(caller string) (string, error) {
	re := regexp.MustCompile("^choria=([\\w\\.\\-]+)")
	match := re.FindStringSubmatch(caller)

	if match == nil {
		return "", fmt.Errorf("could not find a valid caller identity name in %s", caller)
	}

	return match[1], nil
}

// CachePublicData caches the public key for a identity
func (s *PuppetSecurity) CachePublicData(data []byte, identity string) error {
	certfile, err := s.cachePath(identity)
	if err != nil {
		return err
	}

	if !s.shouldCacheClientCert(data, identity) {
		return fmt.Errorf("certificate '%s' did not pass validation", identity)
	}

	err = ioutil.WriteFile(certfile, []byte(data), os.FileMode(int(0644)))
	if err != nil {
		return fmt.Errorf("could not cache client public certificate: %s", err.Error())
	}

	return nil
}

// CachedPublicData retrieves the previously cached public data for a given identity
func (s *PuppetSecurity) CachedPublicData(identity string) ([]byte, error) {
	certfile, err := s.cachePath(identity)
	if err != nil {
		return []byte{}, fmt.Errorf("could not cache public data: %s", err)
	}

	if _, err := os.Stat(certfile); os.IsNotExist(err) {
		return []byte{}, fmt.Errorf("unknown public data: %s", identity)
	}

	return ioutil.ReadFile(certfile)
}

func (s *PuppetSecurity) privilegedCerts() []string {
	certs := []string{}

	cache, err := s.certCacheDir()
	if err != nil {
		return []string{}
	}

	filepath.Walk(cache, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			cert := []byte(strings.TrimSuffix(filepath.Base(path), ".pem"))

			if MatchAnyRegex(cert, s.conf.Choria.PrivilegedUsers) {
				certs = append(certs, string(cert))
			}
		}

		return nil
	})

	sort.Strings(certs)

	return certs
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
	ca, err := s.caPath()
	if err != nil {
		s.log.Errorf("Could not determine CA location: %s", err)
		return err, false
	}

	capem, err := ioutil.ReadFile(ca)
	if err != nil {
		s.log.Errorf("Could not read CA '%s': %s", s.caPath, err)
		return errors.New(err.Error()), false
	}

	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(capem) {
		s.log.Warnf("Could not use CA '%s' as PEM data: %s", ca, err)
		return errors.New(err.Error()), false
	}

	block, _ := pem.Decode(certpem)
	if block == nil {
		s.log.Warnf("Could not decode certificate '%s' PEM data: %s", name, err)
		return errors.New(err.Error()), false
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		s.log.Warnf("Could not parse certificate '%s': %s", name, err)
		return errors.New(err.Error()), false
	}

	opts := x509.VerifyOptions{
		Roots: roots,
	}

	if name != "" {
		opts.DNSName = name
	}

	_, err = cert.Verify(opts)
	if err != nil {
		s.log.Warnf("Certificate does not pass verification as '%s': %s", name, err)
		return errors.New(err.Error()), false
	}

	return nil, true
}

// PublicCertPem retrieves the public certificate for this instance
func (s *PuppetSecurity) PublicCertPem() (*pem.Block, error) {
	path, err := s.publicCertPath()
	if err != nil {
		return nil, err
	}

	return s.decodePEM(path)
}

// PublicCertTXT retrieves pem data in textual form for the public certificate of the current identity
func (s *PuppetSecurity) PublicCertTXT() ([]byte, error) {
	path, err := s.publicCertPath()
	if err != nil {
		return nil, err
	}

	return ioutil.ReadFile(path)
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
	pub, _ := s.publicCertPath()
	pri, _ := s.privateKeyPath()
	ca, _ := s.caPath()

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

// SSLContext creates a SSL context loaded with our certs and ca
func (s *PuppetSecurity) SSLContext() (*http.Transport, error) {
	tlsConfig, err := s.TLSConfig()
	if err != nil {
		return nil, err
	}

	transport := &http.Transport{TLSClientConfig: tlsConfig}

	return transport, nil
}

func (s *PuppetSecurity) decodePEM(certpath string) (*pem.Block, error) {
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

func (s *PuppetSecurity) certCacheDir() (string, error) {
	ssldir, err := s.sslDir()
	if err != nil {
		return "", fmt.Errorf("could not determine Client Certificate Cache Directory: %s", err)
	}

	path := filepath.FromSlash(filepath.Join(ssldir, "choria_security", "public_certs"))

	err = os.MkdirAll(path, os.FileMode(int(0755)))
	if err != nil {
		return "", fmt.Errorf("could not create Client Certificate Cache Directory: %s", err)
	}

	return path, nil
}

func (s *PuppetSecurity) privateKeyPEM() (pb *pem.Block, err error) {
	key, err := s.privateKeyPath()
	if err != nil {
		return pb, fmt.Errorf("Could not read Client Private Key PEM data: %s", err)
	}

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

	if os.Getenv("HOME") == "" {
		return "", errors.New("cannot determine home dir while looking for SSL Directory, no HOME environment is set.  Please set HOME or configure plugin.choria.ssldir")
	}

	return filepath.FromSlash(filepath.Join(os.Getenv("HOME"), ".puppetlabs", "etc", "puppet", "ssl")), nil
}

func (s *PuppetSecurity) shouldCacheClientCert(data []byte, name string) bool {
	if err, ok := s.VerifyCertificate(data, ""); !ok {
		s.log.Warnf("Received certificate '%s' certiicate did not pass verification: %s", name, err)
		return false
	}

	if MatchAnyRegex([]byte(name), s.conf.Choria.PrivilegedUsers) {
		s.log.Warnf("Caching privileged certificate %s", name)
		return true
	}

	if err, ok := s.VerifyCertificate(data, name); !ok {
		s.log.Warnf("Received certificate '%s' did not pass verification: %s", name, err)
		return false
	}

	if !MatchAnyRegex([]byte(name), s.conf.Choria.CertnameWhitelist) {
		s.log.Warnf("Received certificate '%s' does not match the allowed list '%s'", name, s.conf.Choria.CertnameWhitelist)
		return false
	}

	return true
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
	} else {
		return fmt.Errorf("could not send CSR to %s://%s:%d: %s", server.Scheme, server.Host, server.Port, resp.Status)
	}
}

// HTTPClient creates a standard HTTP client with optional security, it will
// be set to use the CA and client certs for auth. servername should match the
// remote hosts name for SNI
func (s *PuppetSecurity) HTTPClient(secure bool) (*http.Client, error) {
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

func (s *PuppetSecurity) csrTXT() ([]byte, error) {
	path, err := s.csrPath()
	if err != nil {
		return nil, err
	}

	return ioutil.ReadFile(path)
}

func (s *PuppetSecurity) writePrivateKey() (*rsa.PrivateKey, error) {
	if s.privateKeyExists() {
		return nil, fmt.Errorf("a private key already exist for %s", s.Identity())
	}

	path, err := s.privateKeyPath()
	if err != nil {
		return nil, fmt.Errorf("could not determine private key path: %s", err)
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
	path, err := s.privateKeyPath()
	if err != nil {
		return false
	}

	if _, err := os.Stat(path); err != nil {
		return false
	}

	return true
}

func (s *PuppetSecurity) publicCertExists() bool {
	path, err := s.publicCertPath()
	if err != nil {
		return false
	}

	if _, err := os.Stat(path); err != nil {
		return false
	}

	return true
}

func (s *PuppetSecurity) caExists() bool {
	path, err := s.caPath()
	if err != nil {
		return false
	}

	if _, err := os.Stat(path); err != nil {
		return false
	}

	return true
}
