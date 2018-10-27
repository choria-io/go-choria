package choria

import (
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/choria-io/go-protocol/protocol"
)

// Message represents a Choria message
type Message struct {
	Payload string
	Agent   string

	Request   *Message
	Filter    *protocol.Filter
	TTL       int
	TimeStamp time.Time

	SenderID        string
	CallerID        string
	RequestID       string
	DiscoveredHosts []string

	CustomTarget string

	expectedMessageID string
	replyTo           string
	collective        string
	msgType           string // message, request, direct_request, reply
	req               protocol.Request
	protoVersion      string

	choria *Framework
}

// NewMessageFromRequest constructs a Message based on a Request
func NewMessageFromRequest(req protocol.Request, replyto string, choria *Framework) (msg *Message, err error) {
	reqm, err := NewMessage(req.Message(), req.Agent(), req.Collective(), "request", nil, choria)
	if err != nil {
		return msg, fmt.Errorf("could not create request message: %s", err)
	}

	if replyto != "" {
		reqm.replyTo = replyto
	}

	msg, err = NewMessage(req.Message(), req.Agent(), req.Collective(), "reply", reqm, choria)
	if err != nil {
		return msg, err
	}

	msg.RequestID = req.RequestID()
	msg.TTL = req.TTL()
	msg.TimeStamp = req.Time()
	msg.Filter, _ = req.Filter()
	msg.SenderID = choria.Config.Identity
	msg.SetBase64Payload(req.Message())
	msg.req = req

	return
}

// NewMessage constructs a basic Message instance
func NewMessage(payload string, agent string, collective string, msgType string, request *Message, choria *Framework) (msg *Message, err error) {
	id, err := choria.NewRequestID()
	if err != nil {
		return
	}

	msg = &Message{
		Payload:         payload,
		RequestID:       id,
		TTL:             choria.Config.TTL,
		DiscoveredHosts: []string{},
		SenderID:        choria.Config.Identity,
		CallerID:        choria.CallerID(),
		Filter:          protocol.NewFilter(),
		choria:          choria,
	}

	err = msg.SetType(msgType)
	if err != nil {
		return
	}

	if request == nil {
		msg.Agent = agent
		err = msg.SetCollective(collective)
		if err != nil {
			return
		}
	} else {
		msg.Request = request
		msg.Agent = request.Agent
		msg.replyTo = request.ReplyTo()
		msg.SetType("reply")
		err = msg.SetCollective(request.collective)
		if err != nil {
			return
		}
	}

	_, err = msg.Validate()

	return
}

// Transport creates a TransportMessage for this Message
//
// In the case of a reply Message made using NewMessage the Transport will have
// the same version as the request that made it.  If you made the Message using
// some other way then look at choria.NewReplyTransportForMessage.
//
// For requests you need to set the protocol version using SetProtocolVersion()
// before calling Transport
func (msg *Message) Transport() (protocol.TransportMessage, error) {
	if msg.msgType == "request" || msg.msgType == "direct_request" {
		return msg.requestTransport()
	} else if msg.msgType == "reply" {
		return msg.replyTransport()
	}

	return nil, fmt.Errorf("do not know how to make a Transport for a %s type Message", msg.msgType)
}

func (msg *Message) requestTransport() (protocol.TransportMessage, error) {
	if msg.protoVersion == "" {
		return nil, errors.New("cannot create a Request Transport without a version, please set it using SetProtocolVersion()")
	}

	if msg.replyTo == "" {
		return nil, errors.New("cannot create a Transport, no reply-to was set, please use SetReplyTo()")
	}

	transport, err := msg.choria.NewRequestTransportForMessage(msg, msg.protoVersion)
	if err != nil {
		return nil, fmt.Errorf("could not create a Transport: %s", err)
	}

	transport.SetReplyTo(msg.ReplyTo())

	return transport, nil
}

func (msg *Message) replyTransport() (protocol.TransportMessage, error) {
	if msg.req == nil {
		return nil, fmt.Errorf("cannot create a Transport, no request were stored in the message")
	}

	return msg.choria.NewReplyTransportForMessage(msg, msg.req)
}

// SetProtocolVersion sets the version of the protocol that will be used by Transport()
func (msg *Message) SetProtocolVersion(version string) {
	msg.protoVersion = version
}

// Validate tests the Message and makes sure its settings are sane
func (msg *Message) Validate() (bool, error) {
	if msg.Agent == "" {
		return false, fmt.Errorf("agent has not been set")
	}

	if msg.collective == "" {
		return false, fmt.Errorf("collective has not been set")
	}

	if !msg.choria.HasCollective(msg.collective) {
		return false, fmt.Errorf("'%s' is not on the list of known collectives", msg.collective)
	}

	return true, nil
}

// ValidateTTL validates the message age, true if the message should be allowed
func (msg *Message) ValidateTTL() bool {
	now := time.Now()
	earliest := now.Add(-1 * time.Duration(msg.TTL) * time.Second)
	latest := now.Add(time.Duration(msg.TTL) * time.Second)

	return msg.TimeStamp.Before(latest) && msg.TimeStamp.After(earliest)
}

// SetBase64Payload sets the payload for the message, use it if the payload is Base64 encoded
func (msg *Message) SetBase64Payload(payload string) error {
	str, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return fmt.Errorf("could not decode supplied payload using base64: %s", err)
	}

	msg.Payload = string(str)

	return nil
}

// Base64Payload retrieves the payload Base64 encoded
func (msg *Message) Base64Payload() string {
	return base64.StdEncoding.EncodeToString([]byte(msg.Payload))
}

// SetExpectedMsgID sets the Request ID that is expected from the reply data
func (msg *Message) SetExpectedMsgID(id string) error {
	if msg.Type() != "reply" {
		return fmt.Errorf("can only store expected message ID for reply messages")
	}

	msg.expectedMessageID = id

	return nil
}

// ExpectedMessageID retrieves the expected message ID
func (msg *Message) ExpectedMessageID() string {
	return msg.expectedMessageID
}

// SetReplyTo sets the NATS target where replies to this message should go
func (msg *Message) SetReplyTo(replyTo string) error {
	if !(msg.Type() == "request" || msg.Type() == "direct_request") {
		return fmt.Errorf("custom reply to targets can only be set for requests")
	}

	msg.replyTo = replyTo

	return nil
}

// ReplyTo retrieve the NATS reply target
func (msg *Message) ReplyTo() string {
	return msg.replyTo
}

// SetCollective sets the sub collective this message is targeting
func (msg *Message) SetCollective(collective string) error {
	if !msg.choria.HasCollective(collective) {
		return fmt.Errorf("cannot set collective to '%s', it is not on the list of known collectives", collective)
	}

	msg.collective = collective

	return nil
}

// Collective retrieves the sub collective this message is targeting
func (msg *Message) Collective() string {
	return msg.collective
}

// SetType sets the mssage type. One message, request, direct_request or reply
func (msg *Message) SetType(msgType string) (err error) {
	if !(msgType == "message" || msgType == "request" || msgType == "direct_request" || msgType == "reply") {
		return fmt.Errorf("%s is not a valid message type", msgType)
	}

	if msgType == "direct_request" {
		if len(msg.DiscoveredHosts) == 0 {
			return fmt.Errorf("direct_request message type can only be set if DiscoveredHosts have been set")
		}

		msg.Filter = protocol.NewFilter()
		msg.Filter.AddAgentFilter(msg.Agent)
	}

	msg.msgType = msgType

	return
}

// Type retrieves the message type
func (msg *Message) Type() string {
	return msg.msgType
}

// String creates a string representation of the message for logs etc
func (msg *Message) String() string {
	return fmt.Sprintf("%s from %s@%s for agent %s", msg.RequestID, msg.CallerID, msg.SenderID, msg.Agent)
}
