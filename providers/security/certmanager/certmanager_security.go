// Copyright (c) 2020-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package certmanagersec

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/choria-io/go-choria/backoff"
	"github.com/choria-io/go-choria/internal/util"
	"github.com/choria-io/go-choria/providers/security/filesec"
	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

// CertManagerSecurity implements a security provider that auto enrolls with Kubernetes Cert Manager
//
// It only supports being used inside a cluster and does not use the kubernetes API client libraries
// due to dependencies and just awfulness with go mod
type CertManagerSecurity struct {
	conf *Config
	log  *logrus.Entry

	ctx  context.Context
	fsec *filesec.FileSecurity
}

type Config struct {
	apiVersion           string
	altnames             []string
	namespace            string
	issuer               string
	identity             string
	replace              bool
	sslDir               string
	alwaysOverwriteCache bool
	privilegedUsers      []string
	csr                  string
	cert                 string
	key                  string
	ca                   string
	cache                string
	legacyCerts          bool
}

func New(opts ...Option) (*CertManagerSecurity, error) {
	cm := &CertManagerSecurity{}

	for _, opt := range opts {
		err := opt(cm)
		if err != nil {
			return nil, err
		}
	}

	if cm.conf == nil {
		return nil, fmt.Errorf("configuration not given")
	}

	if cm.log == nil {
		return nil, fmt.Errorf("logger not given")
	}

	if cm.ctx == nil {
		return nil, fmt.Errorf("context is required")
	}

	cm.conf.csr = filepath.Join(cm.conf.sslDir, "csr.pem")
	cm.conf.cert = filepath.Join(cm.conf.sslDir, "cert.pem")
	cm.conf.key = filepath.Join(cm.conf.sslDir, "key.pem")
	cm.conf.ca = filepath.Join(cm.conf.sslDir, "ca.pem")
	cm.conf.cache = filepath.Join(cm.conf.sslDir, "cache")

	return cm, cm.reinit()
}

func (cm *CertManagerSecurity) reinit() error {
	var err error

	fc := filesec.Config{
		Identity:                   cm.conf.identity,
		Certificate:                cm.conf.cert,
		Key:                        cm.conf.key,
		CA:                         cm.conf.ca,
		Cache:                      cm.conf.cache,
		PrivilegedUsers:            cm.conf.privilegedUsers,
		AlwaysOverwriteCache:       cm.conf.alwaysOverwriteCache,
		BackwardCompatVerification: cm.conf.legacyCerts,
	}

	cm.fsec, err = filesec.New(filesec.WithConfig(&fc), filesec.WithLog(cm.log))
	if err != nil {
		return err
	}

	if cm.shouldEnroll() {
		cm.log.Infof("Attempting to enroll with Cert Manager in namespace %q using issuer %q", cm.conf.namespace, cm.conf.issuer)
		err = cm.Enroll(cm.ctx, time.Minute, func(_ string, i int) {
			cm.log.Infof("Enrollment attempt %d", i)
		})
		if err != nil {
			return fmt.Errorf("enrollment failed: %s", err)
		}

		cm.log.Infof("Enrollment with Cert Manager completed in namespace %q", cm.conf.namespace)
	}

	return nil
}

func (cm *CertManagerSecurity) Enroll(ctx context.Context, wait time.Duration, cb func(digest string, try int)) error {
	if !cm.shouldEnroll() {
		cm.log.Infof("Enrollment already completed, remove %q to force re-enrolment", cm.conf.sslDir)
		return nil
	}

	err := cm.createSSLDirectories()
	if err != nil {
		return fmt.Errorf("could not initialize ssl directories: %s", err)
	}

	var key *rsa.PrivateKey
	if !cm.privateKeyExists() {
		cm.log.Debugf("Creating a new Private Key %s", cm.Identity())

		key, err = cm.writePrivateKey()
		if err != nil {
			return fmt.Errorf("could not write a new private key: %s", err)
		}
	}

	if !cm.csrExists() {
		cm.log.Debugf("Creating a new CSR for %s", cm.Identity())

		err = cm.writeCSR(key, cm.Identity(), "choria.io")
		if err != nil {
			return fmt.Errorf("could not write CSR: %s", err)
		}
	}

	if !cm.publicCertExists() || cm.conf.replace {
		err = cm.processCSR()
		if err != nil {
			return fmt.Errorf("csr submission failed: %s", err)
		}
	}

	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	err = backoff.Default.For(ctx, func(try int) error {
		cm.log.Infof("Attempt %d at fetching certificate %q", try, cm.Identity())
		return cm.fetchCertAndCA()
	})
	if err != nil {
		return err
	}

	return nil
}

func (cm *CertManagerSecurity) Provider() string {
	return "certmanager"
}

func (cm *CertManagerSecurity) fetchCertAndCA() error {
	url := fmt.Sprintf("https://kubernetes.default.svc/apis/cert-manager.io/%s/namespaces/%s/certificaterequests/%s", cm.conf.apiVersion, cm.conf.namespace, cm.Identity())
	resp, err := cm.k8sRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("could not load CSR for %q: %s", cm.Identity(), err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("could not load CSR for %q: %s", cm.Identity(), err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("could not load CSR for %q: code: %d body: %q", cm.Identity(), resp.StatusCode, body)
	}

	ca := gjson.GetBytes(body, "status.ca")
	if !ca.Exists() {
		return fmt.Errorf("did not receive a CA from Cert Manager")
	}

	capem, err := base64.StdEncoding.DecodeString(ca.String())
	if err != nil {
		return err
	}

	err = os.WriteFile(cm.conf.ca, capem, 0644)
	if err != nil {
		return fmt.Errorf("could not write ca %s: %s", cm.conf.ca, err)
	}

	cert := gjson.GetBytes(body, "status.certificate")
	if !cert.Exists() {
		return fmt.Errorf("did not receive a certificate from Cert Manager")
	}

	certpem, err := base64.StdEncoding.DecodeString(cert.String())
	if err != nil {
		return err
	}

	err = os.WriteFile(cm.conf.cert, certpem, 0644)
	if err != nil {
		return fmt.Errorf("could not write certificate %s: %s", cm.conf.ca, err)
	}

	return nil
}

func (cm *CertManagerSecurity) processCSR() error {
	code, body, err := cm.submitCSR()
	if err != nil {
		return err
	}

	switch code {
	case 201:
		// ok
	case 409:
		if !cm.conf.replace {
			return fmt.Errorf("found an existing CSR for %q", cm.Identity())
		}

		cm.log.Warnf("Found an existing CSR for %q, removing and creating a new one", cm.Identity())
		code, err := cm.deleteCSR()
		if err != nil {
			return fmt.Errorf("deleting existing CSR for %q failed: %s", cm.Identity(), err)
		}

		if code != 200 {
			return fmt.Errorf("deleting existing CSR for %q failed: code %d", cm.Identity(), code)
		}

		code, body, err = cm.submitCSR()
		if err != nil {
			return fmt.Errorf("csr creation failed: %s", err)
		}

		if code != 201 {
			return fmt.Errorf("csr creation failed: code: %d body: %q", code, body)
		}

	default:
		return fmt.Errorf("unexpected error from the Kubernetes API: code: %d body: %q", code, body)
	}

	return nil
}

func (cm *CertManagerSecurity) deleteCSR() (code int, err error) {
	url := fmt.Sprintf("https://kubernetes.default.svc/apis/cert-manager.io/%s/namespaces/%s/certificaterequests/%s", cm.conf.apiVersion, cm.conf.namespace, cm.Identity())
	resp, err := cm.k8sRequest("DELETE", url, nil)
	if err != nil {
		return 500, err
	}

	return resp.StatusCode, nil
}

func (cm *CertManagerSecurity) submitCSR() (code int, body []byte, err error) {
	csr, err := cm.csrTXT()
	if err != nil {
		return 500, nil, fmt.Errorf("could not read CSR: %s", err)
	}

	csrReq := map[string]any{
		"apiVersion": fmt.Sprintf("cert-manager.io/%s", cm.conf.apiVersion),
		"kind":       "CertificateRequest",
		"metadata": map[string]any{
			"name":      cm.Identity(),
			"namespace": cm.conf.namespace,
		},
		"spec": map[string]any{
			"issuerRef": map[string]any{
				"name": cm.conf.issuer,
			},
			"csr":     csr,
			"request": csr,
		},
	}

	jreq, err := json.Marshal(csrReq)
	if err != nil {
		return 500, nil, err
	}

	cm.log.Infof("Submitting CSR for %q to Cert Manager", cm.Identity())

	url := fmt.Sprintf("https://kubernetes.default.svc/apis/cert-manager.io/%s/namespaces/%s/certificaterequests", cm.conf.apiVersion, cm.conf.namespace)
	resp, err := cm.k8sRequest("POST", url, bytes.NewReader(jreq))
	if err != nil {
		return 500, nil, err
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	return resp.StatusCode, body, err
}

func (cm *CertManagerSecurity) k8sRequest(method string, url string, body io.Reader) (*http.Response, error) {
	token, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return nil, err
	}

	tlsConfig, err := cm.k8sTLSConfig()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(cm.ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+string(token))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{
		Transport: &http.Transport{TLSClientConfig: tlsConfig},
	}

	return client.Do(req)
}

func (cm *CertManagerSecurity) k8sTLSConfig() (*tls.Config, error) {
	tlsc := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		PreferServerCipherSuites: true,
	}

	ca, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/ca.crt")
	if err != nil {
		return nil, err
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(ca) {
		return nil, fmt.Errorf("could not add kubernetes CA to the cert pool")
	}

	tlsc.ClientCAs = pool
	tlsc.RootCAs = pool

	return tlsc, nil
}

func (cm *CertManagerSecurity) csrTXT() ([]byte, error) {
	return os.ReadFile(cm.conf.csr)
}

func (cm *CertManagerSecurity) shouldEnroll() bool {
	// TODO re-enroll when expired
	return !(cm.privateKeyExists() && cm.caExists() && cm.publicCertExists())
}

func (cm *CertManagerSecurity) writePrivateKey() (*rsa.PrivateKey, error) {
	if cm.privateKeyExists() {
		return nil, fmt.Errorf("a private key already exist for %s", cm.Identity())
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("could not generate rsa key: %cm", err)
	}

	pemdata := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(key),
		},
	)

	err = os.WriteFile(cm.conf.key, pemdata, 0640)
	if err != nil {
		return nil, fmt.Errorf("could not write private key: %cm", err)
	}

	return key, nil
}

func (cm *CertManagerSecurity) writeCSR(key *rsa.PrivateKey, cn string, ou string) error {
	if cm.csrExists() {
		return fmt.Errorf("a certificate request already exist for %s", cm.Identity())
	}

	path := cm.conf.csr

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

	template.DNSNames = append(template.DNSNames, cm.conf.altnames...)

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

func (cm *CertManagerSecurity) createSSLDirectories() error {
	err := os.MkdirAll(cm.conf.sslDir, 0771)
	if err != nil {
		return err
	}

	err = os.MkdirAll(cm.conf.cache, 0700)
	if err != nil {
		return err
	}

	return nil
}

func (cm *CertManagerSecurity) csrExists() bool {
	return util.FileExist(cm.conf.csr)
}

func (cm *CertManagerSecurity) privateKeyExists() bool {
	return util.FileExist(cm.conf.key)
}

func (cm *CertManagerSecurity) publicCertExists() bool {
	return util.FileExist(cm.conf.cert)
}

func (cm *CertManagerSecurity) caExists() bool {
	return util.FileExist(cm.conf.ca)
}

func (cm *CertManagerSecurity) Validate() (errs []string, ok bool) {
	if !util.FileIsDir(cm.conf.sslDir) {
		errs = append(errs, fmt.Sprintf("%s does not exist or is not a directory", cm.conf.sslDir))
	}

	if !util.FileIsDir(cm.conf.cache) {
		errs = append(errs, fmt.Sprintf("%s does not exist or is not a diectory", cm.conf.cache))
	}

	return errs, len(errs) == 0
}

func (cm *CertManagerSecurity) Identity() string {
	return cm.conf.identity
}

func (cm *CertManagerSecurity) CallerName() string {
	return cm.fsec.CallerName()
}

func (cm *CertManagerSecurity) CallerIdentity(caller string) (string, error) {
	return cm.fsec.CallerIdentity(caller)
}

func (cm *CertManagerSecurity) SignBytes(b []byte) (signature []byte, err error) {
	return cm.fsec.SignBytes(b)
}

func (cm *CertManagerSecurity) VerifyByteSignature(str []byte, signature []byte, identity string) bool {
	return cm.fsec.VerifyByteSignature(str, signature, identity)
}

func (cm *CertManagerSecurity) SignString(s string) (signature []byte, err error) {
	return cm.fsec.SignString(s)
}

func (cm *CertManagerSecurity) RemoteSignRequest(ctx context.Context, str []byte) (signed []byte, err error) {
	return cm.fsec.RemoteSignRequest(ctx, str)
}

func (cm *CertManagerSecurity) IsRemoteSigning() bool {
	return cm.fsec.IsRemoteSigning()
}

func (cm *CertManagerSecurity) VerifyStringSignature(str string, signature []byte, identity string) bool {
	return cm.fsec.VerifyStringSignature(str, signature, identity)
}

func (cm *CertManagerSecurity) PrivilegedVerifyByteSignature(dat []byte, sig []byte, identity string) bool {
	return cm.fsec.PrivilegedVerifyByteSignature(dat, sig, identity)
}

func (cm *CertManagerSecurity) PrivilegedVerifyStringSignature(dat string, sig []byte, identity string) bool {
	return cm.fsec.PrivilegedVerifyStringSignature(dat, sig, identity)
}

func (cm *CertManagerSecurity) ChecksumBytes(data []byte) []byte {
	return cm.fsec.ChecksumBytes(data)
}

func (cm *CertManagerSecurity) ChecksumString(data string) []byte {
	return cm.fsec.ChecksumString(data)
}

func (cm *CertManagerSecurity) ClientTLSConfig() (*tls.Config, error) {
	return cm.fsec.ClientTLSConfig()
}

func (cm *CertManagerSecurity) TLSConfig() (*tls.Config, error) {
	return cm.fsec.TLSConfig()
}

func (cm *CertManagerSecurity) SSLContext() (*http.Transport, error) {
	return cm.fsec.SSLContext()
}

func (cm *CertManagerSecurity) HTTPClient(secure bool) (*http.Client, error) {
	return cm.fsec.HTTPClient(secure)
}

func (cm *CertManagerSecurity) VerifyCertificate(certpem []byte, identity string) error {
	return cm.fsec.VerifyCertificate(certpem, identity)
}

func (cm *CertManagerSecurity) PublicCert() (*x509.Certificate, error) {
	return cm.fsec.PublicCert()
}

func (cm *CertManagerSecurity) PublicCertPem() (*pem.Block, error) {
	return cm.fsec.PublicCertPem()
}

func (cm *CertManagerSecurity) PublicCertTXT() ([]byte, error) {
	return cm.fsec.PublicCertTXT()
}

func (cm *CertManagerSecurity) CachePublicData(data []byte, identity string) error {
	return cm.fsec.CachePublicData(data, identity)
}

func (cm *CertManagerSecurity) CachedPublicData(identity string) ([]byte, error) {
	return cm.fsec.CachedPublicData(identity)
}
