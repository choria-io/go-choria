package choria

import (
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

	"github.com/choria-io/go-protocol/protocol"
)

// CheckSSLSetup validates the various SSL files and directories exist and are well formed
func (self *Framework) CheckSSLSetup() (errors []string, ok bool) {
	if _, err := self.SSLDir(); err != nil {
		errors = append(errors, fmt.Sprintf("SSL Directory does not exist: %s", err.Error()))
		return errors, false
	}

	if c, err := self.ClientPublicCert(); err == nil {
		if _, err := os.Stat(c); err != nil {
			errors = append(errors, fmt.Sprintf("The Public Certificate %s does not exist", c))
		}
	} else {
		errors = append(errors, fmt.Sprintf("Could not determine Public Certificate path: %s", err.Error()))
	}

	if c, err := self.ClientPrivateKey(); err == nil {
		if _, err := os.Stat(c); err != nil {
			errors = append(errors, fmt.Sprintf("The Private Key %s does not exist", c))
		}
	} else {
		errors = append(errors, fmt.Sprintf("Could not determine Private Certificate path: %s", err.Error()))
	}

	if c, err := self.CAPath(); err == nil {
		if _, err := os.Stat(c); err != nil {
			errors = append(errors, fmt.Sprintf("The CA %s does not exist", c))
		}
	} else {
		errors = append(errors, fmt.Sprintf("Could not determine CA path: %s", err.Error()))
	}

	if len(errors) == 0 {
		ok = true
	}

	return errors, ok
}

// SignString signs a message using a SHA256 PKCS1v15 protocol
func (self *Framework) SignString(str []byte) (signature []byte, err error) {
	pkpem, err := self.ClientPrivateKeyPEM()
	if err != nil {
		return
	}

	pk, err := x509.ParsePKCS1PrivateKey(pkpem.Bytes)
	if err != nil {
		err = fmt.Errorf("Could not parse private key PEM data: %s", err.Error())
		return
	}

	rng := rand.Reader
	hashed := sha256.Sum256(str)
	signature, err = rsa.SignPKCS1v15(rng, pk, crypto.SHA256, hashed[:])
	if err != nil {
		err = fmt.Errorf("Could not sign message: %s", err.Error())
	}

	return
}

// Certname determines the choria certname
func (self *Framework) Certname() string {
	if self.Config.OverrideCertname != "" {
		return self.Config.OverrideCertname
	}

	if certname, ok := os.LookupEnv("MCOLLECTIVE_CERTNAME"); ok {
		return certname
	}

	certname := self.Config.Identity

	if os.Getuid() != 0 {
		if u, ok := os.LookupEnv("USER"); ok {
			certname = fmt.Sprintf("%s.mcollective", u)
		}
	}

	return certname
}

// CAPath determines the path to the CA file
func (self *Framework) CAPath() (string, error) {
	ssl, err := self.SSLDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(ssl, "certs", "ca.pem"), nil
}

// ClientPrivateKeyPEM returns the PEM data for the Client Private Key
func (self *Framework) ClientPrivateKeyPEM() (pb *pem.Block, err error) {
	key, err := self.ClientPrivateKey()
	if err != nil {
		return pb, fmt.Errorf("Could not read Client Private Key PEM data: %s", err.Error())
	}

	keydat, err := ioutil.ReadFile(key)
	if err != nil {
		return pb, fmt.Errorf("Could not read Private Key %s: %s", key, err.Error())
	}

	pb, _ = pem.Decode(keydat)
	if pb == nil {
		return pb, fmt.Errorf("Failed to parse PEM data from key %s", key)
	}

	return
}

// ClientPublicCertPEM returns the PEM data for the Client Public Certificate
func (self *Framework) ClientPublicCertPEM() (pb *pem.Block, err error) {
	cert, err := self.ClientPublicCert()
	if err != nil {
		return pb, fmt.Errorf("Could not read Client Private Key PEM data: %s", err.Error())
	}

	certdat, err := ioutil.ReadFile(cert)
	if err != nil {
		err = fmt.Errorf("Could not read Public Certificate %s: %s", cert, err.Error())
	}

	pb, _ = pem.Decode(certdat)
	if pb == nil {
		return pb, fmt.Errorf("Failed to parse PEM data from certificate %s", cert)
	}

	return
}

// ClientPrivateKeyTXT reads the private key file as text
func (self *Framework) ClientPrivateKeyTXT() (cert []byte, err error) {
	file, err := self.ClientPrivateKey()
	if err != nil {
		return cert, fmt.Errorf("Could not read Client Private Key PEM data: %s", err.Error())
	}

	cert, err = ioutil.ReadFile(file)
	if err != nil {
		err = fmt.Errorf("Could not read Public Certificate %s: %s", cert, err.Error())
	}

	return
}

// ClientPrivateKey determines the location to the client cert
func (self *Framework) ClientPrivateKey() (string, error) {
	ssl, err := self.SSLDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(ssl, "private_keys", fmt.Sprintf("%s.pem", self.Certname())), nil
}

// ClientPublicCertTXT reads the public certificate file as text
func (self *Framework) ClientPublicCertTXT() (cert []byte, err error) {
	file, err := self.ClientPublicCert()
	if err != nil {
		return cert, fmt.Errorf("Could not read Client Public Certificate PEM data: %s", err.Error())
	}

	cert, err = ioutil.ReadFile(file)
	if err != nil {
		err = fmt.Errorf("Could not read Public Certificate %s: %s", cert, err.Error())
	}

	return
}

// ClientPublicCert determines the location to the client cert
func (self *Framework) ClientPublicCert() (string, error) {
	ssl, err := self.SSLDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(ssl, "certs", fmt.Sprintf("%s.pem", self.Certname())), nil
}

// SSLDir determines the AIO SSL directory
func (self *Framework) SSLDir() (string, error) {
	if !protocol.IsSecure() {
		return "/nonexisting", nil
	}

	if self.Config.Choria.SSLDir != "" {
		return self.Config.Choria.SSLDir, nil
	}

	if os.Getuid() == 0 {
		path, err := self.PuppetSetting("ssldir")
		if err != nil {
			return "", err
		}

		// store it so future calls to this wil not call out to Puppet again
		self.Config.Choria.SSLDir = path

		return path, nil
	}

	if os.Getenv("HOME") == "" {
		return "", fmt.Errorf("Cannot determine home dir while looking for SSL Directory, no HOME environment is set.  Please set HOME or configure plugin.choria.ssldir.")
	}

	return filepath.Join(os.Getenv("HOME"), ".puppetlabs", "etc", "puppet", "ssl"), nil
}

// ClientCertCacheDir determines the cache directory for client certs and creates it
// if it does not exist
func (self *Framework) ClientCertCacheDir() (string, error) {
	ssldir, err := self.SSLDir()
	if err != nil {
		return "", fmt.Errorf("Could not determine Client Certificate Cache Directory: %s", err.Error())
	}

	path := filepath.Join(ssldir, "choria_secuirty", "public_certs")

	err = os.MkdirAll(path, os.FileMode(int(0755)))
	if err != nil {
		return "", fmt.Errorf("Could not create Client Certificate Cache Directory: %s", err)
	}

	return path, nil
}

// TLSConfig creates a TLS configuration for use by NATS, HTTPS etc
func (self *Framework) TLSConfig() (tlsc *tls.Config, err error) {
	pub, _ := self.ClientPublicCert()
	pri, _ := self.ClientPrivateKey()
	ca, _ := self.CAPath()

	cert, err := tls.LoadX509KeyPair(pub, pri)
	if err != nil {
		err = errors.New("Could not load certificate " + pub + " and key " + pri + ": " + err.Error())
		return
	}

	caCert, err := ioutil.ReadFile(ca)

	if err != nil {
		return
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	tlsc = &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
		MinVersion:   tls.VersionTLS12,
	}

	if self.Config.DisableTLSVerify {
		tlsc.InsecureSkipVerify = true
	}

	tlsc.BuildNameToCertificate()

	return
}

// SSLContext creates a SSL context loaded with our certs and ca
func (self *Framework) SSLContext() (*http.Transport, error) {
	tlsConfig, err := self.TLSConfig()
	if err != nil {
		return &http.Transport{}, err
	}

	transport := &http.Transport{TLSClientConfig: tlsConfig}

	return transport, nil
}
