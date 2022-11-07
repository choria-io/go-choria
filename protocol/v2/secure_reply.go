// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package v2

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/protocol"
)

// SecureReply contains 1 serialized Reply hashed
type SecureReply struct {
	// The protocol version for this secure reply `io.choria.protocol.v2.secure_reply` / protocol.SecureReplyV2
	Protocol protocol.ProtocolVersion `json:"protocol"`
	// The reply held in the Secure Request
	MessageBody []byte `json:"reply"`
	// A sha256 of the reply
	Hash string `json:"hash"`
	// A signature made using the ed25519 seed of the sender
	Signature []byte `json:"signature"`
	// The JWT of the sending host
	SenderJWT string `json:"sender"`

	security inter.SecurityProvider
	mu       sync.Mutex
}

// NewSecureReply creates a io.choria.protocol.v2.secure_reply
func NewSecureReply(reply protocol.Reply, security inter.SecurityProvider) (protocol.SecureReply, error) {
	if security.BackingTechnology() != inter.SecurityTechnologyED25519JWT {
		return nil, fmt.Errorf("version 2 protocol requires a ed25519+jwt based security system")
	}

	secure := &SecureReply{
		Protocol: protocol.SecureReplyV2,
		security: security,
	}

	err := secure.SetMessage(reply)
	if err != nil {
		return nil, fmt.Errorf("could not set message on SecureReply structure: %s", err)
	}

	return secure, nil
}

// NewSecureReplyFromTransport creates a new io.choria.protocol.v2.secure_reply from the data contained in a Transport message
func NewSecureReplyFromTransport(message protocol.TransportMessage, security inter.SecurityProvider, skipvalidate bool) (protocol.SecureReply, error) {
	if security.BackingTechnology() != inter.SecurityTechnologyED25519JWT {
		return nil, fmt.Errorf("version 2 protocol requires a ed25519+jwt based security system")
	}

	secure := &SecureReply{
		Protocol: protocol.SecureReplyV2,
		security: security,
	}

	data, err := message.Message()
	if err != nil {
		return nil, err
	}

	err = secure.IsValidJSON(data)
	if err != nil {
		return nil, fmt.Errorf("the JSON body from the TransportMessage is not a valid SecureReply message: %s", err)
	}

	err = json.Unmarshal(data, &secure)
	if err != nil {
		return nil, err
	}

	if !skipvalidate {
		if !secure.Valid() {
			return nil, fmt.Errorf("SecureReply message created from the Transport Message is not valid")
		}
	}

	return secure, nil
}

func (r *SecureReply) SetMessage(reply protocol.Reply) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	j, err := reply.JSON()
	if err != nil {
		protocolErrorCtr.Inc()
		return fmt.Errorf("could not JSON encode reply: %v", err)
	}

	if r.security.ShouldSignReplies() {
		jwt, err := r.security.TokenBytes()
		if err != nil {
			return fmt.Errorf("could not read caller token: %v", err)
		}

		sig, err := r.security.SignBytes(j)
		if err != nil {
			return err
		}

		r.SenderJWT = string(jwt)
		r.Signature = sig
	}

	r.MessageBody = j
	r.Hash = base64.StdEncoding.EncodeToString(r.security.ChecksumBytes(j))

	return nil
}

func (r *SecureReply) Valid() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if base64.StdEncoding.EncodeToString(r.security.ChecksumBytes(r.MessageBody)) != r.Hash {
		invalidCtr.Inc()
		return false
	}

	validCtr.Inc()
	return true
}

func (r *SecureReply) JSON() ([]byte, error) {
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

func (r *SecureReply) Message() []byte {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.MessageBody
}

func (r *SecureReply) Version() protocol.ProtocolVersion {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.Protocol
}

func (r *SecureReply) IsValidJSON(data []byte) error {
	if !protocol.ClientStrictValidation {
		return nil
	}

	_, errors, err := schemaValidate(protocol.SecureReplyV2, data)
	if err != nil {
		return fmt.Errorf("could not validate SecureReply JSON data: %s", err)
	}

	if len(errors) != 0 {
		return fmt.Errorf("%w: %s", ErrInvalidJSON, strings.Join(errors, ", "))
	}

	return nil
}
