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

// Additional to these the package for a specific version must also provide these constructors
// with signature matching those in v1/constructors.go

// Request is a core MCollective Request containing JSON serialized agent payload
type Request interface {
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
	IsValidJSON(data string) (error)
}

// Reply is a core MCollective Reply containing JSON serialized agent payload
type Reply interface {
	SetMessage(message string)

	Message() string
	RequestID() string
	SenderID() string
	Agent() string
	Time() time.Time
	JSON() (string, error)
	Version() string
	IsValidJSON(data string) (error)
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
	IsValidJSON(data string) (error)
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
	IsValidJSON(data string) (error)
}

// TransportMessage is a container for SecureRequests and SecureReplies it
// has routing information required to construct the various middleware topic
// names and such, it's also Federation aware and can track reply to targets,
// who saw it etc
type TransportMessage interface {
	SetReplyData(reply SecureReply) error
	SetRequestData(request SecureRequest) error

	SetReplyTo(reply string)
	SetSender(sender string)

	SetFederationRequestID(id string)
	SetFederationReplyTo(reply string)
	SetFederationTargets(targets []string)
	RecordNetworkHop(in string, processor string, out string)

	ReplyTo() string
	SenderID() string
	SeenBy() [][3]string
	Message() (string, error)

	FederationRequestID() (string, bool)
	FederationReplyTo() (string, bool)
	FederationTargets() ([]string, bool)

	IsValidJSON(data string) error
	JSON() (string, error)
	Version() string
}
