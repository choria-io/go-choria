package mcollective

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
	"os/user"
	"path/filepath"
)

// SignString signs a message using a SHA256 PKCS1v15 protocol
func (c *Choria) SignString(str []byte) (signature []byte, err error) {
	pkpem, err := c.ClientPrivateKeyPEM()
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
func (c *Choria) Certname() string {
	certname := c.Config.Identity

	currentUser, _ := user.Current()

	if currentUser.Uid != "0" {
		if u, ok := os.LookupEnv("USER"); ok {
			certname = fmt.Sprintf("%s.mcollective", u)
		}
	}

	if u, ok := os.LookupEnv("MCOLLECTIVE_CERTNAME"); ok {
		certname = u
	}

	return certname
}

// CAPath determines the path to the CA file
func (c *Choria) CAPath() (string, error) {
	ssl, err := c.SSLDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(ssl, "certs", "ca.pem"), nil
}

// ClientPrivateKeyPEM returns the PEM data for the Client Private Key
func (c *Choria) ClientPrivateKeyPEM() (pb *pem.Block, err error) {
	key, err := c.ClientPrivateKey()
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
func (c *Choria) ClientPublicCertPEM() (pb *pem.Block, err error) {
	cert, err := c.ClientPublicCert()
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
func (c *Choria) ClientPrivateKeyTXT() (cert []byte, err error) {
	file, err := c.ClientPrivateKey()
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
func (c *Choria) ClientPrivateKey() (string, error) {
	ssl, err := c.SSLDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(ssl, "private_keys", fmt.Sprintf("%s.pem", c.Certname())), nil
}

// ClientPublicCertTXT reads the public certificate file as text
func (c *Choria) ClientPublicCertTXT() (cert []byte, err error) {
	file, err := c.ClientPublicCert()
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
func (c *Choria) ClientPublicCert() (string, error) {
	ssl, err := c.SSLDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(ssl, "certs", fmt.Sprintf("%s.pem", c.Certname())), nil
}

// SSLDir determines the AIO SSL directory
func (c *Choria) SSLDir() (string, error) {
	if c.Config.Choria.SSLDir != "" {
		return c.Config.Choria.SSLDir, nil
	}

	u, _ := user.Current()
	if u.Uid == "0" {
		path, err := c.PuppetSetting("ssldir")
		if err != nil {
			return "", err
		}

		return path, nil
	}

	return filepath.Join(u.HomeDir, ".puppetlabs", "etc", "puppet", "ssl"), nil
}

// SSLContext creates a SSL context loaded with our certs and ca
func (c *Choria) SSLContext() (*http.Transport, error) {
	pub, _ := c.ClientPublicCert()
	pri, _ := c.ClientPrivateKey()
	ca, _ := c.CAPath()

	cert, err := tls.LoadX509KeyPair(pub, pri)
	if err != nil {
		return &http.Transport{}, errors.New("Could not load certificate " + pub + " and key " + pri + ": " + err.Error())
	}

	caCert, err := ioutil.ReadFile(ca)

	if err != nil {
		return &http.Transport{}, err
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
		MinVersion:   tls.VersionTLS12,
	}

	tlsConfig.BuildNameToCertificate()

	transport := &http.Transport{TLSClientConfig: tlsConfig}

	return transport, nil
}
