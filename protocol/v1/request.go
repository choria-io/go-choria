// Copyright (c) 2017-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1

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
	MessageBody string                   `json:"message"`
	Envelope    *RequestEnvelope         `json:"envelope"`

	mu sync.Mutex
}

type RequestEnvelope struct {
	RequestID  string           `json:"requestid"`
	SenderID   string           `json:"senderid"`
	CallerID   string           `json:"callerid"`
	Collective string           `json:"collective"`
	Agent      string           `json:"agent"`
	TTL        int              `json:"ttl"`
	Time       int64            `json:"time"`
	Filter     *protocol.Filter `json:"filter"`

	seenBy     [][3]string
	federation *FederationTransportHeader
}

// RecordNetworkHop appends a hop onto the list of those who processed this message
func (r *Request) RecordNetworkHop(in string, processor string, out string) {
	r.Envelope.seenBy = append(r.Envelope.seenBy, [3]string{in, processor, out})
}

// NetworkHops returns a list of tuples this messaged traveled through
func (r *Request) NetworkHops() [][3]string {
	return r.Envelope.seenBy
}

// FederationTargets retrieves the list of targets this message is destined for
func (r *Request) FederationTargets() (targets []string, federated bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Envelope.federation == nil {
		return nil, false
	}

	return r.Envelope.federation.Targets, true
}

// FederationReplyTo retrieves the reply to string set by the federation broker
func (r *Request) FederationReplyTo() (replyto string, federated bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Envelope.federation == nil {
		return "", false
	}

	return r.Envelope.federation.ReplyTo, true
}

// FederationRequestID retrieves the federation specific requestid
func (r *Request) FederationRequestID() (id string, federated bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Envelope.federation == nil {
		return "", false
	}

	return r.Envelope.federation.RequestID, true
}

// SetRequestID sets the request ID for this message
func (r *Request) SetRequestID(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.Envelope.RequestID = id
}

// SetFederationTargets sets the list of hosts this message should go to.
//
// Federation brokers will duplicate the message and send one for each target
func (r *Request) SetFederationTargets(targets []string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Envelope.federation == nil {
		r.Envelope.federation = &FederationTransportHeader{}
	}

	r.Envelope.federation.Targets = targets
}

// SetFederationReplyTo stores the original reply-to destination in the federation headers
func (r *Request) SetFederationReplyTo(reply string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Envelope.federation == nil {
		r.Envelope.federation = &FederationTransportHeader{}
	}

	r.Envelope.federation.ReplyTo = reply
}

// SetFederationRequestID sets the request ID for federation purposes
func (r *Request) SetFederationRequestID(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Envelope.federation == nil {
		r.Envelope.federation = &FederationTransportHeader{}
	}

	r.Envelope.federation.RequestID = id
}

// IsFederated determines if this message is federated
func (r *Request) IsFederated() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.Envelope.federation != nil
}

// SetUnfederated removes any federation information from the message
func (r *Request) SetUnfederated() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.Envelope.federation = nil
}

// SetMessage set the message body thats contained in this request.  It should be JSON encoded text
func (r *Request) SetMessage(message []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.MessageBody = string(message)
}

// SetCallerID sets the caller id for this request
func (r *Request) SetCallerID(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// TODO validate it
	r.Envelope.CallerID = id
}

// SetCollective sets the collective this request is directed at
func (r *Request) SetCollective(collective string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.Envelope.Collective = collective
}

// SetAgent sets the agent this requires is created for
func (r *Request) SetAgent(agent string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.Envelope.Agent = agent
}

// SetTTL sets the validity period for this message
func (r *Request) SetTTL(ttl int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.Envelope.TTL = ttl
}

// Message retrieves the JSON encoded Message body
func (r *Request) Message() []byte {
	r.mu.Lock()
	defer r.mu.Unlock()

	return []byte(r.MessageBody)
}

// RequestID retrieves the unique request ID
func (r *Request) RequestID() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.Envelope.RequestID
}

// SenderID retrieves the sender id that sent the message
func (r *Request) SenderID() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.Envelope.SenderID
}

// CallerID retrieves the caller id that sent the message
func (r *Request) CallerID() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.Envelope.CallerID
}

// Collective retrieves the name of the sub collective this message is aimed at
func (r *Request) Collective() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.Envelope.Collective
}

// Agent retrieves the agent name this message is for
func (r *Request) Agent() string {
	return r.Envelope.Agent
}

// TTL retrieves the maximum allow lifetime of this message
func (r *Request) TTL() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.Envelope.TTL
}

// Time retrieves the time this message was first made
func (r *Request) Time() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()

	return time.Unix(r.Envelope.Time, 0)
}

// Filter retrieves the filter for the message.  The boolean is true when the filter is not empty
func (r *Request) Filter() (filter *protocol.Filter, filtered bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.Envelope.Filter, !r.Envelope.Filter.Empty()
}

// NewFilter creates a new empty filter and sets it
func (r *Request) NewFilter() *protocol.Filter {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.Envelope.Filter = protocol.NewFilter()

	return r.Envelope.Filter
}

// JSON creates a JSON encoded request
func (r *Request) JSON() ([]byte, error) {
	r.mu.Lock()
	j, err := json.Marshal(r)
	r.mu.Unlock()
	if err != nil {
		protocolErrorCtr.Inc()
		return nil, fmt.Errorf("could not JSON Marshal: %s", err)
	}

	if err = r.IsValidJSON(j); err != nil {
		return nil, fmt.Errorf("serialized JSON produced from the Request does not pass validation: %s", err)
	}

	return j, nil
}

// SetFilter sets and overwrites the filter for a message with a new one
func (r *Request) SetFilter(filter *protocol.Filter) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.Envelope.Filter = filter
}

// Version retrieves the protocol version for this message
func (r *Request) Version() protocol.ProtocolVersion {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.Protocol
}

// IsValidJSON validates the given JSON data against the schema
func (r *Request) IsValidJSON(data []byte) error {
	_, errors, err := schemaValidate(protocol.RequestV1, data)
	if err != nil {
		return fmt.Errorf("could not validate Request JSON data: %s", err)
	}

	if len(errors) != 0 {
		return fmt.Errorf("supplied JSON document is not a valid Request message: %s", strings.Join(errors, ", "))
	}

	return nil
}

// CallerPublicData is not supported for version 1 messages and is always an empty string
func (r *Request) CallerPublicData() string {
	return ""
}

// SignerPublicData is not supported for version 1 messages and is always an empty string
func (r *Request) SignerPublicData() string {
	return ""
}
