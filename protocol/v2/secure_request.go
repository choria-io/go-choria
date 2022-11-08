// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package v2

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/protocol"
	log "github.com/sirupsen/logrus"
)

// SecureRequest contains 1 serialized Request signed and with the related JWTs attached
type SecureRequest struct {
	// The protocol version for this secure request `io.choria.protocol.v2.secure_request` / protocol.SecureRequestV2
	Protocol protocol.ProtocolVersion `json:"protocol"`
	// The request held in the Secure Request
	MessageBody []byte `json:"request"`
	// A signature made using the ed25519 seed of the caller or signer
	Signature []byte `json:"signature"`
	// The JWT of the caller
	CallerJWT string `json:"caller"`
	// The JWT of the delegated signer, present when the AAA server is used
	SignerJWT string `json:"signer,omitempty"`

	security inter.SecurityProvider
	mu       sync.Mutex
}

// NewSecureRequest creates a choria:secure:request:1
func NewSecureRequest(request protocol.Request, security inter.SecurityProvider) (protocol.SecureRequest, error) {
	if security.BackingTechnology() != inter.SecurityTechnologyED25519JWT {
		return nil, ErrIncorrectProtocol
	}

	secure := &SecureRequest{
		Protocol: protocol.SecureRequestV2,
		security: security,
	}

	// TODO: we might choose to only support secure mode, but complicated with provisioning
	if protocol.IsSecure() {
		token, err := security.TokenBytes()
		if err != nil {
			return nil, err
		}

		secure.CallerJWT = string(token)
	}

	err := secure.SetMessage(request)
	if err != nil {
		return nil, err
	}

	return secure, nil
}

// NewRemoteSignedSecureRequest is a NewSecureRequest that delegates the signing to a remote signer like aaasvc
func NewRemoteSignedSecureRequest(request protocol.Request, security inter.SecurityProvider) (protocol.SecureRequest, error) {
	if security.BackingTechnology() != inter.SecurityTechnologyED25519JWT {
		return nil, ErrIncorrectProtocol
	}

	// no need for remote stuff, we don't do any signing or certs,
	// additionally the service hosting the remote signing service isnt
	// secured by choria protocol since at calling time the client does
	// not have a cert etc, but the request expects a signed JWT so that
	// provides the security of that request
	//
	// TODO: we might choose to only support secure mode, but complicated with provisioning, provisioning though shouldnt make requests
	if !protocol.IsSecure() || protocol.IsRemoteSignerAgent(request.Agent()) {
		return NewSecureRequest(request, security)
	}

	reqj, err := request.JSON()
	if err != nil {
		return nil, err
	}

	// TODO this should somehow be a passed in context but looks like quite big surgery
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	secj, err := security.RemoteSignRequest(ctx, reqj)
	if err != nil {
		return nil, err
	}

	secure := &SecureRequest{
		Protocol: protocol.SecureRequestV2,
		security: security,
	}

	err = json.Unmarshal(secj, &secure)
	if err != nil {
		return nil, fmt.Errorf("could not parse signed request: %s", err)
	}

	if secure.SignerJWT == "" {
		return nil, fmt.Errorf("remote signer did not set a signer JWT")
	}

	// We set our caller token here despite having done a delegated request because the
	// delegation would not know our token, so we just set this here which will not modify
	// the secure payload or its signature etc
	//
	// TODO: we might choose to only support secure mode, but complicated with provisioning
	if protocol.IsSecure() {
		token, err := security.TokenBytes()
		if err != nil {
			return nil, err
		}

		secure.CallerJWT = string(token)
	}

	return secure, nil
}

// NewSecureRequestFromTransport creates a new choria:secure:request:1 from the data contained in a Transport message
func NewSecureRequestFromTransport(message protocol.TransportMessage, security inter.SecurityProvider, skipvalidate bool) (protocol.SecureRequest, error) {
	if security.BackingTechnology() != inter.SecurityTechnologyED25519JWT {
		return nil, ErrIncorrectProtocol
	}

	secure := &SecureRequest{
		Protocol: protocol.SecureRequestV2,
		security: security,
	}

	data, err := message.Message()
	if err != nil {
		return nil, err
	}

	err = secure.IsValidJSON(data)
	if err != nil {
		return nil, fmt.Errorf("the JSON body from the TransportMessage is not a valid SecureRequest: %w", err)
	}

	err = json.Unmarshal(data, &secure)
	if err != nil {
		return nil, err
	}

	if !skipvalidate {
		if !secure.Valid() {
			return nil, fmt.Errorf("secure request messages created from Transport Message did not pass security validation")
		}
	}

	return secure, nil
}

func (r *SecureRequest) SetMessage(request protocol.Request) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	j, err := request.JSON()
	if err != nil {
		protocolErrorCtr.Inc()
		return fmt.Errorf("could not JSON encode reply message to store it in the Secure Request: %w", err)
	}

	r.Signature = []byte("insecure")

	// TODO: we might choose to only support secure mode, but complicated with provisioning
	if protocol.IsSecure() && !protocol.IsRemoteSignerAgent(request.Agent()) {
		sig, err := r.security.SignBytes(j)
		if err != nil {
			return err
		}
		r.Signature = sig
	}

	r.MessageBody = j

	return nil
}

func (r *SecureRequest) Valid() bool {
	// TODO: we might choose to only support secure mode, but complicated with provisioning
	if !protocol.IsSecure() {
		log.Debug("Bypassing validation on secure request due to build time flags")
		return true
	}

	jwts := [][]byte{[]byte(r.CallerJWT)}
	// delegated signatures
	if r.SignerJWT != "" {
		jwts = append(jwts, []byte(r.SignerJWT))
	}

	should, _ := r.security.VerifySignatureBytes(r.MessageBody, r.Signature, jwts...)
	if !should {
		log.Errorf("Signature in request did not pass validation")
		invalidCtr.Inc()
		return false
	}

	req, err := NewRequestFromSecureRequest(r)
	if err != nil {
		log.Errorf("Could not create Request to validate Secure Request with: %s", err)
		protocolErrorCtr.Inc()
		return false
	}

	_, err = r.security.ShouldAllowCaller(req.CallerID(), jwts...)
	if err != nil {
		log.Errorf("Caller verification failed: %s", err)
		protocolErrorCtr.Inc()
		return false
	}

	validCtr.Inc()

	return true
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

func (r *SecureRequest) Version() protocol.ProtocolVersion {
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

func (r *SecureRequest) SetSigner(signer []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.SignerJWT = string(signer)

	return nil
}
