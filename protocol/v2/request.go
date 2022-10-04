// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package v2

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/choria-io/go-choria/protocol"
)

type Request struct {
	Protocol    protocol.ProtocolVersion `json:"protocol"`
	MessageBody []byte                   `json:"message"`

	ReqEnvelope

	mu sync.Mutex
}

type ReqEnvelope struct {
	RequestID  string           `json:"id"`
	SenderID   string           `json:"sender"`
	CallerID   string           `json:"caller"`
	Collective string           `json:"collective"`
	Agent      string           `json:"agent"`
	TTL        int              `json:"ttl"`
	Time       int64            `json:"time"`
	Filter     *protocol.Filter `json:"filter,omitempty"`

	seenBy     [][3]string
	federation *FederationTransportHeader
}

// NewRequest creates a io.choria.protocol.v2.request
func NewRequest(agent string, sender string, caller string, ttl int, id string, collective string) (protocol.Request, error) {
	req := &Request{
		Protocol: protocol.RequestV2,
		ReqEnvelope: ReqEnvelope{
			SenderID:  sender,
			TTL:       ttl,
			RequestID: id,
			Time:      time.Now().UnixNano(),
		},
	}

	req.SetCollective(collective)
	req.SetAgent(agent)
	req.SetCallerID(caller)
	req.SetFilter(protocol.NewFilter())

	return req, nil
}

// NewRequestFromSecureRequest creates a io.choria.protocol.v2.request based on the data contained in a SecureRequest
func NewRequestFromSecureRequest(sr protocol.SecureRequest) (protocol.Request, error) {
	if sr.Version() != protocol.SecureRequestV2 {
		return nil, fmt.Errorf("cannot create a version 2 SecureRequest from a %s SecureRequest", sr.Version())
	}

	req := &Request{
		Protocol: protocol.RequestV2,
	}

	err := req.IsValidJSON(sr.Message())
	if err != nil {
		return nil, fmt.Errorf("the JSON body from the SecureRequest is not a valid Request message: %s", err)
	}

	err = json.Unmarshal(sr.Message(), req)
	if err != nil {
		return nil, fmt.Errorf("could not parse JSON data from Secure Request: %s", err)
	}

	return req, nil
}

// RecordNetworkHop appends a hop onto the list of those who processed this message
func (r *Request) RecordNetworkHop(in string, processor string, out string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.ReqEnvelope.seenBy = append(r.ReqEnvelope.seenBy, [3]string{in, processor, out})
}

// NetworkHops returns a list of tuples this messaged traveled through
func (r *Request) NetworkHops() [][3]string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.ReqEnvelope.seenBy
}

// FederationTargets retrieves the list of targets this message is destined for
func (r *Request) FederationTargets() (targets []string, federated bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.federation == nil {
		return nil, false
	}

	return r.ReqEnvelope.federation.Targets, true
}

// FederationReplyTo retrieves the reply to string set by the federation broker
func (r *Request) FederationReplyTo() (replyTo string, federated bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.ReqEnvelope.federation == nil {
		return "", false
	}

	return r.ReqEnvelope.federation.ReplyTo, true
}

// FederationRequestID retrieves the federation specific requestid
func (r *Request) FederationRequestID() (id string, federated bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.ReqEnvelope.federation == nil {
		return "", false
	}

	return r.ReqEnvelope.federation.RequestID, true
}

// SetRequestID sets the request ID for this message
func (r *Request) SetRequestID(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.ReqEnvelope.RequestID = id
}

// SetFederationTargets sets the list of hosts this message should go to.
//
// Federation brokers will duplicate the message and send one for each target
func (r *Request) SetFederationTargets(targets []string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.ReqEnvelope.federation == nil {
		r.ReqEnvelope.federation = &FederationTransportHeader{}
	}

	r.ReqEnvelope.federation.Targets = targets
}

// SetFederationReplyTo stores the original reply-to destination in the federation headers
func (r *Request) SetFederationReplyTo(reply string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.ReqEnvelope.federation == nil {
		r.ReqEnvelope.federation = &FederationTransportHeader{}
	}

	r.ReqEnvelope.federation.ReplyTo = reply
}

// SetFederationRequestID sets the request ID for federation purposes
func (r *Request) SetFederationRequestID(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.ReqEnvelope.federation == nil {
		r.ReqEnvelope.federation = &FederationTransportHeader{}
	}

	r.ReqEnvelope.federation.RequestID = id
}

// IsFederated determines if this message is federated
func (r *Request) IsFederated() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.ReqEnvelope.federation != nil
}

// SetUnfederated removes any federation information from the message
func (r *Request) SetUnfederated() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.ReqEnvelope.federation = nil
}

// SetMessage set the message body that's contained in this request
func (r *Request) SetMessage(message []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.MessageBody = message
}

// SetCallerID sets the caller id for this request
func (r *Request) SetCallerID(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// TODO validate it
	r.ReqEnvelope.CallerID = id
}

// SetCollective sets the collective this request is directed at
func (r *Request) SetCollective(collective string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.ReqEnvelope.Collective = collective
}

// SetAgent sets the agent this requires is created for
func (r *Request) SetAgent(agent string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.ReqEnvelope.Agent = agent
}

// SetTTL sets the validity period for this message
func (r *Request) SetTTL(ttl int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.ReqEnvelope.TTL = ttl
}

// Message retrieves the Message body
func (r *Request) Message() []byte {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.MessageBody
}

// RequestID retrieves the unique request ID
func (r *Request) RequestID() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.ReqEnvelope.RequestID
}

// SenderID retrieves the sender id that sent the message
func (r *Request) SenderID() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.ReqEnvelope.SenderID
}

// CallerID retrieves the caller id that sent the message
func (r *Request) CallerID() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.ReqEnvelope.CallerID
}

// Collective retrieves the name of the sub collective this message is aimed at
func (r *Request) Collective() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.ReqEnvelope.Collective
}

// Agent retrieves the agent name this message is for
func (r *Request) Agent() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.ReqEnvelope.Agent
}

// TTL retrieves the maximum allow lifetime of this message
func (r *Request) TTL() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.ReqEnvelope.TTL
}

// Time retrieves the time this message was first made
func (r *Request) Time() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()

	return time.Unix(0, r.ReqEnvelope.Time)
}

// Filter retrieves the filter for the message.  The boolean is true when the filter is not empty
func (r *Request) Filter() (filter *protocol.Filter, filtered bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.ReqEnvelope.Filter == nil {
		r.ReqEnvelope.Filter = protocol.NewFilter()
	}

	return r.ReqEnvelope.Filter, !r.ReqEnvelope.Filter.Empty()
}

// NewFilter creates a new empty filter and sets it
func (r *Request) NewFilter() *protocol.Filter {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.ReqEnvelope.Filter = protocol.NewFilter()

	return r.ReqEnvelope.Filter
}

// JSON creates a JSON encoded request
func (r *Request) JSON() ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	j, err := json.Marshal(r)
	if err != nil {
		protocolErrorCtr.Inc()
		return nil, fmt.Errorf("could not JSON Marshal: %s", err)
	}

	if err = r.isValidJSONUnlocked(j); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidJSON, err)
	}

	return j, nil
}

// SetFilter sets and overwrites the filter for a message with a new one
func (r *Request) SetFilter(filter *protocol.Filter) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.ReqEnvelope.Filter = filter
}

// Version retrieves the protocol version for this message
func (r *Request) Version() protocol.ProtocolVersion {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.Protocol
}

// IsValidJSON validates the given JSON data against the schema
func (r *Request) IsValidJSON(data []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.isValidJSONUnlocked(data)
}

func (r *Request) isValidJSONUnlocked(data []byte) error {
	_, errors, err := schemaValidate(protocol.RequestV2, data)
	if err != nil {
		return fmt.Errorf("could not validate Request JSON data: %s", err)
	}

	if len(errors) != 0 {
		return fmt.Errorf("%w: %s", ErrInvalidJSON, strings.Join(errors, ", "))
	}

	return nil
}
