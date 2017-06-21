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

func NewMessageFromRequest(req protocol.Request, choria *Choria) (msg *Message, err error) {
	msg, err = NewMessage("", req.Agent(), req.Collective(), "request", nil, choria)
	if err != nil {
		return
	}

	msg.RequestID = req.RequestID()
	msg.TTL = req.TTL()
	msg.Filter, _ = req.Filter()
	msg.SenderID = req.SenderID()
	msg.SetBase64Payload(req.Message())

	return
}

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
		msg.SetType("reply")
		err = msg.SetCollective(request.collective)
		if err != nil {
			return
		}
	}

	err, _ = msg.Validate()
	if err != nil {
		return
	}

	return
}

func (m Message) Validate() (error, bool) {
	if m.Agent == "" {
		return fmt.Errorf("Agent has not been set"), false
	}

	if m.collective == "" {
		return fmt.Errorf("Collective has not been set"), false
	}

	if !m.choria.HasCollective(m.collective) {
		return fmt.Errorf("%s is not on the list of known collectives", m.collective), false
	}

	return nil, true
}

func (m *Message) SetBase64Payload(payload string) error {
	str, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return fmt.Errorf("Could not decode supplied payload using base64: %s", err.Error())
	}

	m.Payload = string(str)

	return nil
}

func (m Message) Base64Payload() string {
	return base64.StdEncoding.EncodeToString([]byte(m.Payload))
}

func (m *Message) SetExpectedMsgID(id string) error {
	if m.Type() != "reply" {
		return fmt.Errorf("Can only store expected message ID for reply messages")
	}

	m.expectedMessageID = id

	return nil
}

func (m Message) ExpectedMessageID() string {
	return m.expectedMessageID
}

func (m *Message) SetReplyTo(replyTo string) error {
	if !(m.Type() == "request" || m.Type() == "direct_request") {
		return fmt.Errorf("Custom reply to targets can only be set for requests")
	}

	m.replyTo = replyTo

	return nil
}

func (m Message) ReplyTo() string {
	return m.replyTo
}

func (m *Message) SetCollective(collective string) error {
	if !m.choria.HasCollective(collective) {
		return fmt.Errorf("%s is not on the list of known collectives", m.collective)
	}

	m.collective = collective

	return nil
}

func (m Message) Collective() string {
	return m.collective
}

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

func (m Message) Type() string {
	return m.msgType
}
