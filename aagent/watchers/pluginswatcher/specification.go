// Copyright (c) 2023, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package machines

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"

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

	if key != "" {
		if iu.FileExist(key) {
			_, pk, err = iu.Ed25519KeyPairFromSeedFile(key)
		} else {
			pk, err = hex.DecodeString(key)
		}
		if err != nil {
			return nil, err
		}

		sig, err := iu.Ed25519Sign(pk, s.Plugins)
		if err != nil {
			return nil, err
		}

		s.Signature = hex.EncodeToString(sig)
	}

	return json.Marshal(s)
}

// VerifySignature verifies that the data is signed using the supplied key
func (s *Specification) VerifySignature(key ed25519.PublicKey) (bool, error) {
	sig, err := hex.DecodeString(s.Signature)
	if err != nil {
		return false, fmt.Errorf("invalid signature data: %w", err)
	}

	return iu.Ed25519Verify(key, s.Plugins, sig)
}
