// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package v2

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/choria-io/go-choria/protocol"
)

// NewRequest creates a io.choria.protocol.v2.request
func NewRequest(agent string, sender string, caller string, ttl int, id string, collective string) (protocol.Request, error) {
	req := &request{
		Protocol: protocol.RequestV2,
		reqEnvelope: reqEnvelope{
			SenderID:  sender,
			TTL:       ttl,
			RequestID: id,
			Time:      time.Now().Unix(),
		},
	}

	req.SetCollective(collective)
	req.SetAgent(agent)
	req.SetCallerID(caller)
	req.SetFilter(protocol.NewFilter())

	return req, nil
}

type request struct {
	Protocol    string `json:"protocol"`
	MessageBody []byte `json:"payload"`

	reqEnvelope

	mu sync.Mutex
}

type reqEnvelope struct {
	RequestID  string           `json:"id"`
	SenderID   string           `json:"sender"`
	CallerID   string           `json:"caller"`
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
	r.mu.Lock()
	defer r.mu.Unlock()

	r.reqEnvelope.seenBy = append(r.reqEnvelope.seenBy, [3]string{in, processor, out})
}

// NetworkHops returns a list of tuples this messaged traveled through
func (r *request) NetworkHops() [][3]string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.reqEnvelope.seenBy
}

// FederationTargets retrieves the list of targets this message is destined for
func (r *request) FederationTargets() (targets []string, federated bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.federation == nil {
		return nil, false
	}

	return r.reqEnvelope.federation.Targets, true
}

// FederationReplyTo retrieves the reply to string set by the federation broker
func (r *request) FederationReplyTo() (replyTo string, federated bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.reqEnvelope.federation == nil {
		return "", false
	}

	return r.reqEnvelope.federation.ReplyTo, true
}

// FederationRequestID retrieves the federation specific requestid
func (r *request) FederationRequestID() (id string, federated bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.reqEnvelope.federation == nil {
		return "", false
	}

	return r.reqEnvelope.federation.RequestID, true
}

// SetRequestID sets the request ID for this message
func (r *request) SetRequestID(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.reqEnvelope.RequestID = id
}

// SetFederationTargets sets the list of hosts this message should go to.
//
// Federation brokers will duplicate the message and send one for each target
func (r *request) SetFederationTargets(targets []string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.reqEnvelope.federation == nil {
		r.reqEnvelope.federation = &federationTransportHeader{}
	}

	r.reqEnvelope.federation.Targets = targets
}

// SetFederationReplyTo stores the original reply-to destination in the federation headers
func (r *request) SetFederationReplyTo(reply string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.reqEnvelope.federation == nil {
		r.reqEnvelope.federation = &federationTransportHeader{}
	}

	r.reqEnvelope.federation.ReplyTo = reply
}

// SetFederationRequestID sets the request ID for federation purposes
func (r *request) SetFederationRequestID(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.reqEnvelope.federation == nil {
		r.reqEnvelope.federation = &federationTransportHeader{}
	}

	r.reqEnvelope.federation.RequestID = id
}

// IsFederated determines if this message is federated
func (r *request) IsFederated() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.reqEnvelope.federation != nil
}

// SetUnfederated removes any federation information from the message
func (r *request) SetUnfederated() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.reqEnvelope.federation = nil
}

// SetMessage set the message body that's contained in this request
func (r *request) SetMessage(message string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.MessageBody = []byte(message)
}

// SetCallerID sets the caller id for this request
func (r *request) SetCallerID(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// TODO validate it
	r.reqEnvelope.CallerID = id
}

// SetCollective sets the collective this request is directed at
func (r *request) SetCollective(collective string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.reqEnvelope.Collective = collective
}

// SetAgent sets the agent this requires is created for
func (r *request) SetAgent(agent string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.reqEnvelope.Agent = agent
}

// SetTTL sets the validity period for this message
func (r *request) SetTTL(ttl int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.reqEnvelope.TTL = ttl
}

// Message retrieves the Message body
func (r *request) Message() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return string(r.MessageBody)
}

// RequestID retrieves the unique request ID
func (r *request) RequestID() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.reqEnvelope.RequestID
}

// SenderID retrieves the sender id that sent the message
func (r *request) SenderID() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.reqEnvelope.SenderID
}

// CallerID retrieves the caller id that sent the message
func (r *request) CallerID() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.reqEnvelope.CallerID
}

// Collective retrieves the name of the sub collective this message is aimed at
func (r *request) Collective() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.reqEnvelope.Collective
}

// Agent retrieves the agent name this message is for
func (r *request) Agent() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.reqEnvelope.Agent
}

// TTL retrieves the maximum allow lifetime of this message
func (r *request) TTL() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.reqEnvelope.TTL
}

// Time retrieves the time this message was first made
func (r *request) Time() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()

	return time.Unix(r.reqEnvelope.Time, 0)
}

// Filter retrieves the filter for the message.  The boolean is true when the filter is not empty
func (r *request) Filter() (filter *protocol.Filter, filtered bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.reqEnvelope.Filter == nil {
		r.reqEnvelope.Filter = protocol.NewFilter()
	}

	return r.reqEnvelope.Filter, !r.reqEnvelope.Filter.Empty()
}

// NewFilter creates a new empty filter and sets it
func (r *request) NewFilter() *protocol.Filter {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.reqEnvelope.Filter = protocol.NewFilter()

	return r.reqEnvelope.Filter
}

// JSON creates a JSON encoded request
func (r *request) JSON() (body string, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	j, err := json.Marshal(r)
	if err != nil {
		protocolErrorCtr.Inc()
		return "", fmt.Errorf("could not JSON Marshal: %s", err)
	}

	body = string(j)

	if err = r.isValidJSONUnlocked(body); err != nil {
		return "", fmt.Errorf("serialized JSON produced from the Request does not pass validation: %s", err)
	}

	return body, nil
}

// SetFilter sets and overwrites the filter for a message with a new one
func (r *request) SetFilter(filter *protocol.Filter) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.reqEnvelope.Filter = filter
}

// Version retrieves the protocol version for this message
func (r *request) Version() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.Protocol
}

// IsValidJSON validates the given JSON data against the schema
func (r *request) IsValidJSON(data string) (err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.isValidJSONUnlocked(data)
}

func (r *request) isValidJSONUnlocked(data string) error {
	// TODO

	// _, errors, err := schemaValidate(requestSchema, data)
	// if err != nil {
	// 	return fmt.Errorf("could not validate Request JSON data: %s", err)
	// }
	//
	// if len(errors) != 0 {
	// 	return fmt.Errorf("supplied JSON document is not a valid Request message: %s", strings.Join(errors, ", "))
	// }

	return nil
}
