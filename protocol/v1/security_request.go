// Copyright (c) 2017-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/protocol"
	log "github.com/sirupsen/logrus"
)

// SecureRequest contains 1 serialized Request signed and with the public cert attached
type SecureRequest struct {
	Protocol          string `json:"protocol"`
	MessageBody       string `json:"message"`
	Signature         string `json:"signature"`
	PublicCertificate string `json:"pubcert"`

	security inter.SecurityProvider
	mu       sync.Mutex
}

// SetMessage sets the message contained in the Request and updates the signature
func (r *SecureRequest) SetMessage(request protocol.Request) (err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	j, err := request.JSON()
	if err != nil {
		protocolErrorCtr.Inc()
		return fmt.Errorf("could not JSON encode reply message to store it in the Secure Request: %s", err)
	}

	r.Signature = "insecure"

	if protocol.IsSecure() && !protocol.IsRemoteSignerAgent(request.Agent()) {
		var signature []byte

		signature, err = r.security.SignBytes(j)
		if err != nil {
			// registration when doing anon tls might not have a certificate - so we allow that to go unsigned
			if !protocol.IsRegistrationAgent(request.Agent()) {
				return fmt.Errorf("could not sign message string: %s", err)
			}
			signature = []byte("insecure registration")
		}

		r.Signature = base64.StdEncoding.EncodeToString(signature)
	}

	r.MessageBody = string(j)

	return nil
}

// Message retrieves the stored message.  It will be a JSON encoded version of the request set via SetMessage
func (r *SecureRequest) Message() []byte {
	r.mu.Lock()
	defer r.mu.Unlock()

	return []byte(r.MessageBody)
}

// Valid determines if the request is valid
func (r *SecureRequest) Valid() bool {
	// should not be locked

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
		log.Errorf("Could not extract certname from caller: %s", err)
		protocolErrorCtr.Inc()
		return false
	}

	_, err = r.security.ShouldAllowCaller([]byte(r.PublicCertificate), certname)
	if err != nil {
		log.Errorf("Client Certificate verification failed: %s", err)
		protocolErrorCtr.Inc()
		return false
	}

	sig, err := base64.StdEncoding.DecodeString(r.Signature)
	if err != nil {
		log.Errorf("Could not bas64 decode signature: %s", err)
		protocolErrorCtr.Inc()
		return false
	}

	should, _ := r.security.VerifyByteSignature([]byte(r.MessageBody), sig, []byte(r.PublicCertificate))
	if !should {
		log.Errorf("Signature in request did not pass validation with embedded public certificate")
		invalidCtr.Inc()
		return false
	}

	validCtr.Inc()

	return true
}

// JSON creates a JSON encoded request
func (r *SecureRequest) JSON() ([]byte, error) {
	r.mu.Lock()
	j, err := json.Marshal(r)
	r.mu.Unlock()
	if err != nil {
		protocolErrorCtr.Inc()
		return nil, fmt.Errorf("could not JSON Marshal: %s", err)
	}

	if err = r.IsValidJSON(j); err != nil {
		return nil, fmt.Errorf("the JSON produced from the SecureRequest does not pass validation: %s", err)
	}

	return j, nil
}

// Version retrieves the protocol version for this message
func (r *SecureRequest) Version() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.Protocol
}

// IsValidJSON validates the given JSON data against the schema
func (r *SecureRequest) IsValidJSON(data []byte) (err error) {
	_, errors, err := schemaValidate(secureRequestSchema, data)
	if err != nil {
		protocolErrorCtr.Inc()
		return fmt.Errorf("could not validate SecureRequest JSON data: %s", err)
	}

	if len(errors) != 0 {
		return fmt.Errorf("supplied JSON document is not a valid SecureRequest message: %s", strings.Join(errors, ", "))
	}

	return nil
}
