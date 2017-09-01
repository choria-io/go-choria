package mcollective

import (
	"encoding/base64"
	"fmt"

	"github.com/choria-io/go-choria/protocol"
)

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
	choria            *Choria
}

// NewMessageFromRequest constructs a Message based on a Request
func NewMessageFromRequest(req protocol.Request, replyto string, choria *Choria) (msg *Message, err error) {
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

	return
}

// NewMessage constructs a basic Message instance
func NewMessage(payload string, agent string, collective string, msgType string, request *Message, choria *Choria) (msg *Message, err error) {
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

// Validate tests the Message and makes sure its settings are sane
func (m Message) Validate() (bool, error) {
	if m.Agent == "" {
		return false, fmt.Errorf("Agent has not been set")
	}

	if m.collective == "" {
		return false, fmt.Errorf("Collective has not been set")
	}

	if !m.choria.HasCollective(m.collective) {
		return false, fmt.Errorf("%s is not on the list of known collectives", m.collective)
	}

	return true, nil
}

// SetBase64Payload sets the payload for the message, use it if the payload is Base64 encoded
func (m *Message) SetBase64Payload(payload string) error {
	str, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return fmt.Errorf("Could not decode supplied payload using base64: %s", err.Error())
	}

	m.Payload = string(str)

	return nil
}

// Base64Payload retrieves the payload Base64 encoded
func (m Message) Base64Payload() string {
	return base64.StdEncoding.EncodeToString([]byte(m.Payload))
}

// SetExpectedMsgID sets the Request ID that is expected from the reply data
func (m *Message) SetExpectedMsgID(id string) error {
	if m.Type() != "reply" {
		return fmt.Errorf("Can only store expected message ID for reply messages")
	}

	m.expectedMessageID = id

	return nil
}

// ExpectedMessageID retrieves the expected message ID
func (m Message) ExpectedMessageID() string {
	return m.expectedMessageID
}

// SetReplyTo sets the NATS target where replies to this message should go
func (m *Message) SetReplyTo(replyTo string) error {
	if !(m.Type() == "request" || m.Type() == "direct_request") {
		return fmt.Errorf("Custom reply to targets can only be set for requests")
	}

	m.replyTo = replyTo

	return nil
}

// ReplyTo retrieve the NATS reply target
func (m Message) ReplyTo() string {
	return m.replyTo
}

// SetCollective sets the sub collective this message is targeting
func (m *Message) SetCollective(collective string) error {
	if !m.choria.HasCollective(collective) {
		return fmt.Errorf("%s is not on the list of known collectives", m.collective)
	}

	m.collective = collective

	return nil
}

// Collective retrieves the sub collective this message is targeting
func (m Message) Collective() string {
	return m.collective
}

// SetType sets the mssage type. One message, request, direct_request or reply
func (m *Message) SetType(msgType string) (err error) {
	if !(msgType == "message" || msgType == "request" || msgType == "direct_request" || msgType == "reply") {
		return fmt.Errorf("%s is not a valid message type", msgType)
	}

	if msgType == "direct_request" {
		if len(m.DiscoveredHosts) == 0 {
			return fmt.Errorf("direct_request message type can only be set if DiscoveredHosts have been set")
		}

		m.Filter = &protocol.Filter{}
		m.Filter.AddAgentFilter(m.Agent)
	}

	m.msgType = msgType

	return
}

// Type retrieves the message type
func (m *Message) Type() string {
	return m.msgType
}

// String creates a string representation of the message for logs etc
func (m *Message) String() string {
	return fmt.Sprintf("%s from %s@%s for agent %s", m.RequestID, m.CallerID, m.SenderID, m.Agent)
}
