package v1

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/choria-io/go-choria/protocol"
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
}

// SetRequestID sets the request ID for this message
func (r *request) SetRequestID(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.Envelope.RequestID = id

	return
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
