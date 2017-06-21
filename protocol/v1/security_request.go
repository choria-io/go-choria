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
	"strings"
	"sync"

	"github.com/choria-io/go-choria/protocol"
)

// SecureRequest contains 1 serialized Request signed and with the public cert attached
type secureRequest struct {
	Protocol          string `json:"protocol"`
	MessageBody       string `json:"message"`
	Signature         string `json:"signature"`
	PublicCertificate string `json:"pubcert"`

	publicCertPath  string
	privateCertPath string
	mu              sync.Mutex
}

// SetMessage sets the message contained in the Request and updates the signature
func (r *secureRequest) SetMessage(request protocol.Request) (err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	j, err := request.JSON()
	if err != nil {
		err = fmt.Errorf("Could not JSON encode reply message to store it in the Secure Request: %s", err.Error())
		return
	}

	signature, err := r.signString([]byte(j))
	if err != nil {
		err = fmt.Errorf("Could not sign message string: %s", err.Error())
		return
	}

	r.MessageBody = string(j)
	r.Signature = base64.StdEncoding.EncodeToString(signature)

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

	signature, err := r.signString([]byte(r.MessageBody))
	if err != nil {
		return false
	}

	if base64.StdEncoding.EncodeToString(signature) != r.Signature {
		return false
	}

	return true
}

// JSON creates a JSON encoded request
func (r *secureRequest) JSON() (body string, err error) {
	j, err := json.Marshal(r)
	if err != nil {
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
		err = fmt.Errorf("Could not validate SecureRequest JSON data: %s", err.Error())
		return
	}

	if len(errors) != 0 {
		err = fmt.Errorf("Supplied JSON document is not a valid SecureRequest message: %s", strings.Join(errors, ", "))
		return
	}

	return
}

func (r *secureRequest) privateKeyPEM() (pb *pem.Block, err error) {
	keydat, err := readFile(r.privateCertPath)
	if err != nil {
		return pb, fmt.Errorf("Could not read Private Key %s: %s", r.privateCertPath, err.Error())
	}

	pb, _ = pem.Decode(keydat)
	if pb == nil {
		return pb, fmt.Errorf("Failed to parse PEM data from key %s", r.privateCertPath)
	}

	return
}

func (r *secureRequest) signString(str []byte) (signature []byte, err error) {
	pkpem, err := r.privateKeyPEM()
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

func readFile(path string) (cert []byte, err error) {
	cert, err = ioutil.ReadFile(path)
	if err != nil {
		err = fmt.Errorf("Could not read file %s: %s", path, err.Error())
	}

	return
}
