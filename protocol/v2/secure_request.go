// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package v2

// SecureRequest contains 1 serialized Request signed and with the related JWTs attached
type SecureRequest struct {
	// The protocol version for this secure request `io.choria.protocol.v2.secure_request` / protocol.SecureRequestV2
	Protocol string `json:"protocol"`
	// The request held in the Secure Request
	MessageBody []byte `json:"request"`
	// A signature made using the ed25519 seed of the sender
	Signature string `json:"signature"`
	// The JWT of the caller
	CallerJWT string `json:"caller"`
	// The JWT of the delegated signer, present when the AAA server is used
	SignerJWT string `json:"signer,omitempty"`
}
