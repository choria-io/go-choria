// Copyright (c) 2017-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package protocol

import (
	"encoding/json"
	"time"
)

// ProtocolVersion defines known protocol versions
type ProtocolVersion string

const (
	RequestV1          ProtocolVersion = "choria:request:1"
	ReplyV1            ProtocolVersion = "choria:reply:1"
	SecureRequestV1    ProtocolVersion = "choria:secure:request:1"
	SecureReplyV1      ProtocolVersion = "choria:secure:reply:1"
	TransportV1        ProtocolVersion = "choria:transport:1"
	RequestV2          ProtocolVersion = "io.choria.protocol.v2.request"
	ReplyV2            ProtocolVersion = "io.choria.protocol.v2.reply"
	SecureRequestV2    ProtocolVersion = "io.choria.protocol.v2.secure_request"
	SecureReplyV2      ProtocolVersion = "io.choria.protocol.v2.secure_reply"
	TransportV2        ProtocolVersion = "io.choria.protocol.v2.transport"
	Unknown            ProtocolVersion = "io.choria.protocol.unknown"
	RemoteSigningAgent                 = "aaa_signer"
	RegistrationAgent                  = "registration"
)

func (p ProtocolVersion) String() string { return string(p) }

func (p *ProtocolVersion) UnmarshalJSON(data []byte) error {
	var rp string

	err := json.Unmarshal(data, &rp)

	switch ProtocolVersion(rp) {
	case RequestV1:
		*p = RequestV1
	case ReplyV1:
		*p = ReplyV1
	case SecureRequestV1:
		*p = SecureRequestV1
	case SecureReplyV1:
		*p = SecureReplyV1
	case TransportV1:
		*p = TransportV1
	case RequestV2:
		*p = RequestV2
	case ReplyV2:
		*p = ReplyV2
	case SecureRequestV2:
		*p = SecureRequestV2
	case SecureReplyV2:
		*p = SecureReplyV2
	case TransportV2:
		*p = TransportV2
	default:
		*p = Unknown
	}

	return err
}

type protoDetect struct {
	Protocol ProtocolVersion `json:"protocol"`
}

// Secure controls the signing and validations of certificates in the protocol
var Secure = "true"

func VersionFromJSON(data []byte) ProtocolVersion {
	d := &protoDetect{}
	err := json.Unmarshal(data, d)
	if err != nil {
		return Unknown
	}

	return d.Protocol
}

// IsSecure determines if this build will validate senders at protocol level
func IsSecure() bool {
	return Secure == "true"
}

// IsRemoteSignerAgent determines if agent is the standard remote signer
func IsRemoteSignerAgent(agent string) bool {
	return agent == RemoteSigningAgent
}

// IsRegistrationAgent determines if agent is the registration target agent
func IsRegistrationAgent(agent string) bool {
	return agent == RegistrationAgent
}

// ClientStrictValidation gives hints to the protocol implementations that
// a client does not wish to be fully validated, this is because validation
// can often be very slow so clients can elect to disable that.
//
// It's not mandatory for a specific version of implementation of the protocol
// to do anything with this, so it's merely a hint
var ClientStrictValidation = false

// Additional to these the package for a specific version must also provide these constructors
// with signature matching those in v1/constructors.go these are in use by mcollective/protocol.gos

// Federable is any kind of message that can carry federation headers
type Federable interface {
	SetFederationRequestID(id string)
	SetFederationReplyTo(reply string)
	SetFederationTargets(targets []string)
	SetUnfederated()

	FederationRequestID() (string, bool)
	FederationReplyTo() (string, bool)
	FederationTargets() ([]string, bool)

	RecordNetworkHop(in string, processor string, out string)
	NetworkHops() [][3]string

	IsFederated() bool
}

// Request is a core MCollective Request containing JSON serialized agent payload
type Request interface {
	Federable

	SetMessage(message []byte)
	SetCallerID(id string)
	SetCollective(collective string)
	SetAgent(agent string)
	NewFilter() *Filter
	SetFilter(*Filter)
	SetRequestID(id string)
	SetTTL(ttl int)

	Message() []byte
	RequestID() string
	SenderID() string
	CallerID() string
	Collective() string
	Agent() string
	TTL() int
	Time() time.Time
	Filter() (*Filter, bool)
	JSON() ([]byte, error)
	Version() ProtocolVersion
	IsValidJSON(data []byte) error
}

// Reply is a core MCollective Reply containing JSON serialized agent payload
type Reply interface {
	Federable

	SetMessage(message []byte)

	Message() []byte
	RequestID() string
	SenderID() string
	Agent() string
	Time() time.Time
	JSON() ([]byte, error)
	Version() ProtocolVersion
	IsValidJSON(data []byte) error
}

// SecureRequest is a container for the Request.  It serializes and signs the
// payload using the private key so that the message cannot be tampered with
// in any way once created.  Recipients of the message can unpack it and validate
// it using the certificate of the stated caller
//
// Should a message have been tampered with this validation would fail, this
// effectively avoids man in the middle attacks and requestor spoofing
type SecureRequest interface {
	SetMessage(request Request) error
	Valid() bool
	JSON() ([]byte, error)
	Version() ProtocolVersion
	IsValidJSON(data []byte) error
	Message() []byte
}

// SecureReply is a container for a Reply.  It's the reply counterpart of a
// SecureRequest but replies are not signed using cryptographic keys it's only
// hashed in transport
type SecureReply interface {
	SetMessage(reply Reply) error
	Valid() bool
	JSON() ([]byte, error)
	Message() []byte
	Version() ProtocolVersion
	IsValidJSON(data []byte) error
}

// TransportMessage is a container for SecureRequests and SecureReplies it
// has routing information required to construct the various middleware topic
// names and such, it's also Federation aware and can track reply to targets,
// who saw it etc
type TransportMessage interface {
	Federable

	SetReplyData(reply SecureReply) error
	SetRequestData(request SecureRequest) error

	SetReplyTo(reply string)
	SetSender(sender string)

	ReplyTo() string
	SenderID() string
	SeenBy() [][3]string
	Message() ([]byte, error)

	IsValidJSON(data []byte) error
	JSON() ([]byte, error)
	Version() ProtocolVersion
}
