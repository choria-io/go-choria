package v1

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/choria-io/go-protocol/protocol"
	log "github.com/sirupsen/logrus"
)

// SecureRequest contains 1 serialized Request signed and with the public cert attached
type secureRequest struct {
	Protocol          string `json:"protocol"`
	MessageBody       string `json:"message"`
	Signature         string `json:"signature"`
	PublicCertificate string `json:"pubcert"`

	publicCertPath  string
	privateCertPath string
	caPath          string
	cachePath       string

	whilelistRegex  []string
	privilegedRegex []string

	mu sync.Mutex
}

// SetMessage sets the message contained in the Request and updates the signature
func (r *secureRequest) SetMessage(request protocol.Request) (err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	j, err := request.JSON()
	if err != nil {
		protocolErrorCtr.Inc()
		err = fmt.Errorf("Could not JSON encode reply message to store it in the Secure Request: %s", err.Error())
		return
	}

	r.Signature = "insecure"

	if protocol.IsSecure() {
		var signature []byte

		signature, err = r.signString([]byte(j))
		if err != nil {
			err = fmt.Errorf("Could not sign message string: %s", err.Error())
			return
		}
		r.Signature = base64.StdEncoding.EncodeToString(signature)
	}

	r.MessageBody = string(j)

	return
}

// Message retrieves the stored message.  It will be a JSON encoded version of the request set via SetMessage
func (r *secureRequest) Message() string {
	return r.MessageBody
}

// Valid determines if the request is valid
func (r *secureRequest) Valid() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !protocol.IsSecure() {
		log.Debug("Bypassing validation on secure request due to build time flags")
		return true
	}

	if r.cachePath == "" || r.caPath == "" {
		log.Debug("SecureRequest validation failed - no cache path or ca path have been set")
		protocolErrorCtr.Inc()
		return false
	}

	cachedpath, err := r.cacheClientCert()
	if err != nil {
		log.Errorf("Could not cache Client Certificate: %s", err.Error())
		protocolErrorCtr.Inc()
		return false
	}

	if cachedpath == "" {
		log.Errorf("Could not cache Client Certificate, no cache file was created")
		protocolErrorCtr.Inc()
		return false
	}

	candidateCerts := append([]string{cachedpath}, r.privilegedCerts()...)

	body := []byte(r.MessageBody)
	sig := []byte(r.Signature)

	for _, candidate := range candidateCerts {
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			continue
		}

		if r.verifySignature(body, sig, candidate) {
			log.Debugf("Secure Request signature verified using %s", candidate)
			validCtr.Inc()
			return true
		}

		log.Debugf("Secure Request signature could not be verified using %s", candidate)
	}

	invalidCtr.Inc()
	return false
}

// JSON creates a JSON encoded request
func (r *secureRequest) JSON() (body string, err error) {
	j, err := json.Marshal(r)
	if err != nil {
		protocolErrorCtr.Inc()
		err = fmt.Errorf("Could not JSON Marshal: %s", err.Error())
		return
	}

	body = string(j)

	if err = r.IsValidJSON(body); err != nil {
		err = fmt.Errorf("JSON produced from the SecureRequest does not pass validation: %s", err.Error())
		return
	}

	return
}

// Version retreives the protocol version for this message
func (r *secureRequest) Version() string {
	return r.Protocol
}

// IsValidJSON validates the given JSON data against the schema
func (r *secureRequest) IsValidJSON(data string) (err error) {
	_, errors, err := schemas.Validate(schemas.SecureRequestV1, data)
	if err != nil {
		protocolErrorCtr.Inc()
		err = fmt.Errorf("Could not validate SecureRequest JSON data: %s", err.Error())
		return
	}

	if len(errors) != 0 {
		err = fmt.Errorf("Supplied JSON document is not a valid SecureRequest message: %s", strings.Join(errors, ", "))
		return
	}

	return
}

func (r *secureRequest) privilegedCerts() []string {
	certs := []string{}

	filepath.Walk(r.cachePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			cert := []byte(strings.TrimSuffix(filepath.Base(path), ".pem"))

			if r.matchAnyRegex(cert, r.privilegedRegex) {
				certs = append(certs, path)
			}
		}

		return nil
	})

	sort.Strings(certs)

	return certs
}

func (r *secureRequest) matchAnyRegex(str []byte, regex []string) bool {
	for _, reg := range regex {
		if matched, _ := regexp.Match(reg, str); matched {
			return true
		}
	}

	return false
}

func (r *secureRequest) cacheClientCert() (string, error) {
	req, err := NewRequestFromSecureRequest(r)
	if err != nil {
		log.Errorf("Could not create Request to validate Secure Request with: %s", err.Error())
		protocolErrorCtr.Inc()
		return "", err
	}

	certname, err := r.requestCallerCertname(req.CallerID())
	if err != nil {
		log.Errorf("Could not extract certname from caller: %s", err.Error())
		protocolErrorCtr.Inc()
		return "", err
	}

	certfile := filepath.Join(r.cachePath, fmt.Sprintf("%s.pem", certname))

	if _, err := os.Stat(certfile); !os.IsNotExist(err) {
		return certfile, nil
	}

	if !r.shouldCacheClientCert(certname) {
		return "", fmt.Errorf("Certificate %s did not pass validation", certname)
	}

	err = ioutil.WriteFile(certfile, []byte(r.PublicCertificate), os.FileMode(int(0644)))
	if err != nil {
		protocolErrorCtr.Inc()
		return "", fmt.Errorf("Could not cache client public certificate: %s", err.Error())
	}

	return certfile, nil
}

func (r *secureRequest) shouldCacheClientCert(name string) bool {
	if !r.verifyCert([]byte(r.PublicCertificate), "") {
		return false
	}

	if r.matchAnyRegex([]byte(name), r.privilegedRegex) {
		log.Warnf("Caching privileged certificate %s", name)
		return true
	}

	if !r.verifyCert([]byte(r.PublicCertificate), name) {
		return false
	}

	if !r.matchAnyRegex([]byte(name), r.whilelistRegex) {
		log.Warnf("Received certificate '%s' does not match the allowed list '%s'", name, r.whilelistRegex)
		return false
	}

	return true
}

// verifies a certificate is signed with the configured CA and if
// name is not "" that it matches the name given
func (r *secureRequest) verifyCert(certpem []byte, name string) bool {
	capem, err := ioutil.ReadFile(r.caPath)
	if err != nil {
		log.Errorf("Could not read CA '%s': %s", r.caPath, err.Error())
		protocolErrorCtr.Inc()
		return false
	}

	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(capem) {
		log.Warnf("Could not use CA '%s' as PEM data: %s", r.caPath, err.Error())
		protocolErrorCtr.Inc()
		return false
	}

	block, _ := pem.Decode(certpem)
	if block == nil {
		log.Warnf("Could not decode certificate '%s' PEM data: %s", name, err.Error())
		protocolErrorCtr.Inc()
		return false
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		log.Warnf("Could not parse certificate '%s': %s", name, err.Error())
		protocolErrorCtr.Inc()
		return false
	}

	opts := x509.VerifyOptions{
		Roots: roots,
	}

	if name != "" {
		opts.DNSName = name
	}

	_, err = cert.Verify(opts)
	if err != nil {
		invalidCertificateCtr.Inc()
		log.Warnf("Certificate does not pass verification as '%s': %s", name, err.Error())
		return false
	}

	return true
}

func (r *secureRequest) requestCallerCertname(caller string) (string, error) {
	re := regexp.MustCompile("^choria=([\\w\\.\\-]+)")
	match := re.FindStringSubmatch(caller)

	if match == nil {
		protocolErrorCtr.Inc()
		return "", fmt.Errorf("Could not find a valid certificate name in %s", caller)
	}

	return match[1], nil
}

func (r *secureRequest) decodePEM(certpath string) (pb *pem.Block, err error) {
	if certpath == "" {
		certpath = r.privateCertPath
	}

	keydat, err := readFile(certpath)
	if err != nil {
		protocolErrorCtr.Inc()
		return pb, fmt.Errorf("Could not read PEM data from %s: %s", certpath, err.Error())
	}

	pb, _ = pem.Decode(keydat)
	if pb == nil {
		protocolErrorCtr.Inc()
		return pb, fmt.Errorf("Failed to parse PEM data from key %s", certpath)
	}

	return
}

func (r *secureRequest) signString(str []byte) (signature []byte, err error) {
	pkpem, err := r.decodePEM("")
	if err != nil {
		return
	}

	pk, err := x509.ParsePKCS1PrivateKey(pkpem.Bytes)
	if err != nil {
		protocolErrorCtr.Inc()
		err = fmt.Errorf("Could not parse private key PEM data: %s", err.Error())
		return
	}

	rng := rand.Reader
	hashed := sha256.Sum256(str)
	signature, err = rsa.SignPKCS1v15(rng, pk, crypto.SHA256, hashed[:])
	if err != nil {
		protocolErrorCtr.Inc()
		err = fmt.Errorf("Could not sign message: %s", err.Error())
	}

	return
}

func (r *secureRequest) verifySignature(str []byte, sig []byte, pubkeyPath string) bool {
	pkpem, err := r.decodePEM(pubkeyPath)
	if err != nil {
		protocolErrorCtr.Inc()
		log.Errorf("Could not decode PEM data in public key %s: %s", pubkeyPath, err.Error())
		return false
	}

	cert, err := x509.ParseCertificate(pkpem.Bytes)
	if err != nil {
		protocolErrorCtr.Inc()
		log.Errorf("Could not parse decoded PEM data for public key %s: %s", pubkeyPath, err.Error())
		return false
	}

	rsaPublicKey := cert.PublicKey.(*rsa.PublicKey)
	hashed := sha256.Sum256(str)

	decodedsig, err := base64.StdEncoding.DecodeString(string(sig))
	if err != nil {
		protocolErrorCtr.Inc()
		log.Errorf("Could not decode signature base64 encoding: %s", err.Error())
		return false
	}

	err = rsa.VerifyPKCS1v15(rsaPublicKey, crypto.SHA256, hashed[:], decodedsig)
	if err != nil {
		log.Errorf("Verification using %s failed: %s", pubkeyPath, err.Error())
		return false
	}

	return true
}

func readFile(path string) (cert []byte, err error) {
	cert, err = ioutil.ReadFile(path)
	if err != nil {
		protocolErrorCtr.Inc()
		err = fmt.Errorf("Could not read file %s: %s", path, err.Error())
	}

	return
}
