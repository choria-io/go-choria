package v1

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/choria-io/go-protocol/protocol"
)

type transportMessage struct {
	Protocol string            `json:"protocol"`
	Data     string            `json:"data"`
	Headers  *transportHeaders `json:"headers"`

	mu sync.Mutex
}

type transportHeaders struct {
	ReplyTo           string                     `json:"reply-to,omitempty"`
	MCollectiveSender string                     `json:"mc_sender,omitempty"`
	SeenBy            [][3]string                `json:"seen-by,omitempty"`
	Federation        *federationTransportHeader `json:"federation,omitempty"`
}

type federationTransportHeader struct {
	RequestID string   `json:"req,omitempty"`
	ReplyTo   string   `json:"reply-to,omitempty"`
	Targets   []string `json:"target,omitempty"`
}

// Message retrieves the stored data
func (m *transportMessage) Message() (data string, err error) {
	d, err := base64.StdEncoding.DecodeString(m.Data)
	if err != nil {
		err = fmt.Errorf("Could not base64 decode data received on the transport: %s", err.Error())
		return
	}

	data = string(d)

	return
}

// IsFederated determines if this message is federated
func (m *transportMessage) IsFederated() bool {
	return m.Headers.Federation != nil
}

// FederationTargets retrieves the list of targets this message is destined for
func (m *transportMessage) FederationTargets() (targets []string, federated bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Headers.Federation == nil {
		federated = false
		return
	}

	federated = true
	targets = m.Headers.Federation.Targets

	return
}

// FederationReply retrieves the reply to string set by the federation broker
func (m *transportMessage) FederationReplyTo() (replyto string, federated bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Headers.Federation == nil {
		federated = false
		return
	}

	federated = true
	replyto = m.Headers.Federation.ReplyTo

	return
}

// FederationRequestID retrieves the federation specific requestid
func (m *transportMessage) FederationRequestID() (id string, federated bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Headers.Federation == nil {
		federated = false
		return
	}

	federated = true
	id = m.Headers.Federation.RequestID

	return
}

// SenderID retrieves the identity of the sending host
func (m *transportMessage) SenderID() string {
	return m.Headers.MCollectiveSender
}

// ReplyTo retrieves the detination description where replies should go to
func (m *transportMessage) ReplyTo() string {
	return m.Headers.ReplyTo
}

// SeenBy retrieves the list of end points that this messages passed thruogh
func (m *transportMessage) SeenBy() [][3]string {
	return m.Headers.SeenBy
}

// SetFederationTargets sets the list of hosts this message should go to.
//
// Federation brokers will duplicate the message and send one for each target
func (m *transportMessage) SetFederationTargets(targets []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Headers.Federation == nil {
		m.Headers.Federation = &federationTransportHeader{}
	}

	m.Headers.Federation.Targets = targets
}

// SetFederationReplyTo stores the original reply-to destination in the federation headers
func (m *transportMessage) SetFederationReplyTo(reply string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Headers.Federation == nil {
		m.Headers.Federation = &federationTransportHeader{}
	}

	m.Headers.Federation.ReplyTo = reply
}

// SetFederationRequestID sets the request ID for federation purposes
func (m *transportMessage) SetFederationRequestID(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Headers.Federation == nil {
		m.Headers.Federation = &federationTransportHeader{}
	}

	m.Headers.Federation.RequestID = id
}

// SetSender sets the "mc_sender" - typically the identity of the sending host
func (m *transportMessage) SetSender(sender string) {
	m.Headers.MCollectiveSender = sender
}

// SetsReplyTo sets the reply-to targget
func (m *transportMessage) SetReplyTo(reply string) {
	m.Headers.ReplyTo = reply
}

// SetReplyData extracts the JSON body from a SecureReply and stores it
func (m *transportMessage) SetReplyData(reply protocol.SecureReply) (err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	j, err := reply.JSON()
	if err != nil {
		err = fmt.Errorf("Could not JSON encode the Reply structure for transport: %s", err.Error())
		return
	}

	m.Data = base64.StdEncoding.EncodeToString([]byte(j))

	return
}

// SetRequestData extracts the JSON body from a SecureRequest and stores it
func (m *transportMessage) SetRequestData(request protocol.SecureRequest) (err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	j, err := request.JSON()
	if err != nil {
		err = fmt.Errorf("Could not JSON encode the Request structure for transport: %s", err.Error())
		return
	}

	m.Data = base64.StdEncoding.EncodeToString([]byte(j))

	return
}

// RecordNetworkHop appends a hop onto the list of those who processed this message
func (m *transportMessage) RecordNetworkHop(in string, processor string, out string) {
	m.Headers.SeenBy = append(m.Headers.SeenBy, [3]string{in, processor, out})
}

// NetworkHops returns a list of tuples this messaged travelled through
func (m *transportMessage) NetworkHops() [][3]string {
	return m.Headers.SeenBy
}

// JSON creates a JSON encoded message
func (m *transportMessage) JSON() (body string, err error) {
	j, err := json.Marshal(m)
	if err != nil {
		err = fmt.Errorf("Could not JSON Marshal: %s", err.Error())
		return
	}

	body = string(j)

	if err = m.IsValidJSON(body); err != nil {
		err = fmt.Errorf("JSON produced from the Transport does not pass validation: %s", err.Error())
		return
	}

	return
}

// SetUnfederated removes any federation information from the message
func (m *transportMessage) SetUnfederated() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Headers.Federation = nil
}

// Version retreives the protocol version for this message
func (m *transportMessage) Version() string {
	return m.Protocol
}

// IsValidJSON validates the given JSON data against the Transport schema
func (m *transportMessage) IsValidJSON(data string) (err error) {
	if !protocol.ClientStrictValidation {
		return nil
	}

	_, errors, err := schemas.Validate(schemas.TransportV1, data)
	if err != nil {
		err = fmt.Errorf("Could not validate Transport JSON data: %s", err.Error())
		return
	}

	if len(errors) != 0 {
		err = fmt.Errorf("Supplied JSON document is not a valid Transport message: %s", strings.Join(errors, ", "))
		return
	}

	return
}
