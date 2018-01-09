package v1

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

type reply struct {
	Protocol    string         `json:"protocol"`
	MessageBody string         `json:"message"`
	Envelope    *replyEnvelope `json:"envelope"`

	mu sync.Mutex
}

type replyEnvelope struct {
	RequestID string `json:"requestid"`
	SenderID  string `json:"senderid"`
	Agent     string `json:"agent"`
	Time      int64  `json:"time"`

	seenBy     [][3]string
	federation *federationTransportHeader
}

// RecordNetworkHop appends a hop onto the list of those who processed this message
func (m *reply) RecordNetworkHop(in string, processor string, out string) {
	m.Envelope.seenBy = append(m.Envelope.seenBy, [3]string{in, processor, out})
}

// NetworkHops returns a list of tuples this messaged travelled through
func (m *reply) NetworkHops() [][3]string {
	return m.Envelope.seenBy
}

// SetMessage sets the data to be stored in the Reply.  It should be JSON encoded already.
func (r *reply) SetMessage(message string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.MessageBody = message

	return
}

// Message retrieves the JSON encoded message set using SetMessage
func (r *reply) Message() (msg string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.MessageBody
}

// RequestID retrieves the unique request id
func (r *reply) RequestID() string {
	return r.Envelope.RequestID
}

// SenderID retrieves the identity of the sending node
func (r *reply) SenderID() string {
	return r.Envelope.SenderID
}

// Agent retrieves the agent name that sent this reply
func (r *reply) Agent() string {
	return r.Envelope.Agent
}

// Time retrieves the time stamp that this message was made
func (r *reply) Time() time.Time {
	return time.Unix(r.Envelope.Time, 0)
}

// JSON creates a JSON encoded reply
func (r *reply) JSON() (body string, err error) {
	j, err := json.Marshal(r)
	if err != nil {
		err = fmt.Errorf("Could not JSON Marshal: %s", err.Error())
		protocolErrorCtr.Inc()
		return
	}

	body = string(j)

	if err = r.IsValidJSON(body); err != nil {
		err = fmt.Errorf("JSON produced from the Reply does not pass validation: %s", err.Error())
		return
	}

	return
}

// Version retrieves the protocol version for this message
func (r *reply) Version() string {
	return r.Protocol
}

// IsValidJSON validates the given JSON data against the schema
func (r *reply) IsValidJSON(data string) (err error) {
	_, errors, err := schemas.Validate(schemas.ReplyV1, data)
	if err != nil {
		err = fmt.Errorf("Could not validate Reply JSON data: %s", err.Error())
		return
	}

	if len(errors) != 0 {
		err = fmt.Errorf("Supplied JSON document is not a valid Reply message: %s", strings.Join(errors, ", "))
		return
	}

	return
}

// FederationTargets retrieves the list of targets this message is destined for
func (m *reply) FederationTargets() (targets []string, federated bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Envelope.federation == nil {
		federated = false
		return
	}

	federated = true
	targets = m.Envelope.federation.Targets

	return
}

// FederationReply retrieves the reply to string set by the federation broker
func (m *reply) FederationReplyTo() (replyto string, federated bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Envelope.federation == nil {
		federated = false
		return
	}

	federated = true
	replyto = m.Envelope.federation.ReplyTo

	return
}

// FederationRequestID retrieves the federation specific requestid
func (m *reply) FederationRequestID() (id string, federated bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Envelope.federation == nil {
		federated = false
		return
	}

	federated = true
	id = m.Envelope.federation.RequestID

	return
}

// SetFederationTargets sets the list of hosts this message should go to.
//
// Federation brokers will duplicate the message and send one for each target
func (m *reply) SetFederationTargets(targets []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Envelope.federation == nil {
		m.Envelope.federation = &federationTransportHeader{}
	}

	m.Envelope.federation.Targets = targets
}

// SetFederationReplyTo stores the original reply-to destination in the federation headers
func (m *reply) SetFederationReplyTo(reply string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Envelope.federation == nil {
		m.Envelope.federation = &federationTransportHeader{}
	}

	m.Envelope.federation.ReplyTo = reply
}

// SetFederationRequestID sets the request ID for federation purposes
func (m *reply) SetFederationRequestID(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Envelope.federation == nil {
		m.Envelope.federation = &federationTransportHeader{}
	}

	m.Envelope.federation.RequestID = id
}

// IsFederated determines if this message is federated
func (m *reply) IsFederated() bool {
	return m.Envelope.federation != nil
}

// SetUnfederated removes any federation information from the message
func (m *reply) SetUnfederated() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Envelope.federation = nil
}
