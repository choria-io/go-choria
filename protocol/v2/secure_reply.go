// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package v2

// SecureReply contains 1 serialized Reply hashed
type SecureReply struct {
	// The protocol version for this secure reply `io.choria.protocol.v2.secure_reply` /  / protocol.SecureReplyV2
	Protocol string `json:"protocol"`
	// The reply held in the Secure Request
	MessageBody []byte `json:"reply"`
	// A sha256 of the reply
	Hash string `json:"hash"`
	// A signature made using the ed25519 seed of the sender
	Signature string `json:"signature"`
	// The JWT of the sending host
	SenderJWT string `json:"sender"`
}
