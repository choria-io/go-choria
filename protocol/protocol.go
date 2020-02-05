package protocol

import (
	"time"
)

const (
	RequestV1       = "choria:request:1"
	ReplyV1         = "choria:reply:1"
	SecureRequestV1 = "choria:secure:request:1"
	SecureReplyV1   = "choria:secure:reply:1"
	TransportV1     = "choria:transport:1"
)

// Secure controls the signing and validations of certificates in the protocol
var Secure = "true"

// IsSecure determines if this build will validate senders at protocol level
func IsSecure() bool {
	return Secure == "true"
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

	SetMessage(message string)
	SetCallerID(id string)
	SetCollective(collective string)
	SetAgent(agent string)
	NewFilter() *Filter
	SetFilter(*Filter)
	SetRequestID(id string)
	SetTTL(ttl int)

	Message() string
	RequestID() string
	SenderID() string
	CallerID() string
	Collective() string
	Agent() string
	TTL() int
	Time() time.Time
	Filter() (*Filter, bool)
	JSON() (string, error)
	Version() string
	IsValidJSON(data string) error
}

// Reply is a core MCollective Reply containing JSON serialized agent payload
type Reply interface {
	Federable

	SetMessage(message string)

	Message() string
	RequestID() string
	SenderID() string
	Agent() string
	Time() time.Time
	JSON() (string, error)
	Version() string
	IsValidJSON(data string) error
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
	JSON() (string, error)
	Version() string
	IsValidJSON(data string) error
	Message() string
}

// SecureReply is a container for a Reply.  It's the reply counter part of a
// SecureRequest but replies are not signed using cryptographic keys it's only
// hashed in transport
type SecureReply interface {
	SetMessage(reply Reply) error
	Valid() bool
	JSON() (string, error)
	Message() string
	Version() string
	IsValidJSON(data string) error
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
	Message() (string, error)

	IsValidJSON(data string) error
	JSON() (string, error)
	Version() string
}
