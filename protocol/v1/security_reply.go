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
)

// SecureReply contains 1 serialized Reply hashed
type secureReply struct {
	Protocol    string `json:"protocol"`
	MessageBody string `json:"message"`
	Hash        string `json:"hash"`

	security inter.SecurityProvider

	mu sync.Mutex
}

// SetMessage sets the message contained in the Reply and updates the hash
func (r *secureReply) SetMessage(reply protocol.Reply) (err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	j, err := reply.JSON()
	if err != nil {
		protocolErrorCtr.Inc()
		return fmt.Errorf("could not JSON encode reply message to store it in the Secure Reply: %s", err)
	}

	hash := r.security.ChecksumBytes(j)
	r.MessageBody = string(j)
	r.Hash = base64.StdEncoding.EncodeToString(hash[:])

	return nil
}

// Message retrieves the stored message content
func (r *secureReply) Message() []byte {
	r.mu.Lock()
	defer r.mu.Unlock()

	return []byte(r.MessageBody)
}

// Valid validates the body of the message by comparing the recorded hash with the hash of the body
func (r *secureReply) Valid() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	hash := r.security.ChecksumBytes([]byte(r.MessageBody))
	if base64.StdEncoding.EncodeToString(hash[:]) == r.Hash {
		validCtr.Inc()
		return true
	}

	invalidCtr.Inc()
	return false
}

// JSON creates a JSON encoded reply
func (r *secureReply) JSON() ([]byte, error) {
	r.mu.Lock()
	j, err := json.Marshal(r)
	r.mu.Unlock()
	if err != nil {
		protocolErrorCtr.Inc()
		return nil, fmt.Errorf("could not JSON Marshal: %s", err)
	}

	if err = r.IsValidJSON(j); err != nil {
		return nil, fmt.Errorf("reply JSON produced from the SecureRequest does not pass validation: %s", err)
	}

	return j, nil
}

// Version retrieves the protocol version for this message
func (r *secureReply) Version() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.Protocol
}

// IsValidJSON validates the given JSON data against the schema
func (r *secureReply) IsValidJSON(data []byte) (err error) {
	if !protocol.ClientStrictValidation {
		return nil
	}

	_, errors, err := schemaValidate(secureReplySchema, data)
	if err != nil {
		return fmt.Errorf("could not validate SecureReply JSON data: %s", err)
	}

	if len(errors) != 0 {
		return fmt.Errorf("supplied JSON document is not a valid SecureReply message: %s", strings.Join(errors, ", "))
	}

	return nil
}
