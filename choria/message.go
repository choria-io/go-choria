package choria

import (
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/choria-io/go-choria/protocol"
)

// Message represents a Choria message
type Message struct {
	Payload string
	Agent   string

	Request *Message
	Filter  *protocol.Filter
	TTL     int

	SenderID        string
	CallerID        string
	RequestID       string
	DiscoveredHosts []string

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
		return msg, err
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
	msg.Filter, _ = req.Filter()
	msg.SenderID = choria.Config.Identity
	msg.SetBase64Payload(req.Message())
	msg.req = req

	return
}

// NewMessage constructs a basic Message instance
func NewMessage(payload string, agent string, collective string, msgType string, request *Message, choria *Framework) (msg *Message, err error) {
	msg = &Message{
		Payload:         payload,
		RequestID:       choria.NewRequestID(),
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
	if err != nil {
		return
	}

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
func (self *Message) Transport() (protocol.TransportMessage, error) {
	if self.msgType == "request" || self.msgType == "direct_request" {
		return self.requestTransport()
	} else if self.msgType == "reply" {
		return self.replyTransport()
	}

	return nil, fmt.Errorf("Do not know how to make a Transport for a %s type Message", self.msgType)
}

func (self *Message) requestTransport() (protocol.TransportMessage, error) {
	if self.protoVersion == "" {
		return nil, errors.New("Cannot create a Request Transport without a version, please set it using SetProtocolVersion()")
	}

	if self.replyTo == "" {
		return nil, errors.New("Cannot create a Transport, no reply-to was set, please use SetReplyTo()")
	}

	transport, err := self.choria.NewRequestTransportForMessage(self, self.protoVersion)
	if err != nil {
		return nil, fmt.Errorf("Could not create a Transport: %s", err.Error())
	}

	transport.SetReplyTo(self.ReplyTo())

	return transport, nil
}

func (self *Message) replyTransport() (protocol.TransportMessage, error) {
	if self.req == nil {
		return nil, fmt.Errorf("Cannot create a Transport, no request were stored in the message")
	}

	return self.choria.NewReplyTransportForMessage(self, self.req)
}

// SetProtocolVersion sets the version of the protocol that will be used by Transport()
func (self *Message) SetProtocolVersion(version string) {
	self.protoVersion = version
}

// Validate tests the Message and makes sure its settings are sane
func (self *Message) Validate() (bool, error) {
	if self.Agent == "" {
		return false, fmt.Errorf("Agent has not been set")
	}

	if self.collective == "" {
		return false, fmt.Errorf("Collective has not been set")
	}

	if !self.choria.HasCollective(self.collective) {
		return false, fmt.Errorf("%s is not on the list of known collectives", self.collective)
	}

	return true, nil
}

// SetBase64Payload sets the payload for the message, use it if the payload is Base64 encoded
func (self *Message) SetBase64Payload(payload string) error {
	str, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return fmt.Errorf("Could not decode supplied payload using base64: %s", err.Error())
	}

	self.Payload = string(str)

	return nil
}

// Base64Payload retrieves the payload Base64 encoded
func (self *Message) Base64Payload() string {
	return base64.StdEncoding.EncodeToString([]byte(self.Payload))
}

// SetExpectedMsgID sets the Request ID that is expected from the reply data
func (self *Message) SetExpectedMsgID(id string) error {
	if self.Type() != "reply" {
		return fmt.Errorf("Can only store expected message ID for reply messages")
	}

	self.expectedMessageID = id

	return nil
}

// ExpectedMessageID retrieves the expected message ID
func (self *Message) ExpectedMessageID() string {
	return self.expectedMessageID
}

// SetReplyTo sets the NATS target where replies to this message should go
func (self *Message) SetReplyTo(replyTo string) error {
	if !(self.Type() == "request" || self.Type() == "direct_request") {
		return fmt.Errorf("Custom reply to targets can only be set for requests")
	}

	self.replyTo = replyTo

	return nil
}

// ReplyTo retrieve the NATS reply target
func (self *Message) ReplyTo() string {
	return self.replyTo
}

// SetCollective sets the sub collective this message is targeting
func (self *Message) SetCollective(collective string) error {
	if !self.choria.HasCollective(collective) {
		return fmt.Errorf("%s is not on the list of known collectives", self.collective)
	}

	self.collective = collective

	return nil
}

// Collective retrieves the sub collective this message is targeting
func (self *Message) Collective() string {
	return self.collective
}

// SetType sets the mssage type. One message, request, direct_request or reply
func (self *Message) SetType(msgType string) (err error) {
	if !(msgType == "message" || msgType == "request" || msgType == "direct_request" || msgType == "reply") {
		return fmt.Errorf("%s is not a valid message type", msgType)
	}

	if msgType == "direct_request" {
		if len(self.DiscoveredHosts) == 0 {
			return fmt.Errorf("direct_request message type can only be set if DiscoveredHosts have been set")
		}

		self.Filter = &protocol.Filter{}
		self.Filter.AddAgentFilter(self.Agent)
	}

	self.msgType = msgType

	return
}

// Type retrieves the message type
func (self *Message) Type() string {
	return self.msgType
}

// String creates a string representation of the message for logs etc
func (self *Message) String() string {
	return fmt.Sprintf("%s from %s@%s for agent %s", self.RequestID, self.CallerID, self.SenderID, self.Agent)
}
