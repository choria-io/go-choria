package choria

import (
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/choria-io/go-choria/protocol"
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

	expectedMessageID    string
	replyTo              string
	collective           string
	msgType              string // message, request, direct_request, reply, service_request
	req                  protocol.Request
	protoVersion         string
	shouldCacheTransport bool
	cachedTransport      protocol.TransportMessage
	onPublish            func()

	sync.Mutex

	choria *Framework
}

const (
	MessageMessageType        string = "message"
	RequestMessageType        string = "request"
	DirectRequestMessageType  string = "direct_request"
	ServiceRequestMessageType string = "service_request"
	ReplyMessageType          string = "reply"
)

// NewMessageFromRequest constructs a Message based on a Request
func NewMessageFromRequest(req protocol.Request, replyto string, choria *Framework) (msg *Message, err error) {
	reqm, err := NewMessage(req.Message(), req.Agent(), req.Collective(), RequestMessageType, nil, choria)
	if err != nil {
		return msg, err
	}

	if replyto != "" {
		reqm.replyTo = replyto
	}

	msg, err = NewMessage(req.Message(), req.Agent(), req.Collective(), ReplyMessageType, reqm, choria)
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

	if choria.Configuration().CacheBatchedTransports {
		msg.shouldCacheTransport = true
	}

	return
}

// NewMessage constructs a basic Message instance
func NewMessage(payload string, agent string, collective string, msgType string, request *Message, choria *Framework) (msg *Message, err error) {
	id, err := choria.NewRequestID()
	if err != nil {
		return
	}

	cfg := choria.Configuration()

	msg = &Message{
		Payload:              payload,
		RequestID:            id,
		TTL:                  cfg.TTL,
		DiscoveredHosts:      []string{},
		SenderID:             cfg.Identity,
		CallerID:             choria.CallerID(),
		Filter:               protocol.NewFilter(),
		choria:               choria,
		shouldCacheTransport: cfg.CacheBatchedTransports,
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
		msg.SetType(ReplyMessageType)
		err = msg.SetCollective(request.collective)
		if err != nil {
			return
		}
	}

	_, err = msg.Validate()

	return
}

// OnPublish sets a callback that should be called just before this message is published,
// it might be called several times for batched or direct messages. This will be called after
// the message have been signed - potentially by a remote signer etc
func (m *Message) OnPublish(f func()) {
	m.Lock()
	defer m.Unlock()

	m.onPublish = f
}

// NotifyPublish triggers the callback set using OnPublish() in a blocking fashion
func (m *Message) NotifyPublish() {
	m.Lock()
	defer m.Unlock()

	if m.onPublish != nil {
		m.onPublish()
	}
}

// IsCachedTransport determines if transport messages will be cached
func (m *Message) IsCachedTransport() bool {
	m.Lock()
	defer m.Unlock()

	return m.shouldCacheTransport
}

// UniqueTransport ensures that every call to Transport() produce a unique transport message
func (m *Message) UniqueTransport() {
	m.Lock()
	defer m.Unlock()

	m.cachedTransport = nil
	m.shouldCacheTransport = false
}

// CacheTransport ensures that multiples calls to Transport() returns the same transport message
func (m *Message) CacheTransport() {
	m.Lock()
	defer m.Unlock()

	m.shouldCacheTransport = true
}

// Transport creates a TransportMessage for this Message
//
// In the case of a reply Message made using NewMessage the Transport will have
// the same version as the request that made it.  If you made the Message using
// some other way then look at choria.NewReplyTransportForMessage.
//
// For requests you need to set the protocol version using SetProtocolVersion()
// before calling Transport
func (m *Message) Transport() (protocol.TransportMessage, error) {
	m.Lock()
	defer m.Unlock()

	if m.shouldCacheTransport && m.cachedTransport != nil {
		return m.cachedTransport, nil
	}

	switch {
	case m.msgType == RequestMessageType || m.msgType == DirectRequestMessageType || m.msgType == ServiceRequestMessageType:
		t, err := m.requestTransport()
		if err != nil {
			return nil, err
		}

		if m.shouldCacheTransport {
			m.cachedTransport = t
		}

		return t, nil

	case m.msgType == ReplyMessageType:
		return m.replyTransport()

	default:
		return nil, fmt.Errorf("do not know how to make a Transport for a %s type Message", m.msgType)
	}
}

func (m *Message) isEmptyFilter() bool {
	if m.Filter == nil {
		return true
	}

	f := m.Filter

	if f.Fact == nil && f.Class == nil && f.Identity == nil && f.Compound == nil {
		return true
	}

	// we specifically handle the case where people do agent discovery against discovery agent and more than 1 agent filter
	if m.Agent == "discovery" && len(f.Agent) > 1 {
		return false
	}

	if (len(f.Agent) == 0 || m.Agent == "discovery" && len(f.Agent) == 1) && len(f.Fact) == 0 && len(f.Class) == 0 && len(f.Identity) == 0 && len(f.Compound) == 0 {
		return true
	}

	return false
}

func (m *Message) requestTransport() (protocol.TransportMessage, error) {
	if m.protoVersion == "" {
		return nil, errors.New("cannot create a Request Transport without a version, please set it using SetProtocolVersion()")
	}

	if m.replyTo == "" {
		return nil, errors.New("cannot create a Transport, no reply-to was set, please use SetReplyTo()")
	}

	if m.choria.Configuration().Choria.RequireClientFilter && m.isEmptyFilter() {
		return nil, fmt.Errorf("cannot create a Request Transport, requests without filters have been disabled")
	}

	transport, err := m.choria.NewRequestTransportForMessage(m, m.protoVersion)
	if err != nil {
		return nil, err
	}

	transport.SetReplyTo(m.ReplyTo())

	return transport, nil
}

func (m *Message) replyTransport() (protocol.TransportMessage, error) {
	if m.req == nil {
		return nil, fmt.Errorf("cannot create a Transport, no request were stored in the message")
	}

	return m.choria.NewReplyTransportForMessage(m, m.req)
}

// SetProtocolVersion sets the version of the protocol that will be used by Transport()
func (m *Message) SetProtocolVersion(version string) {
	m.protoVersion = version
}

// Validate tests the Message and makes sure its settings are sane
func (m *Message) Validate() (bool, error) {
	if m.Agent == "" {
		return false, fmt.Errorf("agent has not been set")
	}

	if m.collective == "" {
		return false, fmt.Errorf("collective has not been set")
	}

	if !m.choria.HasCollective(m.collective) {
		return false, fmt.Errorf("'%s' is not on the list of known collectives", m.collective)
	}

	return true, nil
}

// ValidateTTL validates the message age, true if the message should be allowed
func (m *Message) ValidateTTL() bool {
	now := time.Now()
	earliest := now.Add(-1 * time.Duration(m.TTL) * time.Second)
	latest := now.Add(time.Duration(m.TTL) * time.Second)

	return m.TimeStamp.Before(latest) && m.TimeStamp.After(earliest)
}

// SetBase64Payload sets the payload for the message, use it if the payload is Base64 encoded
func (m *Message) SetBase64Payload(payload string) error {
	str, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return fmt.Errorf("could not decode supplied payload using base64: %s", err)
	}

	m.Payload = string(str)

	return nil
}

// Base64Payload retrieves the payload Base64 encoded
func (m *Message) Base64Payload() string {
	return base64.StdEncoding.EncodeToString([]byte(m.Payload))
}

// SetExpectedMsgID sets the Request ID that is expected from the reply data
func (m *Message) SetExpectedMsgID(id string) error {
	if m.Type() != ReplyMessageType {
		return fmt.Errorf("can only store expected message ID for reply messages")
	}

	m.expectedMessageID = id

	return nil
}

// ExpectedMessageID retrieves the expected message ID
func (m *Message) ExpectedMessageID() string {
	return m.expectedMessageID
}

// SetReplyTo sets the NATS target where replies to this message should go
func (m *Message) SetReplyTo(replyTo string) error {
	if !(m.Type() == RequestMessageType || m.Type() == DirectRequestMessageType || m.Type() == ServiceRequestMessageType) {
		return fmt.Errorf("custom reply to targets can only be set for requests")
	}

	m.replyTo = replyTo

	return nil
}

// ReplyTo retrieve the NATS reply target
func (m *Message) ReplyTo() string {
	return m.replyTo
}

// SetCollective sets the sub collective this message is targeting
func (m *Message) SetCollective(collective string) error {
	if !m.choria.HasCollective(collective) {
		return fmt.Errorf("cannot set collective to '%s', it is not on the list of known collectives", collective)
	}

	m.collective = collective

	return nil
}

// Collective retrieves the sub collective this message is targeting
func (m *Message) Collective() string {
	return m.collective
}

// SetType sets the message type. One message, request, direct_request, service_request or reply
func (m *Message) SetType(msgType string) (err error) {
	if !(msgType == MessageMessageType || msgType == RequestMessageType || msgType == DirectRequestMessageType || msgType == ReplyMessageType || msgType == ServiceRequestMessageType) {
		return fmt.Errorf("%s is not a valid message type", msgType)
	}

	if msgType == DirectRequestMessageType {
		if len(m.DiscoveredHosts) == 0 {
			return fmt.Errorf("%s message type can only be set if DiscoveredHosts have been set", DirectRequestMessageType)
		}

		m.Filter.AddAgentFilter(m.Agent)
	}

	if msgType == ServiceRequestMessageType && len(m.DiscoveredHosts) != 0 {
		return fmt.Errorf("%s message type does not support DiscoveredHosts", ServiceRequestMessageType)
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
