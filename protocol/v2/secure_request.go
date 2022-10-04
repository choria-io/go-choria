// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package v2

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/protocol"
	log "github.com/sirupsen/logrus"
)

// SecureRequest contains 1 serialized Request signed and with the related JWTs attached
type SecureRequest struct {
	// The protocol version for this secure request `io.choria.protocol.v2.secure_request` / protocol.SecureRequestV2
	Protocol string `json:"protocol"`
	// The request held in the Secure Request
	MessageBody []byte `json:"request"`
	// A signature made using the ed25519 seed of the caller or signer
	Signature string `json:"signature"`
	// The JWT of the caller
	CallerJWT string `json:"caller"`
	// The JWT of the delegated signer, present when the AAA server is used
	SignerJWT string `json:"signer,omitempty"`

	mu sync.Mutex
}

// NewSecureRequest creates a choria:secure:request:1
func NewSecureRequest(request protocol.Request, security inter.SecurityProvider) (secure protocol.SecureRequest, err error) {
	if security.BackingTechnology() != inter.SecurityTechnologyED25519JWT {
		return nil, fmt.Errorf("version 2 protocol requires a ed25519+jwt based security system")
	}

	panic("implemented me")
}

// NewRemoteSignedSecureRequest is a NewSecureRequest that delegates the signing to a remote signer like aaasvc
func NewRemoteSignedSecureRequest(request protocol.Request, security inter.SecurityProvider) (secure protocol.SecureRequest, err error) {
	if security.BackingTechnology() != inter.SecurityTechnologyED25519JWT {
		return nil, fmt.Errorf("version 2 protocol requires a ed25519+jwt based security system")
	}

	panic("implemented me")
}

// NewSecureRequestFromTransport creates a new choria:secure:request:1 from the data contained in a Transport message
func NewSecureRequestFromTransport(message protocol.TransportMessage, security inter.SecurityProvider, skipvalidate bool) (secure protocol.SecureRequest, err error) {
	if security.BackingTechnology() != inter.SecurityTechnologyED25519JWT {
		return nil, fmt.Errorf("version 2 protocol requires a ed25519+jwt based security system")
	}

	panic("implemented me")
}

func (r *SecureRequest) SetMessage(request protocol.Request) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	j, err := request.JSON()
	if err != nil {
		protocolErrorCtr.Inc()
		return fmt.Errorf("could not JSON encode reply message to store it in the Secure Request: %s", err)
	}

	r.Signature = "insecure"

	// TODO: sign etc, support remove signers
	if protocol.IsSecure() {
		panic("signing not yet implemented")
	}

	r.MessageBody = j

	return nil
}

func (r *SecureRequest) Valid() bool {
	if !protocol.IsSecure() {
		log.Debug("Bypassing validation on secure request due to build time flags")
		return true
	}

	panic("signing not yet implemented")
}

func (r *SecureRequest) JSON() ([]byte, error) {
	r.mu.Lock()
	j, err := json.Marshal(r)
	r.mu.Unlock()
	if err != nil {
		protocolErrorCtr.Inc()
		return nil, fmt.Errorf("could not JSON Marshal: %s", err)
	}

	if err = r.IsValidJSON(j); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidJSON, err)
	}

	return j, nil
}

func (r *SecureRequest) Version() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.Protocol
}

func (r *SecureRequest) IsValidJSON(data []byte) error {
	_, errors, err := schemaValidate(protocol.SecureRequestV2, data)
	if err != nil {
		protocolErrorCtr.Inc()
		return fmt.Errorf("could not validate SecureRequest JSON data: %s", err)
	}

	if len(errors) != 0 {
		return fmt.Errorf("%w: %s", ErrInvalidJSON, strings.Join(errors, ", "))
	}

	return nil
}

func (r *SecureRequest) Message() []byte {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.MessageBody
}
