package v1

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
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

	security SecurityProvider
	mu       sync.Mutex
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

		signature, err = r.security.SignString(j)
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

	req, err := NewRequestFromSecureRequest(r)
	if err != nil {
		log.Errorf("Could not create Request to validate Secure Request with: %s", err)
		protocolErrorCtr.Inc()
		return false
	}

	certname, err := r.security.CallerIdentity(req.CallerID())
	if err != nil {
		log.Errorf("Could not extract certname from caller: %s", err.Error())
		protocolErrorCtr.Inc()
		return false
	}

	err = r.security.CachePublicData([]byte(r.PublicCertificate), certname)
	if err != nil {
		log.Errorf("Could not cache Client Certificate: %s", err.Error())
		protocolErrorCtr.Inc()
		return false
	}

	sig, err := base64.StdEncoding.DecodeString(r.Signature)
	if err != nil {
		log.Errorf("Could not bas64 decode signature: %s", err)
		protocolErrorCtr.Inc()
		return false
	}

	if !r.security.PrivilegedVerifyStringSignature(r.MessageBody, sig, certname) {
		invalidCtr.Inc()
		return false
	}

	validCtr.Inc()
	return true
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
