// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package v2

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/choria-io/go-choria/protocol"
)

// SecureReply contains 1 serialized Reply hashed
type SecureReply struct {
	// The protocol version for this secure reply `io.choria.protocol.v2.secure_reply` / protocol.SecureReplyV2
	Protocol string `json:"protocol"`
	// The reply held in the Secure Request
	MessageBody []byte `json:"reply"`
	// A sha256 of the reply
	Hash string `json:"hash"`
	// A signature made using the ed25519 seed of the sender
	Signature string `json:"signature"`
	// The JWT of the sending host
	SenderJWT string `json:"sender"`

	mu sync.Mutex
}

func (r *SecureReply) SetMessage(reply protocol.Reply) error {
	// TODO implement me
	panic("implement me")
}

func (r *SecureReply) Valid() bool {
	// TODO implement me
	panic("implement me")
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

func (r *SecureReply) Version() string {
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
