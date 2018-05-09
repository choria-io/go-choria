package v1

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/choria-io/go-protocol/protocol"
)

type request struct {
	Protocol    string           `json:"protocol"`
	MessageBody string           `json:"message"`
	Envelope    *requestEnvelope `json:"envelope"`

	mu sync.Mutex
}

type requestEnvelope struct {
	RequestID  string           `json:"requestid"`
	SenderID   string           `json:"senderid"`
	CallerID   string           `json:"callerid"`
	Collective string           `json:"collective"`
	Agent      string           `json:"agent"`
	TTL        int              `json:"ttl"`
	Time       int64            `json:"time"`
	Filter     *protocol.Filter `json:"filter"`

	seenBy     [][3]string
	federation *federationTransportHeader
}

// RecordNetworkHop appends a hop onto the list of those who processed this message
func (r *request) RecordNetworkHop(in string, processor string, out string) {
	r.Envelope.seenBy = append(r.Envelope.seenBy, [3]string{in, processor, out})
}

// NetworkHops returns a list of tuples this messaged travelled through
func (r *request) NetworkHops() [][3]string {
	return r.Envelope.seenBy
}

// FederationTargets retrieves the list of targets this message is destined for
func (r *request) FederationTargets() (targets []string, federated bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Envelope.federation == nil {
		federated = false
		return
	}

	federated = true
	targets = r.Envelope.federation.Targets

	return
}

// FederationReply retrieves the reply to string set by the federation broker
func (r *request) FederationReplyTo() (replyto string, federated bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Envelope.federation == nil {
		federated = false
		return
	}

	federated = true
	replyto = r.Envelope.federation.ReplyTo

	return
}

// FederationRequestID retrieves the federation specific requestid
func (r *request) FederationRequestID() (id string, federated bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Envelope.federation == nil {
		federated = false
		return
	}

	federated = true
	id = r.Envelope.federation.RequestID

	return
}

// SetRequestID sets the request ID for this message
func (r *request) SetRequestID(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.Envelope.RequestID = id

	return
}

// SetFederationTargets sets the list of hosts this message should go to.
//
// Federation brokers will duplicate the message and send one for each target
func (r *request) SetFederationTargets(targets []string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Envelope.federation == nil {
		r.Envelope.federation = &federationTransportHeader{}
	}

	r.Envelope.federation.Targets = targets
}

// SetFederationReplyTo stores the original reply-to destination in the federation headers
func (r *request) SetFederationReplyTo(reply string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Envelope.federation == nil {
		r.Envelope.federation = &federationTransportHeader{}
	}

	r.Envelope.federation.ReplyTo = reply
}

// SetFederationRequestID sets the request ID for federation purposes
func (r *request) SetFederationRequestID(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Envelope.federation == nil {
		r.Envelope.federation = &federationTransportHeader{}
	}

	r.Envelope.federation.RequestID = id
}

// IsFederated determines if this message is federated
func (r *request) IsFederated() bool {
	return r.Envelope.federation != nil
}

// SetUnfederated removes any federation information from the message
func (r *request) SetUnfederated() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.Envelope.federation = nil
}

// SetMessage set the message body thats contained in this request.  It should be JSON encoded text
func (r *request) SetMessage(message string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.MessageBody = message

	return
}

// SetCallerID sets the caller id for this request
func (r *request) SetCallerID(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// TODO validate it
	r.Envelope.CallerID = id
}

// SetCollective sets the collective this request is directed at
func (r *request) SetCollective(collective string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.Envelope.Collective = collective
}

// SetAgent sets the agent this requires is created for
func (r *request) SetAgent(agent string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.Envelope.Agent = agent
}

// SetTTL sets the validity period for this message
func (r *request) SetTTL(ttl int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.Envelope.TTL = ttl
}

// Message retrieves the JSON encoded Message body
func (r *request) Message() string {
	return r.MessageBody
}

// RequestID retrieves the unique request ID
func (r *request) RequestID() string {
	return r.Envelope.RequestID
}

// SenderID retrieves the sender id that sent the message
func (r *request) SenderID() string {
	return r.Envelope.SenderID
}

// CallerID retrieves the caller id that sent the message
func (r *request) CallerID() string {
	return r.Envelope.CallerID
}

// Collective retrieves the name of the sub collective this message is aimed at
func (r *request) Collective() string {
	return r.Envelope.Collective
}

// Agent retrieves the agent name this message is for
func (r *request) Agent() string {
	return r.Envelope.Agent
}

// TTL retrieves the maximum allow lifetime of this message
func (r *request) TTL() int {
	return r.Envelope.TTL
}

// Time retrieves the time this message was first made
func (r *request) Time() time.Time {
	return time.Unix(r.Envelope.Time, 0)
}

// Filter retrieves the filter for the message.  The boolean is true when the filter is not empty
func (r *request) Filter() (filter *protocol.Filter, filtered bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Envelope.Filter.Empty() {
		filtered = false
	} else {
		filtered = true
	}

	filter = r.Envelope.Filter

	return
}

// NewFilter creates a new empty filter and sets it
func (r *request) NewFilter() *protocol.Filter {
	r.Envelope.Filter = protocol.NewFilter()

	return r.Envelope.Filter
}

// JSON creates a JSON encoded request
func (r *request) JSON() (body string, err error) {
	j, err := json.Marshal(r)
	if err != nil {
		protocolErrorCtr.Inc()
		err = fmt.Errorf("Could not JSON Marshal: %s", err.Error())
		return
	}

	body = string(j)

	if err = r.IsValidJSON(body); err != nil {
		err = fmt.Errorf("JSON produced from the Request does not pass validation: %s", err.Error())
		return
	}

	return
}

// SetFilter sets and overwrites the filter for a message with a new one
func (r *request) SetFilter(filter *protocol.Filter) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.Envelope.Filter = filter
}

// Version retreives the protocol version for this message
func (r *request) Version() string {
	return r.Protocol
}

// IsValidJSON validates the given JSON data against the schema
func (r *request) IsValidJSON(data string) (err error) {
	_, errors, err := schemas.Validate(schemas.RequestV1, data)
	if err != nil {
		err = fmt.Errorf("Could not validate Request JSON data: %s", err.Error())
		return
	}

	if len(errors) != 0 {
		err = fmt.Errorf("Supplied JSON document is not a valid Request message: %s", strings.Join(errors, ", "))
		return
	}

	return
}
