// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package v2

type TransportMessage struct {
	// The protocol version for this transport `io.choria.protocol.v2.transport` / protocol.TransportV2
	Protocol string `json:"protocol"`
	// The payload to be transport, a Secure Request or Secure Reply
	Data []byte `json:"data"`
	// Optional headers
	Headers *TransportHeaders `json:"headers,omitempty"`
}

type TransportHeaders struct {
	// A transport specific response channel for this message, used in requests
	ReplyTo string `json:"reply,omitempty"`
	// The host that sent this message
	Sender string `json:"sender,omitempty"`
	// A trace of host/broker pairs that the message traversed
	SeenBy [][3]string `json:"trace,omitempty"`
	// Headers to assist federation
	Federation *FederationTransportHeader `json:"federation,omitempty"`
}

type FederationTransportHeader struct {
	// The request ID a federated message belongs to
	RequestID string `json:"request,omitempty"`
	// The original `reply` before federation
	ReplyTo string `json:"reply,omitempty"`
	// The identities who the federated message is for
	Targets []string `json:"targets,omitempty"`
}
