// Copyright (c) 2023, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package machines

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"

	iu "github.com/choria-io/go-choria/internal/util"
)

// Specification holds []ManagedPlugin marshaled to JSON with an optional ed25519 signature
type Specification struct {
	Plugins   []byte `json:"plugins"`
	Signature string `json:"signature,omitempty"`
}

// Encode sets the signature and Marshals the specification to JSON, if key is empty signature is not made
func (s *Specification) Encode(key string) ([]byte, error) {
	var pk ed25519.PrivateKey
	var err error

	data, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	if key != "" {
		if iu.IsEncodedEd25519KeyString(key) {
			pk, err = hex.DecodeString(key)
		} else {
			_, pk, err = iu.Ed25519KeyPairFromSeedFile(key)
		}
		if err != nil {
			return nil, err
		}

		sig, err := iu.Ed25519Sign(pk, data)
		if err != nil {
			return nil, err
		}

		s.Signature = hex.EncodeToString(sig)
	}

	return json.Marshal(s)
}
