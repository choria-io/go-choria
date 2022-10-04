// Copyright (c) 2020-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package pkcs11sec

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/choria-io/go-choria/inter"
	"github.com/miekg/pkcs11"
	"github.com/miekg/pkcs11/p11"
	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/providers/security/filesec"
)

// fetched from https://golang.org/src/crypto/rsa/pkcs1v15.go
var hashPrefixes = map[crypto.Hash][]byte{
	crypto.MD5:       {0x30, 0x20, 0x30, 0x0c, 0x06, 0x08, 0x2a, 0x86, 0x48, 0x86, 0xf7, 0x0d, 0x02, 0x05, 0x05, 0x00, 0x04, 0x10},
	crypto.SHA1:      {0x30, 0x21, 0x30, 0x09, 0x06, 0x05, 0x2b, 0x0e, 0x03, 0x02, 0x1a, 0x05, 0x00, 0x04, 0x14},
	crypto.SHA224:    {0x30, 0x2d, 0x30, 0x0d, 0x06, 0x09, 0x60, 0x86, 0x48, 0x01, 0x65, 0x03, 0x04, 0x02, 0x04, 0x05, 0x00, 0x04, 0x1c},
	crypto.SHA256:    {0x30, 0x31, 0x30, 0x0d, 0x06, 0x09, 0x60, 0x86, 0x48, 0x01, 0x65, 0x03, 0x04, 0x02, 0x01, 0x05, 0x00, 0x04, 0x20},
	crypto.SHA384:    {0x30, 0x41, 0x30, 0x0d, 0x06, 0x09, 0x60, 0x86, 0x48, 0x01, 0x65, 0x03, 0x04, 0x02, 0x02, 0x05, 0x00, 0x04, 0x30},
	crypto.SHA512:    {0x30, 0x51, 0x30, 0x0d, 0x06, 0x09, 0x60, 0x86, 0x48, 0x01, 0x65, 0x03, 0x04, 0x02, 0x03, 0x05, 0x00, 0x04, 0x40},
	crypto.MD5SHA1:   {},
	crypto.RIPEMD160: {0x30, 0x20, 0x30, 0x08, 0x06, 0x06, 0x28, 0xcf, 0x06, 0x03, 0x00, 0x31, 0x04, 0x14},
}

type Pkcs11Security struct {
	conf *Config
	log  *logrus.Entry

	fsec *filesec.FileSecurity

	cert    *tls.Certificate
	pKey    *PrivateKey
	pin     *string
	session p11.Session
}

type PrivateKey struct {
	PublicKey  crypto.PublicKey
	PrivateKey *p11.PrivateKey
}

func (k *PrivateKey) Public() crypto.PublicKey {
	return k.PublicKey
}

// Sign signs any compatible hash that is sent to it (see hashPrefixes for supported hashes)
// need to handle as many hash types as possible, since this is being used by http/tls driver
func (k *PrivateKey) Sign(_ io.Reader, msg []byte, opts crypto.SignerOpts) (signature []byte, err error) {
	prefix, ok := hashPrefixes[opts.HashFunc()]
	if !ok {
		return nil, fmt.Errorf("unknown hash function")
	}
	mechanism := pkcs11.NewMechanism(pkcs11.CKM_RSA_PKCS, nil)
	input := append(prefix, msg...)

	output, err := k.PrivateKey.Sign(*mechanism, input)
	if err != nil {
		return nil, err
	}
	return output, nil
}

type Config struct {
	// CAFile is the file where the trusted CA cert resides
	CAFile string

	// PrivilegedUsers is a list of regular expressions that identity privileged users
	PrivilegedUsers []string

	// AllowList is a list of regular expressions that identity valid users to allow in
	AllowList []string

	// DisableTLSVerify disables TLS verify in HTTP clients etc
	DisableTLSVerify bool

	// PKCS11DriverFile points to the dynamic library file to use (usually a .so file)
	PKCS11DriverFile string

	// PKCS11Slot specifies which slot of the pkcs11 device to use
	PKCS11Slot uint

	// RemoteSigner is the signer used to sign requests using a remote like AAA Service
	RemoteSigner inter.RequestSigner
}

func New(opts ...Option) (*Pkcs11Security, error) {
	p := &Pkcs11Security{}

	for _, opt := range opts {
		err := opt(p)
		if err != nil {
			return nil, err
		}
	}

	if p.conf == nil {
		return nil, fmt.Errorf("configuration not given")
	}

	if p.log == nil {
		return nil, fmt.Errorf("logger not given")
	}

	if p.conf.PKCS11DriverFile == "" {
		return nil, fmt.Errorf("pkcs11: PKCS11DriverFile option is required")
	}

	if p.pin != nil {
		err := p.loginToToken()
		if err != nil {
			return nil, fmt.Errorf("failed to login to token in New(): %s", err)
		}
	}

	return p, p.reinit()
}

func (p *Pkcs11Security) promptForPin() (*string, error) {
	pin := ""
	prompt := &survey.Password{
		Message: "PIN",
	}
	err := survey.AskOne(prompt, &pin)
	if err != nil {
		return nil, err
	}
	return &pin, nil
}

func (p *Pkcs11Security) loginToToken() error {
	var err error

	if p.pin == nil {
		p.pin, err = p.promptForPin()
		if err != nil {
			fmt.Printf("err is %s", err.Error())
			return err
		}
	}

	p.log.Debugf("Attempting to open PKCS11 driver file %s", p.conf.PKCS11DriverFile)

	module, err := p11.OpenModule(p.conf.PKCS11DriverFile)
	if err != nil {
		return fmt.Errorf("failed to open PKCS11 driver file %s: %s", p.conf.PKCS11DriverFile, err)
	}

	p.log.Debug("Attempting to fetch PKCS11 driver slots")

	slots, err := module.Slots()
	if err != nil {
		return fmt.Errorf("failed to fetch PKCS11 driver slots: %s", err)
	}

	var slot *p11.Slot
	found := false
	for _, aSlot := range slots {
		p.log.Debugf("Found slot %d", aSlot.ID())

		if aSlot.ID() == p.conf.PKCS11Slot {
			slot = &aSlot
			found = true
			break
		}
	}
	if !found {
		if len(slots) == 1 {
			slot = &slots[0]
		} else {
			return fmt.Errorf("failed to find slot with label %d", p.conf.PKCS11Slot)
		}
	}
	p.log.Debugf("Attempting to open session for selected slot %d", p.conf.PKCS11Slot)

	session, err := slot.OpenSession()
	if err != nil {
		return fmt.Errorf("failed to open PKCS11 session: %s", err)
	}

	p.session = session

	err = session.Login(*p.pin)
	if err != nil {
		if !strings.Contains(err.Error(), "CKR_USER_ALREADY_LOGGED_IN") {
			return fmt.Errorf("failed to login with provided pin: %s", err)
		}
	}

	p.log.Debug("Attempting to find private key object")
	privateKeyObject, err := session.FindObject([]*pkcs11.Attribute{pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PRIVATE_KEY)})
	if err != nil {
		return fmt.Errorf("failed to find private key object: %s", err)
	}

	p.log.Debug("Attempting to find certificate object")
	certObject, err := session.FindObject([]*pkcs11.Attribute{pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_CERTIFICATE)})
	if err != nil {
		return fmt.Errorf("failed to find certificate object: %s", err)
	}

	certData, err := certObject.Value()
	if err != nil {
		return fmt.Errorf("failed to get certificate object value: %s", err)
	}

	parsedCert, err := x509.ParseCertificate(certData)
	if err != nil {
		return fmt.Errorf("failed to parse X509 certificate: %s", err)
	}

	if parsedCert.Subject.CommonName == "" {
		return fmt.Errorf("cert on token must have valid CommonName")
	}

	pubKey, ok := parsedCert.PublicKey.(crypto.PublicKey)
	if !ok {
		return fmt.Errorf("public key in certificate is not a crypto.PublicKey: %s", err)
	}

	privateKey := p11.PrivateKey(privateKeyObject)

	p.pKey = &PrivateKey{
		PublicKey:  pubKey,
		PrivateKey: &privateKey,
	}

	p.cert = &tls.Certificate{
		Certificate: [][]byte{certData},
		Leaf:        parsedCert,
		PrivateKey:  p.pKey,
	}

	return nil
}

// PublicCert is the parsed public certificate
func (p *Pkcs11Security) PublicCert() (*x509.Certificate, error) {
	if p.cert == nil {
		return nil, fmt.Errorf("not logged in")
	}

	return p.cert.Leaf, nil
}

func (p *Pkcs11Security) reinit() error {
	var err error

	fc := filesec.Config{
		AllowList:        p.conf.AllowList,
		DisableTLSVerify: p.conf.DisableTLSVerify,
		PrivilegedUsers:  p.conf.PrivilegedUsers,
		CA:               p.conf.CAFile,
		Certificate:      "unused",
		Identity:         "unused",
		RemoteSigner:     p.conf.RemoteSigner,
	}

	p.fsec, err = filesec.New(filesec.WithConfig(&fc), filesec.WithLog(p.log))
	if err != nil {
		return err
	}

	return nil
}

func (p *Pkcs11Security) Logout() error {
	return p.session.Logout()
}

func (p *Pkcs11Security) BackingTechnology() inter.SecurityTechnology {
	return p.fsec.BackingTechnology()
}

func (p *Pkcs11Security) Provider() string {
	return "pkcs11"
}

func (p *Pkcs11Security) Enroll(ctx context.Context, wait time.Duration, cb func(digest string, try int)) error {
	return fmt.Errorf("pkcs11 security provider does not support enrollment")
}

// RemoteSignRequest signs a choria request against using a remote signer and returns a secure request
func (p *Pkcs11Security) RemoteSignRequest(ctx context.Context, str []byte) (signed []byte, err error) {
	return nil, fmt.Errorf("pkcs11 security provider does not support remote signing requests")
}

func (p *Pkcs11Security) IsRemoteSigning() bool { return false }

// Validate determines if the node represents a valid SSL configuration
func (p *Pkcs11Security) Validate() ([]string, bool) {
	var errorsList []string

	stat, err := os.Stat(p.conf.CAFile)
	switch {
	case os.IsNotExist(err):
		errorsList = append(errorsList, err.Error())
	case !stat.Mode().IsRegular():
		errorsList = append(errorsList, fmt.Sprintf("%s is not a regular file", p.conf.CAFile))
	}

	if p.pin == nil {
		p.log.Debug("Attempting to login to token in Validate()")
		if err := p.loginToToken(); err != nil {
			errorsList = append(errorsList, fmt.Sprintf("failed to login to token in Validate(): %s", err))
		}
	}

	return errorsList, len(errorsList) == 0
}

// ChecksumBytes calculates a sha256 checksum for data
func (p *Pkcs11Security) ChecksumBytes(data []byte) []byte {
	return p.fsec.ChecksumBytes(data)
}

// SignBytes signs a message using a SHA256 PKCS1v15 protocol
func (p *Pkcs11Security) SignBytes(str []byte) ([]byte, error) {
	hashed := p.ChecksumBytes(str)
	mechanism := pkcs11.NewMechanism(pkcs11.CKM_RSA_PKCS, nil)
	input := append(hashPrefixes[crypto.SHA256], hashed...)

	output, err := p.pKey.PrivateKey.Sign(*mechanism, input)
	if err != nil {
		return nil, err
	}
	return output, nil
}

// VerifyByteSignature verify that dat matches signature sig made by the key, if pub cert is empty the active public key will be used
func (p *Pkcs11Security) VerifyByteSignature(dat []byte, sig []byte, public ...[]byte) (should bool, signer string) {
	if len(public) != 1 {
		p.log.Errorf("Could not process public data: only single signer public data is supported")
		return false, ""
	}

	pubcert := public[0]

	var cert *x509.Certificate
	var err error

	if len(pubcert) > 0 {
		pkpem, _ := pem.Decode(pubcert)
		if pkpem == nil {
			p.log.Errorf("Could not decode PEM data in public key: invalid pem data")
			return false, ""
		}

		cert, err = x509.ParseCertificate(pkpem.Bytes)
		if err != nil {
			p.log.Errorf("Could not parse decoded PEM data for public key: %s", err)
			return false, ""
		}
	} else {
		cert = p.cert.Leaf
	}

	rsaPublicKey := cert.PublicKey.(*rsa.PublicKey)
	hashed := p.ChecksumBytes(dat)

	err = rsa.VerifyPKCS1v15(rsaPublicKey, crypto.SHA256, hashed[:], sig)
	if err != nil {
		p.log.Errorf("Signature verification failed: %s", err)
		return false, ""
	}

	names := []string{cert.Subject.CommonName}
	names = append(names, cert.DNSNames...)

	if len(names) == 0 {
		p.log.Errorf("Signature verification failed: no names found in signer certificate")
		return false, ""
	}

	p.log.Debugf("Verified signature from %s", strings.Join(names, ", "))

	return true, names[0]
}

// CallerName creates a choria like caller name in the form of choria=identity
func (p *Pkcs11Security) CallerName() string {
	return fmt.Sprintf("choria=%s", p.cert.Leaf.Subject.CommonName)
}

// CallerIdentity extracts the identity from a choria like caller name in the form of choria=identity
func (p *Pkcs11Security) CallerIdentity(caller string) (string, error) {
	return p.fsec.CallerIdentity(caller)
}

// ShouldAllowCaller verifies the public data
func (p *Pkcs11Security) ShouldAllowCaller(data []byte, name string) (privileged bool, err error) {
	return p.fsec.ShouldAllowCaller(data, name)
}

// VerifyCertificate verifies a certificate is signed with the configured CA and if
// name is not "" that it matches the name given
func (p *Pkcs11Security) VerifyCertificate(certpem []byte, name string) error {
	return p.fsec.VerifyCertificate(certpem, name)
}

// PublicCertPem retrieves the public certificate for this instance
func (p *Pkcs11Security) PublicCertPem() (*pem.Block, error) {
	pb := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: p.cert.Leaf.Raw,
	}

	return pb, nil
}

// PublicCertBytes retrieves pem data in textual form for the public certificate of the current identity
func (p *Pkcs11Security) PublicCertBytes() ([]byte, error) {

	pemCert, err := p.PublicCertPem()
	if err != nil {
		return nil, fmt.Errorf("failed to run PublicCertPem: %s", err)
	}
	var buf bytes.Buffer
	err = pem.Encode(&buf, pemCert)
	if err != nil {
		return nil, fmt.Errorf("failed to run pem.Encode: %s", err)
	}
	return buf.Bytes(), nil
}

// Identity determines the choria certname
func (p *Pkcs11Security) Identity() string {
	return p.cert.Leaf.Subject.CommonName
}

// ClientTLSConfig creates a client TLS configuration
func (p *Pkcs11Security) ClientTLSConfig() (*tls.Config, error) {
	return p.TLSConfig()
}

// TLSConfig creates a TLS configuration for use by NATS, HTTPS etc
func (p *Pkcs11Security) TLSConfig() (*tls.Config, error) {
	caCert, err := os.ReadFile(p.conf.CAFile)
	if err != nil {
		return nil, err
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	tlsc := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{*p.cert},
		GetClientCertificate: func(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
			return p.cert, nil
		},
		ClientCAs: caCertPool,
		RootCAs:   caCertPool,
	}

	if p.conf.DisableTLSVerify {
		tlsc.InsecureSkipVerify = true
	}

	return tlsc, nil
}

// SSLContext creates a SSL context loaded with our certs and ca
func (p *Pkcs11Security) SSLContext() (*http.Transport, error) {
	tlsConfig, err := p.TLSConfig()
	if err != nil {
		return nil, err
	}

	transport := &http.Transport{TLSClientConfig: tlsConfig}

	return transport, nil
}

func (p *Pkcs11Security) HTTPClient(secure bool) (*http.Client, error) {
	client := &http.Client{}

	if secure {
		tlsc, err := p.TLSConfig()
		if err != nil {
			return nil, fmt.Errorf("pkcs11: could not set up HTTP connection: %s", err)
		}

		client.Transport = &http.Transport{TLSClientConfig: tlsc}
	}

	return client, nil
}
